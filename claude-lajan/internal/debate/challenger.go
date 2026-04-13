package debate

import (
	"context"
	"fmt"

	"github.com/ariary/claude-lajan/internal/config"
)

const challengerSystem = `You are the Challenger agent in a multi-agent analysis of a Claude Code AI session.

Your role is to stress-test the findings made by both the Advocate and the Critic:
- Is the Advocate's praise actually warranted, or was it luck/coincidence?
- Is the Critic's critique actually fair, or was the approach reasonable given the context?
- Are there findings that are too vague to be actionable?
- Are there findings that contradict each other?
- Which findings are genuinely important vs. trivial?

Push back hard on weak reasoning from both sides. Your goal is to force rigour.
Keep your response concise (3-5 bullet points max). Focus on the most important challenges.`

func runChallenger(ctx context.Context, sessionSummary, advocateOut, criticOut string, previousRounds []DebateRound) (string, error) {
	prompt := buildContextPrompt("Challenger", sessionSummary, previousRounds,
		fmt.Sprintf(`Challenge and stress-test the following findings:

**Advocate's findings:**
%s

**Critic's findings:**
%s

Which findings are well-supported? Which are weak or vague? Push back where warranted.`,
			advocateOut, criticOut))

	out, err := call(ctx, config.ModelChallenger, challengerSystem, prompt)
	if err != nil {
		return "", fmt.Errorf("challenger: %w", err)
	}
	return out, nil
}
