package orchestrator

import (
	"encoding/json"
	"testing"

	orchestratorv1 "comfy-service-tests/internal/proto/orchestratorv1"
)

type testGraph struct {
	Nodes []testNode `json:"nodes"`
}

type testNode struct {
	Type          string `json:"type"`
	WidgetsValues []any  `json:"widgets_values"`
}

func TestParseWorkflowDefaults(t *testing.T) {
	spec := parseWorkflow(nil)
	if spec.Width != 512 || spec.Height != 512 {
		t.Fatalf("expected default size 512x512, got %dx%d", spec.Width, spec.Height)
	}
	if spec.Steps != 20 || spec.Cfg != 8 {
		t.Fatalf("unexpected defaults: steps=%d cfg=%v", spec.Steps, spec.Cfg)
	}
	if spec.Sampler != "euler" || spec.Scheduler != "normal" {
		t.Fatalf("unexpected defaults: sampler=%s scheduler=%s", spec.Sampler, spec.Scheduler)
	}
}

func TestParseWorkflowFromGraph(t *testing.T) {
	graph := testGraph{
		Nodes: []testNode{
			{Type: "CheckpointLoaderSimple", WidgetsValues: []any{"model.safetensors"}},
			{Type: "CLIPTextEncode", WidgetsValues: []any{"positive"}},
			{Type: "CLIPTextEncode", WidgetsValues: []any{"negative"}},
			{Type: "EmptyLatentImage", WidgetsValues: []any{640.0, 384.0}},
			{Type: "KSampler", WidgetsValues: []any{1234.0, 30.0, 6.5, "euler_a", "karras"}},
		},
	}

	payload, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}

	req := &orchestratorv1.ExecuteWorkflowRequest{
		Graph: &orchestratorv1.WorkflowGraph{WorkflowJson: string(payload)},
	}

	spec := parseWorkflow(req)
	if spec.Checkpoint != "model.safetensors" {
		t.Fatalf("unexpected checkpoint: %s", spec.Checkpoint)
	}
	if spec.Positive != "positive" || spec.Negative != "negative" {
		t.Fatalf("unexpected prompts: %s / %s", spec.Positive, spec.Negative)
	}
	if spec.Width != 640 || spec.Height != 384 {
		t.Fatalf("unexpected size: %dx%d", spec.Width, spec.Height)
	}
	if spec.Seed != 1234 || spec.Steps != 30 || spec.Cfg != 6.5 {
		t.Fatalf("unexpected sampler params: seed=%d steps=%d cfg=%v", spec.Seed, spec.Steps, spec.Cfg)
	}
	if spec.Sampler != "euler_a" || spec.Scheduler != "karras" {
		t.Fatalf("unexpected scheduler: %s %s", spec.Sampler, spec.Scheduler)
	}
}

func TestParseWorkflowInvalidJSON(t *testing.T) {
	req := &orchestratorv1.ExecuteWorkflowRequest{
		Graph: &orchestratorv1.WorkflowGraph{WorkflowJson: "{not-json"},
	}
	spec := parseWorkflow(req)
	if spec.Width != 512 || spec.Height != 512 {
		t.Fatalf("expected defaults on invalid JSON")
	}
}

func TestValueHelpers(t *testing.T) {
	values := []any{"text", 12.0, int64(9), 3, 4.5}

	if got := stringValue(values, 0); got != "text" {
		t.Fatalf("stringValue unexpected: %s", got)
	}
	if got := stringValue(values, 2); got != "" {
		t.Fatalf("stringValue should ignore non-string: %s", got)
	}
	if got := intValue(values, 1, 0); got != 12 {
		t.Fatalf("intValue float64 unexpected: %d", got)
	}
	if got := intValue(values, 0, 7); got != 7 {
		t.Fatalf("intValue fallback unexpected: %d", got)
	}
	if got := int64Value(values, 2, 0); got != 9 {
		t.Fatalf("int64Value int64 unexpected: %d", got)
	}
	if got := int64Value(values, 3, 0); got != 3 {
		t.Fatalf("int64Value int unexpected: %d", got)
	}
	if got := int64Value(values, 10, 7); got != 7 {
		t.Fatalf("int64Value fallback unexpected: %d", got)
	}
	if got := floatValue(values, 4, 0); got != 4.5 {
		t.Fatalf("floatValue float unexpected: %v", got)
	}
	if got := floatValue(values, 0, 2.5); got != 2.5 {
		t.Fatalf("floatValue fallback unexpected: %v", got)
	}
}
