package report

import (
	"fmt"
	"os"
	"strings"

	"github.com/ariary/claude-lajan/internal/config"
	"github.com/ariary/claude-lajan/internal/debate"
)

const digestHeader = "# Claude Session Reviewer — Digest\n\nTop learnings from recent sessions, injected at session start.\n\n"

// UpdateDigest merges new findings into the rolling digest file.
func UpdateDigest(findings []debate.Finding) error {
	existing := loadDigest()
	merged := merge(existing, findings)

	// Keep only top N
	if len(merged) > config.DigestMaxItems {
		merged = merged[:config.DigestMaxItems]
	}

	return writeDigest(merged)
}

// LoadTopN returns the top N findings from the digest for prompt injection.
func LoadTopN(n int) []debate.Finding {
	findings := loadDigest()
	if len(findings) > n {
		findings = findings[:n]
	}
	return findings
}

func loadDigest() []debate.Finding {
	data, err := os.ReadFile(config.DigestFile())
	if err != nil {
		return nil
	}
	return parseDigest(string(data))
}

// parseDigest reads the markdown digest and reconstructs findings.
// Format per line: `- [type/scope] text`
func parseDigest(content string) []debate.Finding {
	var findings []debate.Finding
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		// Parse [type/scope]
		if !strings.HasPrefix(line, "[") {
			continue
		}
		end := strings.Index(line, "]")
		if end < 0 {
			continue
		}
		tag := line[1:end]
		text := strings.TrimSpace(line[end+1:])
		parts := strings.SplitN(tag, "/", 2)
		if len(parts) != 2 {
			continue
		}
		findings = append(findings, debate.Finding{
			Type:  debate.FindingType(parts[0]),
			Scope: debate.Scope(parts[1]),
			Text:  text,
		})
	}
	return findings
}

// merge deduplicates and prepends new findings to existing ones.
func merge(existing, newFindings []debate.Finding) []debate.Finding {
	seen := map[string]bool{}
	for _, f := range existing {
		seen[f.Text] = true
	}
	var result []debate.Finding
	for _, f := range newFindings {
		if !seen[f.Text] {
			result = append(result, f)
			seen[f.Text] = true
		}
	}
	return append(result, existing...)
}

func writeDigest(findings []debate.Finding) error {
	if err := os.MkdirAll(config.ReviewerDir(), 0755); err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(digestHeader)
	for _, f := range findings {
		fmt.Fprintf(&sb, "- [%s/%s] %s\n", f.Type, f.Scope, f.Text)
	}
	return os.WriteFile(config.DigestFile(), []byte(sb.String()), 0644)
}
