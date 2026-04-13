package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ariary/claude-lajan/internal/config"
	"github.com/ariary/claude-lajan/internal/debate"
	"github.com/ariary/claude-lajan/internal/session"
)

// formatTokens formats an int64 token count with comma separators (e.g. 53333 → "53,333").
func formatTokens(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	rem := len(s) % 3
	if rem > 0 {
		out = append(out, s[:rem]...)
	}
	for i := rem; i < len(s); i += 3 {
		if len(out) > 0 {
			out = append(out, ',')
		}
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}

// renderMetrics returns a markdown table summarising session metrics.
func renderMetrics(m session.Metrics) string {
	var sb strings.Builder

	sb.WriteString("## Session Metrics\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")

	// Duration
	dur := m.Duration.Round(time.Second).String()
	fmt.Fprintf(&sb, "| Duration | %s |\n", dur)

	// Total tokens
	totalTokens := m.TotalInputTokens + m.TotalOutputTokens
	fmt.Fprintf(&sb, "| Total tokens | %s (↓ %s input · ↑ %s output) |\n",
		formatTokens(totalTokens),
		formatTokens(m.TotalInputTokens),
		formatTokens(m.TotalOutputTokens),
	)

	// Cache savings — show token count; cost breakdown not available without per-price data
	fmt.Fprintf(&sb, "| Cache savings | %s tokens read |\n", formatTokens(m.TotalCacheRead))

	// Estimated cost
	fmt.Fprintf(&sb, "| Estimated cost | $%.4f |\n", m.EstimatedCostUSD)

	// Tool calls
	fmt.Fprintf(&sb, "| Tool calls | %d total · %d failed |\n", m.ToolCallsTotal, m.ToolCallsFailed)

	// Failed tools — we only have total failed count, not per-tool breakdown
	if m.ToolCallsFailed > 0 {
		fmt.Fprintf(&sb, "| Failed tools | %d failed (no per-tool breakdown) |\n", m.ToolCallsFailed)
	}

	return sb.String()
}

// Write saves a per-session markdown report and returns its path.
func Write(s *session.Session, result *debate.Result) (string, error) {
	dir := filepath.Join(config.ReportsDir(), time.Now().Format("2006-01-02"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	reportPath := filepath.Join(dir, s.ID+".md")
	content := render(s, result)
	if err := os.WriteFile(reportPath, []byte(content), 0644); err != nil {
		return "", err
	}
	return reportPath, nil
}

func render(s *session.Session, result *debate.Result) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Session Review: %s\n\n", s.ID)
	fmt.Fprintf(&sb, "**Date:** %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "**CWD:** %s\n", s.CWD)
	if !s.StartTime.IsZero() && !s.EndTime.IsZero() {
		fmt.Fprintf(&sb, "**Duration:** %s\n", s.EndTime.Sub(s.StartTime).Round(time.Second))
	}
	fmt.Fprintf(&sb, "**Debate rounds:** %d\n\n", result.RoundCount)

	// Findings summary
	var strengths, improvements []debate.Finding
	for _, f := range result.Findings {
		if f.Type == debate.Strength {
			strengths = append(strengths, f)
		} else {
			improvements = append(improvements, f)
		}
	}

	sb.WriteString("---\n\n")

	if len(strengths) > 0 {
		sb.WriteString("## Strengths\n\n")
		for _, f := range strengths {
			fmt.Fprintf(&sb, "- **[%s]** %s\n", f.Scope, f.Text)
		}
		sb.WriteString("\n")
	}

	if len(improvements) > 0 {
		sb.WriteString("## Areas for Improvement\n\n")
		for _, f := range improvements {
			fmt.Fprintf(&sb, "- **[%s]** %s\n", f.Scope, f.Text)
		}
		sb.WriteString("\n")
	}

	if len(result.Findings) == 0 {
		sb.WriteString("## Result\n\nNo significant findings — session was efficient and well-executed.\n\n")
	}

	// Session Metrics
	sb.WriteString("---\n\n")
	sb.WriteString(renderMetrics(s.Metrics))
	sb.WriteString("\n")

	// Debate transcript
	sb.WriteString("---\n\n## Debate Transcript\n\n")
	for _, r := range result.Rounds {
		fmt.Fprintf(&sb, "### Round %d\n\n", r.Round)
		fmt.Fprintf(&sb, "**Advocate:**\n%s\n\n", r.AdvocateOut)
		fmt.Fprintf(&sb, "**Critic:**\n%s\n\n", r.CriticOut)
		fmt.Fprintf(&sb, "**Challenger:**\n%s\n\n", r.ChallengerOut)
		fmt.Fprintf(&sb, "**Consensus:** `%s` — %s\n\n", r.Consensus.Status, r.Consensus.Rationale)
	}

	return sb.String()
}
