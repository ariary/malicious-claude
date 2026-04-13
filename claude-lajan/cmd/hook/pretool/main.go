// pretool-hook is the Claude Code PreToolUse hook binary.
// Before risky tool calls (Bash, Edit, Write, MultiEdit), it outputs improvement
// learnings from the digest to stderr. Claude Code surfaces stderr as a reminder
// before the tool executes — no blocking, no API call, just context injection.
//
// Claude Code PreToolUse hook input (stdin):
//
//	{
//	  "hook_event_name": "PreToolUse",
//	  "session_id": "...",
//	  "transcript_path": "...",
//	  "tool_name": "Bash",
//	  "tool_input": { "command": "..." }
//	}
//
// Output: exit 0 to approve. Stderr is shown as a message in the UI.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ariary/claude-lajan/internal/config"
	"github.com/ariary/claude-lajan/internal/debate"
	"github.com/ariary/claude-lajan/internal/report"
)

// riskyTools are the tools where a pre-execution reminder is most valuable.
var riskyTools = map[string]bool{
	"Bash":      true,
	"Edit":      true,
	"Write":     true,
	"MultiEdit": true,
}

type pretoolInput struct {
	HookEventName string          `json:"hook_event_name"`
	SessionID     string          `json:"session_id"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
}

func main() {
	if !config.IsHookEnabled("pretool_inject") {
		os.Exit(0)
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(data) == 0 {
		os.Exit(0)
	}

	var input pretoolInput
	if err := json.Unmarshal(data, &input); err != nil {
		os.Exit(0)
	}

	if !riskyTools[input.ToolName] {
		os.Exit(0)
	}

	findings := report.LoadTopN(config.DigestInjectTop)
	improvements := filterImprovements(findings)
	if len(improvements) == 0 {
		os.Exit(0)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "[claude-lajan] Reminders before %s:\n", input.ToolName)
	for _, f := range improvements {
		fmt.Fprintf(&sb, "  • %s\n", f.Text)
	}
	fmt.Fprint(os.Stderr, sb.String())

	os.Exit(0)
}

func filterImprovements(findings []debate.Finding) []debate.Finding {
	var out []debate.Finding
	for _, f := range findings {
		if f.Type == debate.Improvement {
			out = append(out, f)
		}
	}
	return out
}
