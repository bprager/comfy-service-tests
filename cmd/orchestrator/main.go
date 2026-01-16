package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"comfy-service-tests/internal/logging"
	"comfy-service-tests/internal/orchestrator"
	orchestratorv1 "comfy-service-tests/internal/proto/orchestratorv1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", ":9090", "gRPC listen address")
	stageAddr := flag.String("stage-addr", envOrDefault("STAGE_SAMPLER_ADDR", "stage-sampler:9091"), "stage sampler address")
	artifactsRoot := flag.String("artifacts", envOrDefault("ARTIFACTS_ROOT", "/artifacts"), "artifacts root directory")
	logDir := flag.String("log-dir", envOrDefault("LOG_DIR", "/logs"), "log directory")
	stageTimeout := flag.Duration("stage-timeout", envDurationOrDefault("STAGE_TIMEOUT", 2*time.Minute), "stage execution timeout")
	stageRetries := flag.Int("stage-retries", envIntOrDefault("STAGE_MAX_RETRIES", 2), "stage execution retry count")
	stageRetryDelay := flag.Duration("stage-retry-delay", envDurationOrDefault("STAGE_RETRY_DELAY", 2*time.Second), "delay between stage retries")
	stageHealthTimeout := flag.Duration("stage-health-timeout", envDurationOrDefault("STAGE_HEALTH_TIMEOUT", 2*time.Minute), "max time to wait for stage health")
	stageHealthInterval := flag.Duration("stage-health-interval", envDurationOrDefault("STAGE_HEALTH_INTERVAL", 2*time.Second), "interval between stage health checks")
	stageHealthRequestTimeout := flag.Duration("stage-health-request-timeout", envDurationOrDefault("STAGE_HEALTH_REQUEST_TIMEOUT", 5*time.Second), "timeout per stage health request")
	flag.Parse()

	if _, err := logging.Setup("orchestrator", *logDir); err != nil {
		log.Fatalf("failed to set up logging: %v", err)
	}
	log.Printf("starting orchestrator addr=%s stage=%s artifacts=%s", *addr, *stageAddr, *artifactsRoot)

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", *addr, err)
	}

	conn, err := grpc.Dial(*stageAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial stage sampler at %s: %v", *stageAddr, err)
	}

	stageClient := orchestratorv1.NewStageRunnerClient(conn)
	if err := waitForStageHealth(stageClient, *stageHealthTimeout, *stageHealthInterval, *stageHealthRequestTimeout); err != nil {
		log.Fatalf("stage sampler health check failed: %v", err)
	}

	server := grpc.NewServer()
	orchestratorv1.RegisterOrchestratorServer(
		server,
		orchestrator.NewServer(stageClient, *artifactsRoot, *stageTimeout, *stageRetries, *stageRetryDelay),
	)

	log.Printf("orchestrator gRPC listening on %s", *addr)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("orchestrator gRPC stopped: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func waitForStageHealth(client orchestratorv1.StageRunnerClient, timeout, interval, requestTimeout time.Duration) error {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if requestTimeout <= 0 {
		requestTimeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Printf("waiting for stage sampler health (timeout=%s)", timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastErr error
	for {
		reqCtx, reqCancel := context.WithTimeout(ctx, requestTimeout)
		_, err := client.Health(reqCtx, &orchestratorv1.HealthRequest{})
		reqCancel()
		if err == nil {
			log.Printf("stage sampler is healthy")
			return nil
		}
		lastErr = err
		log.Printf("stage sampler health check failed: %v", err)
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("last error: %w", lastErr)
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
