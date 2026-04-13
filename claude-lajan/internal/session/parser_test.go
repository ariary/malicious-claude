package session

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestParse(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	fixture := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "sample.jsonl")

	s, err := Parse(fixture)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if s.ID != "test-session-123" {
		t.Errorf("ID = %q, want %q", s.ID, "test-session-123")
	}
	if s.CWD != "/tmp/myproject" {
		t.Errorf("CWD = %q, want %q", s.CWD, "/tmp/myproject")
	}
	if s.AssistantTurns < 1 {
		t.Errorf("AssistantTurns = %d, want >= 1", s.AssistantTurns)
	}
	if len(s.FilesTouched) == 0 {
		t.Error("expected at least one file touched")
	}
	if len(s.BashCommands) == 0 {
		t.Error("expected at least one bash command")
	}
	if s.BashCommands[0] != "go build ./..." {
		t.Errorf("BashCommands[0] = %q, want %q", s.BashCommands[0], "go build ./...")
	}

	// Metrics assertions
	if s.Metrics.TotalInputTokens != 100 {
		t.Errorf("TotalInputTokens = %d, want 100", s.Metrics.TotalInputTokens)
	}
	if s.Metrics.ToolCallsFailed != 1 {
		t.Errorf("ToolCallsFailed = %d, want 1", s.Metrics.ToolCallsFailed)
	}
	if s.Metrics.EstimatedCostUSD <= 0 {
		t.Error("EstimatedCostUSD should be > 0")
	}
	if s.Metrics.Duration <= 0 {
		t.Error("Duration should be > 0")
	}
}
