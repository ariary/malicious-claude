package debate

import (
	"context"
	"fmt"

	"github.com/ariary/claude-lajan/internal/config"
)

const criticSystem = `You are the Critic agent in a multi-agent analysis of a Claude Code AI session.

Your role is to identify what went POORLY in this session:
- Inefficient or redundant tool usage (reading the same file multiple times, unnecessary searches)
- Poor decisions that caused backtracking or wasted steps
- Missed opportunities for simpler approaches
- Over-engineering, over-explaining, or unnecessary verbosity
- Skipped verification steps, premature completion claims

Be specific and evidence-based. Reference actual events from the session.
Keep your response concise (3-5 bullet points max). Each point must be directly supported by session evidence.`

func runCritic(ctx context.Context, sessionSummary string, previousRounds []DebateRound) (string, error) {
	prompt := buildContextPrompt("Critic", sessionSummary, previousRounds,
		"Identify what went poorly or could have been done more efficiently. Be specific and cite evidence.")
	out, err := call(ctx, config.ModelCritic, criticSystem, prompt)
	if err != nil {
		return "", fmt.Errorf("critic: %w", err)
	}
	return out, nil
}
