package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	orchestratorv1 "comfy-service-tests/internal/proto/orchestratorv1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type jobResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type statusResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type gateway struct {
	mu            sync.Mutex
	client        orchestratorv1.OrchestratorClient
	lastJobID     string
	artifactsRoot string
}

func main() {
	addr := envOrDefault("GATEWAY_ADDR", ":8084")
	orchestratorAddr := envOrDefault("ORCHESTRATOR_ADDR", "orchestrator:9090")
	artifactsRoot := envOrDefault("ARTIFACTS_ROOT", "/artifacts")

	conn, err := grpc.Dial(orchestratorAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial orchestrator at %s: %v", orchestratorAddr, err)
	}

	g := &gateway{
		client:        orchestratorv1.NewOrchestratorClient(conn),
		artifactsRoot: artifactsRoot,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/nodes", g.handleNodes)
	mux.HandleFunc("/v1/workflows", g.handleWorkflows)
	mux.HandleFunc("/v1/jobs/", g.handleJob)
	mux.HandleFunc("/v1/jobs", g.handleJobIndex)
	mux.HandleFunc("/v1/events", g.handleEvents)

	log.Printf("gateway listening on %s", addr)
	if err := http.ListenAndServe(addr, logRequests(withCORS(mux))); err != nil {
		log.Fatalf("gateway stopped: %v", err)
	}
}

func (g *gateway) handleNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp, err := g.client.ListNodes(ctx, &orchestratorv1.ListNodesRequest{})
	if err != nil {
		http.Error(w, "failed to load node catalog", http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (g *gateway) handleWorkflows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20+1))
	if err != nil {
		http.Error(w, "failed to read workflow", http.StatusBadRequest)
		return
	}
	if int64(len(payload)) > 1<<20 {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	execResp, err := g.client.ExecuteWorkflow(ctx, &orchestratorv1.ExecuteWorkflowRequest{
		Graph: &orchestratorv1.WorkflowGraph{
			Format:       "comfyui",
			WorkflowJson: string(payload),
		},
	})
	if err != nil {
		http.Error(w, "failed to submit workflow", http.StatusBadGateway)
		return
	}

	g.mu.Lock()
	g.lastJobID = execResp.WorkflowId
	g.mu.Unlock()

	writeJSON(w, http.StatusAccepted, jobResponse{JobID: execResp.WorkflowId, Status: "queued"})
}

func (g *gateway) handleJobIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	g.mu.Lock()
	jobID := g.lastJobID
	g.mu.Unlock()

	if jobID == "" {
		writeJSON(w, http.StatusOK, statusResponse{ID: "", Status: "idle"})
		return
	}

	g.fetchStatus(w, r, jobID)
}

func (g *gateway) handleJob(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/output") {
		g.handleJobOutput(w, r)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/v1/jobs/")
	if id == "" {
		http.Error(w, "missing job id", http.StatusBadRequest)
		return
	}

	g.fetchStatus(w, r, id)
}

func (g *gateway) fetchStatus(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp, err := g.client.GetWorkflowStatus(ctx, &orchestratorv1.StatusRequest{WorkflowId: id})
	if err != nil {
		http.Error(w, "failed to get status", http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, statusResponse{ID: resp.WorkflowId, Status: resp.State, Detail: resp.Message})
}

func (g *gateway) handleJobOutput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/jobs/"), "/output")
	if id == "" {
		http.Error(w, "missing job id", http.StatusBadRequest)
		return
	}

	outputPath := filepath.Join(g.artifactsRoot, id, "output.png")
	if _, err := os.Stat(outputPath); err != nil {
		http.Error(w, "output not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, outputPath)
}

func (g *gateway) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		g.mu.Lock()
		jobID = g.lastJobID
		g.mu.Unlock()
	}

	if jobID == "" {
		http.Error(w, "no active job", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	stream, err := g.client.StreamStatus(ctx, &orchestratorv1.StatusRequest{WorkflowId: jobID})
	if err != nil {
		http.Error(w, "failed to stream events", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	for {
		event, err := stream.Recv()
		if err != nil {
			return
		}
		payload, _ := json.Marshal(event)
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(payload)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
