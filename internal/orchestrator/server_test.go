package orchestrator

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	orchestratorv1 "comfy-service-tests/internal/proto/orchestratorv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type fakeStageClient struct {
	mu   sync.Mutex
	req  *orchestratorv1.StageRequest
	resp *orchestratorv1.StageResult
	err  error
}

func (f *fakeStageClient) RunStage(ctx context.Context, req *orchestratorv1.StageRequest, _ ...grpc.CallOption) (*orchestratorv1.StageResult, error) {
	f.mu.Lock()
	f.req = req
	f.mu.Unlock()
	return f.resp, f.err
}

func (f *fakeStageClient) Health(ctx context.Context, _ *orchestratorv1.HealthRequest, _ ...grpc.CallOption) (*orchestratorv1.HealthResponse, error) {
	return &orchestratorv1.HealthResponse{Status: "ok"}, nil
}

func TestExecuteWorkflowUpdatesJob(t *testing.T) {
	fake := &fakeStageClient{resp: &orchestratorv1.StageResult{
		StageId: "job",
		Status:  "completed",
		OutputRefs: map[string]*orchestratorv1.TensorRef{
			"image": {Uri: "/tmp/output.png"},
		},
	}}
	server := NewServer(fake, "/artifacts", time.Second)

	graph := testGraph{Nodes: []testNode{{Type: "CLIPTextEncode", WidgetsValues: []any{"hello"}}}}
	payload, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}

	resp, err := server.ExecuteWorkflow(context.Background(), &orchestratorv1.ExecuteWorkflowRequest{
		Graph: &orchestratorv1.WorkflowGraph{WorkflowJson: string(payload)},
	})
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if resp.WorkflowId == "" {
		t.Fatalf("expected workflow id")
	}

	waitFor(t, time.Second, func() bool {
		status, _ := server.GetWorkflowStatus(context.Background(), &orchestratorv1.StatusRequest{WorkflowId: resp.WorkflowId})
		return status.State == "completed"
	})

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.req == nil {
		t.Fatalf("expected stage request")
	}
	if fake.req.NodeType != "text_to_image" {
		t.Fatalf("unexpected node type: %s", fake.req.NodeType)
	}
}

func TestGetWorkflowStatusUnknown(t *testing.T) {
	server := NewServer(&fakeStageClient{}, "/artifacts", time.Second)
	resp, err := server.GetWorkflowStatus(context.Background(), &orchestratorv1.StatusRequest{WorkflowId: "missing"})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if resp.State != "unknown" {
		t.Fatalf("expected unknown state, got %s", resp.State)
	}
}

func TestListNodes(t *testing.T) {
	server := NewServer(&fakeStageClient{}, "/artifacts", time.Second)
	resp, err := server.ListNodes(context.Background(), &orchestratorv1.ListNodesRequest{})
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(resp.Nodes) == 0 {
		t.Fatalf("expected nodes")
	}
}

func TestStreamStatusCompleted(t *testing.T) {
	server := NewServer(&fakeStageClient{}, "/artifacts", time.Second)
	server.jobs["job-1"] = &Job{ID: "job-1", State: "completed", Message: "done", Progress: 1}

	stream := &fakeStatusStream{ctx: context.Background()}
	err := server.StreamStatus(&orchestratorv1.StatusRequest{WorkflowId: "job-1"}, stream)
	if err != nil {
		t.Fatalf("stream status: %v", err)
	}
	if len(stream.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(stream.events))
	}
	if stream.events[0].State != "completed" {
		t.Fatalf("unexpected event state: %s", stream.events[0].State)
	}
}

func TestRunJobFailureUpdatesState(t *testing.T) {
	fake := &fakeStageClient{err: context.DeadlineExceeded}
	server := NewServer(fake, "/artifacts", time.Second)
	server.jobs["job-2"] = &Job{ID: "job-2", State: "queued"}

	server.runJob("job-2", &orchestratorv1.ExecuteWorkflowRequest{})

	status, err := server.GetWorkflowStatus(context.Background(), &orchestratorv1.StatusRequest{WorkflowId: "job-2"})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.State != "failed" {
		t.Fatalf("expected failed state, got %s", status.State)
	}
}

func waitFor(t *testing.T, timeout time.Duration, fn func() bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", timeout)
}

type fakeStatusStream struct {
	ctx    context.Context
	events []*orchestratorv1.StatusEvent
}

func (f *fakeStatusStream) Send(event *orchestratorv1.StatusEvent) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeStatusStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeStatusStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeStatusStream) SetTrailer(metadata.MD)       {}
func (f *fakeStatusStream) Context() context.Context     { return f.ctx }
func (f *fakeStatusStream) SendMsg(any) error            { return nil }
func (f *fakeStatusStream) RecvMsg(any) error            { return nil }
