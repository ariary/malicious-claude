package debate

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ariary/claude-lajan/internal/config"
	"github.com/ariary/claude-lajan/internal/session"
)

// Run executes the adversarial debate for a session and returns the final consensus.
func Run(ctx context.Context, s *session.Session) (*Result, error) {
	summary := buildSessionSummary(s)
	return RunWithSummary(ctx, summary)
}

// RunN executes the adversarial debate with a caller-supplied max rounds limit.
func RunN(ctx context.Context, s *session.Session, maxRounds int) (*Result, error) {
	summary := buildSessionSummary(s)
	return runDebate(ctx, summary, maxRounds)
}

// RunWithSummary runs the debate with a pre-built session summary string (useful for testing).
func RunWithSummary(ctx context.Context, summary string) (*Result, error) {
	return runDebate(ctx, summary, config.MaxDebateRounds)
}

func runDebate(ctx context.Context, summary string, maxRounds int) (*Result, error) {
	var rounds []DebateRound

	for round := 1; round <= maxRounds; round++ {
		fmt.Printf("  [round %d/%d] running debate...\n", round, maxRounds)

		dr, err := runRound(ctx, summary, rounds, round)
		if err != nil {
			return nil, fmt.Errorf("round %d: %w", round, err)
		}
		rounds = append(rounds, dr)

		fmt.Printf("  [round %d] consensus status: %s\n", round, dr.Consensus.Status)
		if dr.Consensus.Status == StatusReached {
			break
		}
	}

	// Collect findings from the last round that reached consensus (or the final round)
	var findings []Finding
	if len(rounds) > 0 {
		findings = rounds[len(rounds)-1].Consensus.Findings
	}

	return &Result{
		Rounds:     rounds,
		Findings:   findings,
		RoundCount: len(rounds),
	}, nil
}

// runRound executes one full debate round.
// Advocate and Critic run in parallel, then Challenger, then Consensus.
func runRound(ctx context.Context, summary string, previous []DebateRound, roundNum int) (DebateRound, error) {
	dr := DebateRound{Round: roundNum}

	// --- Phase 1: Advocate + Critic in parallel ---
	var (
		advOut, critOut string
		advErr, critErr error
		wg              sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		advOut, advErr = runAdvocate(ctx, summary, previous)
	}()
	go func() {
		defer wg.Done()
		critOut, critErr = runCritic(ctx, summary, previous)
	}()
	wg.Wait()

	if advErr != nil {
		return dr, advErr
	}
	if critErr != nil {
		return dr, critErr
	}
	dr.AdvocateOut = advOut
	dr.CriticOut = critOut

	// --- Phase 2: Challenger ---
	chalOut, err := runChallenger(ctx, summary, advOut, critOut, previous)
	if err != nil {
		return dr, err
	}
	dr.ChallengerOut = chalOut

	// --- Phase 3: Consensus ---
	cons, err := runConsensus(ctx, summary, advOut, critOut, chalOut, previous)
	if err != nil {
		return dr, err
	}
	dr.Consensus = cons

	return dr, nil
}

// buildSessionSummary constructs a concise human-readable summary of a session
// suitable for feeding to the debate agents.
func buildSessionSummary(s *session.Session) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "**Session ID:** %s\n", s.ID)
	fmt.Fprintf(&sb, "**Working Directory:** %s\n", s.CWD)
	if !s.StartTime.IsZero() && !s.EndTime.IsZero() {
		dur := s.EndTime.Sub(s.StartTime).Round(time.Second)
		fmt.Fprintf(&sb, "**Duration:** %s\n", dur)
	}
	fmt.Fprintf(&sb, "**User turns:** %d | **Assistant turns:** %d\n", s.UserTurns, s.AssistantTurns)
	fmt.Fprintf(&sb, "**Tool calls:** %d\n", len(s.ToolCalls))

	if len(s.FilesTouched) > 0 {
		sb.WriteString("\n**Files touched:**\n")
		for _, f := range s.FilesTouched {
			fmt.Fprintf(&sb, "- %s\n", f)
		}
	}

	if len(s.BashCommands) > 0 {
		sb.WriteString("\n**Bash commands run (first 20):**\n")
		limit := min(20, len(s.BashCommands))
		for _, cmd := range s.BashCommands[:limit] {
			if len(cmd) > 120 {
				cmd = cmd[:120] + "..."
			}
			fmt.Fprintf(&sb, "- `%s`\n", cmd)
		}
	}

	// Metrics block — gives agents concrete efficiency data
	m := s.Metrics
	sb.WriteString("\n**Metrics:**\n")
	fmt.Fprintf(&sb, "- Duration: %s\n", m.Duration.Round(time.Second))
	fmt.Fprintf(&sb, "- Estimated cost: $%.4f\n", m.EstimatedCostUSD)
	fmt.Fprintf(&sb, "- Tool calls: %d total · %d failed\n", m.ToolCallsTotal, m.ToolCallsFailed)
	fmt.Fprintf(&sb, "- Input tokens: %d · Output: %d · Cache read: %d\n",
		m.TotalInputTokens, m.TotalOutputTokens, m.TotalCacheRead)
	sb.WriteString("\n")

	// Include the raw conversation (truncated) so agents can reason about the actual dialogue
	sb.WriteString("\n**Conversation transcript (truncated):**\n\n")
	charBudget := 8000
	for _, entry := range s.Entries {
		if entry.Message == nil {
			continue
		}
		role := entry.Message.Role
		if role == "" {
			continue
		}
		for _, block := range entry.Message.Content {
			var text string
			switch block.Type {
			case "text":
				text = block.Text
			case "tool_use":
				text = fmt.Sprintf("[tool_use: %s]", block.Name)
			case "tool_result":
				text = "[tool_result]"
			case "thinking":
				continue // skip internal thinking
			}
			if text == "" {
				continue
			}
			if len(text) > 500 {
				text = text[:500] + "..."
			}
			line := fmt.Sprintf("**%s:** %s\n\n", role, text)
			if charBudget-len(line) < 0 {
				sb.WriteString("*[transcript truncated]*\n")
				goto done
			}
			sb.WriteString(line)
			charBudget -= len(line)
		}
	}
done:
	return sb.String()
}
