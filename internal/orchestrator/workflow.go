package orchestrator

import (
	"encoding/json"

	orchestratorv1 "comfy-service-tests/internal/proto/orchestratorv1"
)

type workflowGraph struct {
	Nodes []workflowNode `json:"nodes"`
}

type workflowNode struct {
	ID            int            `json:"id"`
	Type          string         `json:"type"`
	WidgetsValues []any          `json:"widgets_values"`
	Properties    map[string]any `json:"properties"`
}

type workflowSpec struct {
	Checkpoint string
	Positive   string
	Negative   string
	Width      int
	Height     int
	Seed       int64
	Steps      int
	Cfg        float64
	Sampler    string
	Scheduler  string
}

func parseWorkflow(req *orchestratorv1.ExecuteWorkflowRequest) workflowSpec {
	spec := workflowSpec{
		Width:     512,
		Height:    512,
		Steps:     20,
		Cfg:       8,
		Sampler:   "euler",
		Scheduler: "normal",
	}

	if req == nil || req.Graph == nil {
		return spec
	}

	var graph workflowGraph
	if err := json.Unmarshal([]byte(req.Graph.WorkflowJson), &graph); err != nil {
		return spec
	}

	prompts := []string{}
	for _, node := range graph.Nodes {
		switch node.Type {
		case "CheckpointLoaderSimple", "LoadCheckpoint":
			spec.Checkpoint = stringValue(node.WidgetsValues, 0)
		case "CLIPTextEncode", "CLIPTextEncodePrompt":
			if text := stringValue(node.WidgetsValues, 0); text != "" {
				prompts = append(prompts, text)
			}
		case "EmptyLatentImage":
			spec.Width = intValue(node.WidgetsValues, 0, spec.Width)
			spec.Height = intValue(node.WidgetsValues, 1, spec.Height)
		case "KSampler":
			spec.Seed = int64Value(node.WidgetsValues, 0, spec.Seed)
			spec.Steps = intValue(node.WidgetsValues, 1, spec.Steps)
			spec.Cfg = floatValue(node.WidgetsValues, 2, spec.Cfg)
			spec.Sampler = stringValue(node.WidgetsValues, 3)
			spec.Scheduler = stringValue(node.WidgetsValues, 4)
		}
	}

	if len(prompts) > 0 {
		spec.Positive = prompts[0]
	}
	if len(prompts) > 1 {
		spec.Negative = prompts[1]
	}

	return spec
}

func stringValue(values []any, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	switch v := values[index].(type) {
	case string:
		return v
	default:
		return ""
	}
}

func intValue(values []any, index int, fallback int) int {
	if index < 0 || index >= len(values) {
		return fallback
	}
	switch v := values[index].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return fallback
	}
}

func int64Value(values []any, index int, fallback int64) int64 {
	if index < 0 || index >= len(values) {
		return fallback
	}
	switch v := values[index].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return fallback
	}
}

func floatValue(values []any, index int, fallback float64) float64 {
	if index < 0 || index >= len(values) {
		return fallback
	}
	switch v := values[index].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return fallback
	}
}
