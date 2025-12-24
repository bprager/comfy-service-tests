package main

import (
	"flag"
	"log"
	"net"
	"os"

	"comfy-service-tests/internal/orchestrator"
	orchestratorv1 "comfy-service-tests/internal/proto/orchestratorv1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", ":9090", "gRPC listen address")
	stageAddr := flag.String("stage-addr", envOrDefault("STAGE_SAMPLER_ADDR", "stage-sampler:9091"), "stage sampler address")
	artifactsRoot := flag.String("artifacts", envOrDefault("ARTIFACTS_ROOT", "/artifacts"), "artifacts root directory")
	flag.Parse()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", *addr, err)
	}

	conn, err := grpc.Dial(*stageAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial stage sampler at %s: %v", *stageAddr, err)
	}

	stageClient := orchestratorv1.NewStageRunnerClient(conn)
	server := grpc.NewServer()
	orchestratorv1.RegisterOrchestratorServer(server, orchestrator.NewServer(stageClient, *artifactsRoot))

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
