package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sort"
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
	NodeStates map[int64]*orchestratorv1.NodeState
}

type Server struct {
	orchestratorv1.UnimplementedOrchestratorServer
	mu            sync.Mutex
	jobs          map[string]*Job
	stageClient   orchestratorv1.StageRunnerClient
	artifactsRoot string
	stageTimeout  time.Duration
}

func NewServer(stageClient orchestratorv1.StageRunnerClient, artifactsRoot string, stageTimeout time.Duration) *Server {
	if stageTimeout <= 0 {
		stageTimeout = 2 * time.Minute
	}
	return &Server{
		jobs:          make(map[string]*Job),
		stageClient:   stageClient,
		artifactsRoot: artifactsRoot,
		stageTimeout:  stageTimeout,
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
			Nodes:      cloneNodeStates(job.NodeStates),
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
	s.initNodeStates(jobID, req)

	ctx, cancel := context.WithTimeout(context.Background(), s.stageTimeout)
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

	nodes := parseWorkflowNodes(req)
	preNodes := nodeIDsForTypes(nodes, "CheckpointLoaderSimple", "LoadCheckpoint", "CLIPTextEncode", "CLIPTextEncodePrompt", "EmptyLatentImage")
	ksamplerNodes := nodeIDsForTypes(nodes, "KSampler")
	postNodes := nodeIDsForTypes(nodes, "VAEDecode", "SaveImage")

	s.updateNodeState(jobID, preNodes, "completed")
	s.updateNodeState(jobID, ksamplerNodes, "running")

	stageResp, err := s.stageClient.RunStage(ctx, stageReq)
	if err != nil {
		log.Printf("stage run failed job=%s err=%v", jobID, err)
		s.updateNodeState(jobID, ksamplerNodes, "failed")
		s.updateJob(jobID, "failed", err.Error(), 1)
		return
	}

	if stageResp == nil || stageResp.Status != "completed" {
		message := "stage failed"
		if stageResp != nil && stageResp.ErrorMessage != "" {
			message = stageResp.ErrorMessage
		}
		log.Printf("stage run failed job=%s status=%s err=%s", jobID, stageResp.GetStatus(), message)
		s.updateNodeState(jobID, ksamplerNodes, "failed")
		s.updateJob(jobID, "failed", message, 1)
		return
	}

	output := stageResp.OutputRefs["image"]
	outputURI := ""
	if output != nil {
		outputURI = output.Uri
	} else {
		log.Printf("stage response missing image output job=%s", jobID)
	}

	if outputURI == "" {
		s.updateNodeState(jobID, ksamplerNodes, "failed")
		s.updateJob(jobID, "failed", "stage returned no output", 1)
		return
	}

	s.updateNodeState(jobID, ksamplerNodes, "completed")
	s.updateNodeState(jobID, postNodes, "completed")

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

func (s *Server) initNodeStates(jobID string, req *orchestratorv1.ExecuteWorkflowRequest) {
	nodes := parseWorkflowNodes(req)
	if len(nodes) == 0 {
		return
	}

	stateMap := make(map[int64]*orchestratorv1.NodeState, len(nodes))
	for _, node := range nodes {
		id := int64(node.ID)
		stateMap[id] = &orchestratorv1.NodeState{
			NodeId:   id,
			NodeType: node.Type,
			State:    "queued",
		}
	}

	s.mu.Lock()
	job := s.jobs[jobID]
	if job != nil {
		job.NodeStates = stateMap
		job.UpdatedAt = time.Now()
	}
	s.mu.Unlock()
}

func (s *Server) updateNodeState(jobID string, nodeIDs []int64, state string) {
	if len(nodeIDs) == 0 {
		return
	}

	s.mu.Lock()
	job := s.jobs[jobID]
	if job == nil {
		s.mu.Unlock()
		return
	}
	if job.NodeStates == nil {
		job.NodeStates = make(map[int64]*orchestratorv1.NodeState)
	}
	for _, id := range nodeIDs {
		entry, ok := job.NodeStates[id]
		if !ok {
			entry = &orchestratorv1.NodeState{NodeId: id}
			job.NodeStates[id] = entry
		}
		entry.State = state
	}
	job.UpdatedAt = time.Now()
	s.mu.Unlock()
}

func nodeIDsForTypes(nodes []workflowNode, types ...string) []int64 {
	if len(nodes) == 0 || len(types) == 0 {
		return nil
	}

	typeSet := make(map[string]struct{}, len(types))
	for _, t := range types {
		typeSet[t] = struct{}{}
	}

	ids := make([]int64, 0, len(nodes))
	for _, node := range nodes {
		if _, ok := typeSet[node.Type]; ok {
			ids = append(ids, int64(node.ID))
		}
	}
	return ids
}

func cloneNodeStates(states map[int64]*orchestratorv1.NodeState) []*orchestratorv1.NodeState {
	if len(states) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	nodes := make([]*orchestratorv1.NodeState, 0, len(states))
	for _, id := range ids {
		state := states[id]
		if state == nil {
			continue
		}
		copy := *state
		nodes = append(nodes, &copy)
	}
	return nodes
}
