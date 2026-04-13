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

Rules:
1. Only include findings that survived the Challenger's scrutiny.
2. Each finding must be specific, actionable, and evidence-backed.
3. Tag each finding with scope: "project" (specific to this codebase) or "global" (applies to all Claude Code sessions).
4. If the debate has produced strong, non-contradictory, actionable findings → output CONSENSUS_REACHED.
5. If findings are still vague, contradictory, or need more scrutiny → output CONTINUE with guidance for next round.
6. If there is genuinely nothing significant to improve or praise → output CONSENSUS_REACHED with empty findings.

You MUST respond with valid JSON only. No prose before or after. Schema:
{
  "status": "CONSENSUS_REACHED" | "CONTINUE",
  "rationale": "brief reason for your decision",
  "findings": [
    {
      "type": "strength" | "improvement",
      "scope": "project" | "global",
      "text": "specific, actionable finding"
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
