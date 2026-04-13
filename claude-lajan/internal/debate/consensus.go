package debate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ariary/claude-lajan/internal/config"
)

const consensusSystem = `You are the Consensus agent in a multi-agent analysis of a Claude Code AI session.

You have read the Advocate's strengths, the Critic's improvement areas, and the Challenger's stress-test.
Your job is to synthesise a final, rigorous consensus.

## Quality bar — findings must clear ALL of these gates:
1. **Survived the Challenger's scrutiny** — no vague or unsupported claims.
2. **Session-specific** — directly observed in this session's transcript, not generic advice.
3. **Non-obvious** — not something any competent developer already knows (e.g. "validate before committing" does NOT qualify). A finding qualifies only if the session exhibited a specific, concrete failure or success that would surprise a senior engineer.
4. **Actionable in the next session** — the reader must be able to change one specific behaviour immediately. "Be more careful" does NOT qualify; "When grep returns empty after 3 attempts on a binary, switch to a minimal test hook (echo $VAR > /tmp/test)" does qualify.
5. **Not redundant** — if the theme is a variation of an existing known pattern (re-reading files, planning too long before coding, over-explaining), skip it unless this session showed an extreme or novel variant that adds new information.

## Hard limits:
- **At most 3 findings total** (strengths + improvements combined). Zero is acceptable.
- Prefer 1 strong finding over 3 weak ones.
- Only include a strength if it represents a genuinely exemplary technique, not just "did the correct thing."

## Decision rules:
- If you have ≥1 finding that clears the quality bar → CONSENSUS_REACHED.
- If findings are vague, contradictory, or need more scrutiny → CONTINUE with focused guidance.
- If nothing significant to report → CONSENSUS_REACHED with empty findings array.

## HOOK SUGGESTIONS (via hookify plugin)
When a finding is mechanical and fully automatable, include a suggested_hook.
Hooks are written as hookify rule files — no shell scripting required.

HOOK FIELDS:
  event   : "bash" | "file" | "prompt" | "stop"
  pattern : Python regex matched against the relevant field
  action  : "warn" (show message, allow) | "block" (prevent the action)
  field   : which part to match:
              bash   → "command"
              file   → "file_path" or "new_text"
              prompt → "user_prompt"
              stop   → (omit field)
  message : plain text shown to Claude when the pattern matches

WORKING EXAMPLES:

Warn if console.log left in edited files:
  event="file", pattern="console\\.log", action="warn", field="new_text",
  message="console.log found — remove before committing"

Block dangerous rm -rf:
  event="bash", pattern="rm\\s+-rf", action="block", field="command",
  message="Dangerous rm -rf — use a safer alternative"

Warn when editing files matching a path pattern:
  event="file", pattern="\\.env$", action="warn", field="file_path",
  message=".env file edited — ensure no secrets are committed"

Warn before force-push:
  event="bash", pattern="push.*--force", action="warn", field="command",
  message="Force push detected — confirm this is intentional"

Reminder at session stop:
  event="stop", pattern=".*", action="warn",
  message="Did you run the test suite before stopping?"

Only include suggested_hook when the pattern is unambiguous and directly addresses the finding. Omit it when unsure.

You MUST respond with valid JSON only. No prose before or after. Schema:
{
  "status": "CONSENSUS_REACHED" | "CONTINUE",
  "rationale": "brief reason for your decision",
  "findings": [
    {
      "type": "strength" | "improvement",
      "scope": "project" | "global",
      "text": "specific, actionable finding referencing concrete evidence from this session",
      "suggested_hook": {
        "event": "file",
        "pattern": "console\\.log",
        "action": "warn",
        "field": "new_text",
        "message": "console.log found — remove before committing"
      }
    }
  ]
}`

func runConsensus(ctx context.Context, sessionSummary, advocateOut, criticOut, challengerOut string, previousRounds []DebateRound) (ConsensusOutput, error) {
	prompt := buildContextPrompt("Consensus", sessionSummary, previousRounds,
		fmt.Sprintf(`Synthesise a consensus from the following debate outputs:

**Advocate:**
%s

**Critic:**
%s

**Challenger:**
%s

Output valid JSON only per the schema in your system prompt.`,
			advocateOut, criticOut, challengerOut))

	raw, err := call(ctx, config.ModelConsensus, consensusSystem, prompt)
	if err != nil {
		return ConsensusOutput{}, fmt.Errorf("consensus: %w", err)
	}

	return parseConsensus(raw)
}

func parseConsensus(raw string) (ConsensusOutput, error) {
	// Strip markdown code fences if present
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) > 2 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var out ConsensusOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		// Fallback: treat as CONTINUE so the debate doesn't terminate on bad JSON
		return ConsensusOutput{
			Status:    StatusContinue,
			Rationale: fmt.Sprintf("failed to parse consensus JSON: %v — raw: %.200s", err, raw),
		}, nil
	}
	return out, nil
}
