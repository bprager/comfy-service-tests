package orchestrator

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	orchestratorv1 "comfy-service-tests/internal/proto/orchestratorv1"
)

type Job struct {
	ID        string
	State     string
	Message   string
	Progress  float64
	OutputURI string
	UpdatedAt time.Time
}

type Server struct {
	orchestratorv1.UnimplementedOrchestratorServer
	mu            sync.Mutex
	jobs          map[string]*Job
	stageClient   orchestratorv1.StageRunnerClient
	artifactsRoot string
}

func NewServer(stageClient orchestratorv1.StageRunnerClient, artifactsRoot string) *Server {
	return &Server{
		jobs:          make(map[string]*Job),
		stageClient:   stageClient,
		artifactsRoot: artifactsRoot,
	}
}

func (s *Server) ExecuteWorkflow(ctx context.Context, req *orchestratorv1.ExecuteWorkflowRequest) (*orchestratorv1.ExecuteWorkflowResponse, error) {
	jobID := fmt.Sprintf("wf-%d", time.Now().UnixNano())
	job := &Job{ID: jobID, State: "queued", UpdatedAt: time.Now()}

	s.mu.Lock()
	s.jobs[jobID] = job
	s.mu.Unlock()

	go s.runJob(jobID, req)

	return &orchestratorv1.ExecuteWorkflowResponse{WorkflowId: jobID}, nil
}

func (s *Server) GetWorkflowStatus(ctx context.Context, req *orchestratorv1.StatusRequest) (*orchestratorv1.StatusResponse, error) {
	job := s.getJob(req.WorkflowId)
	if job == nil {
		return &orchestratorv1.StatusResponse{WorkflowId: req.WorkflowId, State: "unknown", Message: "not found"}, nil
	}
	return &orchestratorv1.StatusResponse{WorkflowId: job.ID, State: job.State, Message: job.Message}, nil
}

func (s *Server) StreamStatus(req *orchestratorv1.StatusRequest, stream orchestratorv1.Orchestrator_StreamStatusServer) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		job := s.getJob(req.WorkflowId)
		if job == nil {
			return nil
		}

		event := &orchestratorv1.StatusEvent{
			WorkflowId: job.ID,
			State:      job.State,
			Message:    job.Message,
			Progress:   job.Progress,
		}

		if err := stream.Send(event); err != nil {
			return err
		}

		if job.State == "completed" || job.State == "failed" {
			return nil
		}

		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
		}
	}
}

func (s *Server) ListNodes(ctx context.Context, req *orchestratorv1.ListNodesRequest) (*orchestratorv1.ListNodesResponse, error) {
	nodes := []*orchestratorv1.NodeDefinition{
		{
			Name:     "CheckpointLoaderSimple",
			Category: "loaders",
			Outputs: map[string]string{
				"MODEL": "MODEL",
				"CLIP":  "CLIP",
				"VAE":   "VAE",
			},
		},
		{
			Name:     "CLIPTextEncode",
			Category: "conditioning",
			Inputs: map[string]string{
				"clip": "CLIP",
			},
			Outputs: map[string]string{
				"CONDITIONING": "CONDITIONING",
			},
		},
		{
			Name:     "EmptyLatentImage",
			Category: "latent",
			Outputs: map[string]string{
				"LATENT": "LATENT",
			},
		},
		{
			Name:     "KSampler",
			Category: "sampling",
			Inputs: map[string]string{
				"model":        "MODEL",
				"positive":     "CONDITIONING",
				"negative":     "CONDITIONING",
				"latent_image": "LATENT",
			},
			Outputs: map[string]string{
				"LATENT": "LATENT",
			},
		},
		{
			Name:     "VAEDecode",
			Category: "latent",
			Inputs: map[string]string{
				"samples": "LATENT",
				"vae":     "VAE",
			},
			Outputs: map[string]string{
				"IMAGE": "IMAGE",
			},
		},
		{
			Name:     "SaveImage",
			Category: "image",
			Inputs: map[string]string{
				"images": "IMAGE",
			},
		},
	}

	return &orchestratorv1.ListNodesResponse{Nodes: nodes}, nil
}

func (s *Server) runJob(jobID string, req *orchestratorv1.ExecuteWorkflowRequest) {
	spec := parseWorkflow(req)
	s.updateJob(jobID, "running", "dispatched", 0.1)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	params := map[string]string{
		"checkpoint": spec.Checkpoint,
		"positive":   spec.Positive,
		"negative":   spec.Negative,
		"width":      strconv.Itoa(spec.Width),
		"height":     strconv.Itoa(spec.Height),
		"seed":       strconv.FormatInt(spec.Seed, 10),
		"steps":      strconv.Itoa(spec.Steps),
		"cfg":        fmt.Sprintf("%.2f", spec.Cfg),
		"sampler":    spec.Sampler,
		"scheduler":  spec.Scheduler,
	}

	stageReq := &orchestratorv1.StageRequest{
		StageId:  jobID,
		NodeType: "text_to_image",
		Params:   params,
	}

	stageResp, err := s.stageClient.RunStage(ctx, stageReq)
	if err != nil {
		s.updateJob(jobID, "failed", err.Error(), 1)
		return
	}

	output := stageResp.OutputRefs["image"]
	outputURI := ""
	if output != nil {
		outputURI = output.Uri
	}

	s.mu.Lock()
	job := s.jobs[jobID]
	if job != nil {
		job.State = "completed"
		job.Message = outputURI
		job.Progress = 1
		job.OutputURI = outputURI
		job.UpdatedAt = time.Now()
	}
	s.mu.Unlock()
}

func (s *Server) updateJob(jobID, state, message string, progress float64) {
	s.mu.Lock()
	job := s.jobs[jobID]
	if job != nil {
		job.State = state
		job.Message = message
		job.Progress = progress
		job.UpdatedAt = time.Now()
	}
	s.mu.Unlock()
}

func (s *Server) getJob(id string) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[id]
}
