package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"comfy-service-tests/internal/imaging"
	"comfy-service-tests/internal/logging"
	orchestratorv1 "comfy-service-tests/internal/proto/orchestratorv1"

	"google.golang.org/grpc"
)

type stageServer struct {
	orchestratorv1.UnimplementedStageRunnerServer
	artifactsRoot string
}

func main() {
	addr := flag.String("addr", ":9091", "gRPC listen address")
	artifactsRoot := flag.String("artifacts", envOrDefault("ARTIFACTS_ROOT", "/artifacts"), "artifacts root directory")
	flag.Parse()
	logDir := envOrDefault("LOG_DIR", ".log")

	if _, err := logging.Setup("stage-sampler", logDir); err != nil {
		log.Fatalf("failed to set up logging: %v", err)
	}

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", *addr, err)
	}

	server := grpc.NewServer()
	orchestratorv1.RegisterStageRunnerServer(server, &stageServer{artifactsRoot: *artifactsRoot})

	log.Printf("stage-sampler gRPC listening on %s", *addr)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("stage-sampler gRPC stopped: %v", err)
	}
}

func (s *stageServer) RunStage(ctx context.Context, req *orchestratorv1.StageRequest) (*orchestratorv1.StageResult, error) {
	width := parseInt(req.Params["width"], 512)
	height := parseInt(req.Params["height"], 512)
	seed := parseInt64(req.Params["seed"], 0)

	payload, err := imaging.RenderPlaceholder(imaging.RenderOptions{
		Width:      width,
		Height:     height,
		Prompt:     req.Params["positive"],
		Negative:   req.Params["negative"],
		Checkpoint: req.Params["checkpoint"],
		Seed:       seed,
	})
	if err != nil {
		return &orchestratorv1.StageResult{StageId: req.StageId, Status: "failed", ErrorMessage: err.Error()}, nil
	}

	outputDir := filepath.Join(s.artifactsRoot, req.StageId)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return &orchestratorv1.StageResult{StageId: req.StageId, Status: "failed", ErrorMessage: err.Error()}, nil
	}

	outputPath := filepath.Join(outputDir, "output.png")
	if err := os.WriteFile(outputPath, payload, 0o644); err != nil {
		return &orchestratorv1.StageResult{StageId: req.StageId, Status: "failed", ErrorMessage: err.Error()}, nil
	}

	outputRef := &orchestratorv1.TensorRef{
		Uri:   outputPath,
		Shape: []int64{int64(height), int64(width), 3},
		Dtype: "image/png",
	}

	return &orchestratorv1.StageResult{
		StageId: req.StageId,
		Status:  "completed",
		OutputRefs: map[string]*orchestratorv1.TensorRef{
			"image": outputRef,
		},
	}, nil
}

func (s *stageServer) Health(ctx context.Context, req *orchestratorv1.HealthRequest) (*orchestratorv1.HealthResponse, error) {
	return &orchestratorv1.HealthResponse{Status: "ok"}, nil
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseInt64(value string, fallback int64) int64 {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
