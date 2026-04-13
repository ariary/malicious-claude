package debate

import (
	"context"
	"fmt"

	"github.com/ariary/claude-lajan/internal/config"
)

const advocateSystem = `You are the Advocate agent in a multi-agent analysis of a Claude Code AI session.

Your role is to identify what worked WELL in this session:
- Efficient tool usage and reasoning steps
- Good decisions that saved time or avoided errors
- Effective patterns worth repeating
- Clear, focused problem-solving

Be specific and evidence-based. Reference actual events from the session.
Keep your response concise (3-5 bullet points max). Each point must be directly supported by session evidence.`

func runAdvocate(ctx context.Context, sessionSummary string, previousRounds []DebateRound) (string, error) {
	prompt := buildContextPrompt("Advocate", sessionSummary, previousRounds,
		"Identify what worked well in this session. Be specific and cite evidence.")
	out, err := call(ctx, config.ModelAdvocate, advocateSystem, prompt)
	if err != nil {
		return "", fmt.Errorf("advocate: %w", err)
	}
	return out, nil
}

func buildContextPrompt(role, sessionSummary string, previousRounds []DebateRound, instruction string) string {
	prompt := fmt.Sprintf("## Session Summary\n\n%s\n\n", sessionSummary)
	if len(previousRounds) > 0 {
		prompt += "## Previous Round Context\n\n"
		last := previousRounds[len(previousRounds)-1]
		prompt += fmt.Sprintf("**Round %d Advocate:** %s\n\n", last.Round, last.AdvocateOut)
		prompt += fmt.Sprintf("**Round %d Critic:** %s\n\n", last.Round, last.CriticOut)
		prompt += fmt.Sprintf("**Round %d Challenger:** %s\n\n", last.Round, last.ChallengerOut)
		if last.Consensus.Rationale != "" {
			prompt += fmt.Sprintf("**Round %d Consensus Rationale:** %s\n\n", last.Round, last.Consensus.Rationale)
		}
	}
	prompt += fmt.Sprintf("## Your Task (%s)\n\n%s", role, instruction)
	return prompt
}
