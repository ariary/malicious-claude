package debate

import (
	"testing"
)

func TestParseConsensus_reached(t *testing.T) {
	raw := `{"status":"CONSENSUS_REACHED","rationale":"clear findings","findings":[{"type":"improvement","scope":"global","text":"verify before claiming done"}]}`
	out, err := parseConsensus(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != StatusReached {
		t.Errorf("status = %q, want CONSENSUS_REACHED", out.Status)
	}
	if len(out.Findings) != 1 {
		t.Errorf("findings count = %d, want 1", len(out.Findings))
	}
	if out.Findings[0].Scope != ScopeGlobal {
		t.Errorf("scope = %q, want global", out.Findings[0].Scope)
	}
}

func TestParseConsensus_continue(t *testing.T) {
	raw := `{"status":"CONTINUE","rationale":"needs more debate","findings":[]}`
	out, err := parseConsensus(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != StatusContinue {
		t.Errorf("status = %q, want CONTINUE", out.Status)
	}
}

func TestParseConsensus_fenced(t *testing.T) {
	raw := "```json\n{\"status\":\"CONSENSUS_REACHED\",\"rationale\":\"ok\",\"findings\":[]}\n```"
	out, err := parseConsensus(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != StatusReached {
		t.Errorf("status = %q, want CONSENSUS_REACHED", out.Status)
	}
}

func TestParseConsensus_badJSON(t *testing.T) {
	out, _ := parseConsensus("not json at all")
	if out.Status != StatusContinue {
		t.Errorf("bad JSON should fall back to CONTINUE, got %q", out.Status)
	}
}
