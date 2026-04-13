// prompt-hook is the Claude Code UserPromptSubmit hook binary.
// It reads the current digest and prepends top learnings to the user's prompt.
//
// Claude Code UserPromptSubmit hook input (stdin):
//
//	{
//	  "hook_event_name": "UserPromptSubmit",
//	  "session_id": "...",
//	  "transcript_path": "...",
//	  "prompt": "the user's prompt text"
//	}
//
// Output (stdout) to modify the prompt:
//
//	{"value": "modified prompt text"}
//
// Exit 0 with no output = pass prompt through unchanged.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ariary/claude-lajan/internal/config"
	"github.com/ariary/claude-lajan/internal/report"
)

type promptInput struct {
	HookEventName  string `json:"hook_event_name"`
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Prompt         string `json:"prompt"`
}

type promptOutput struct {
	Value string `json:"value"`
}

func main() {
	if !config.IsHookEnabled("prompt_inject") {
		os.Exit(0)
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(0) // on error, pass through unchanged
	}

	var input promptInput
	if err := json.Unmarshal(data, &input); err != nil {
		os.Exit(0)
	}

	cfg, _ := config.LoadUserConfig()
	findings := report.LoadTopN(cfg.DigestInjectTop)
	if len(findings) == 0 {
		os.Exit(0) // nothing to inject
	}

	var sb strings.Builder
	sb.WriteString("<session-learnings>\n")
	sb.WriteString("Previous session analysis identified these key learnings:\n")
	for _, f := range findings {
		fmt.Fprintf(&sb, "- [%s/%s] %s\n", f.Type, f.Scope, f.Text)
	}
	sb.WriteString("</session-learnings>\n\n")
	sb.WriteString(input.Prompt)

	out := promptOutput{Value: sb.String()}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		os.Exit(0)
	}
}
