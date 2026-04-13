package report

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ariary/claude-lajan/internal/config"
)

// SessionSummary is a lightweight record of one session's key metrics,
// parsed from an existing markdown report file.
type SessionSummary struct {
	Date             string
	SessionID        string
	DurationStr      string
	EstimatedCostUSD float64
	ToolCallsTotal   int
	ToolCallsFailed  int
	TotalTokens      int64
}

// LoadRecentSummaries returns up to n session summaries sorted oldest→newest,
// parsed from report markdown files in ~/.claude-lajan/reports/.
func LoadRecentSummaries(n int) []SessionSummary {
	reportsDir := config.ReportsDir()

	type fileEntry struct {
		path    string
		modTime int64
	}

	var files []fileEntry
	_ = filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			files = append(files, fileEntry{path: path, modTime: info.ModTime().UnixNano()})
		}
		return nil
	})

	// Sort newest first
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	// Take the most recent n
	if len(files) > n {
		files = files[:n]
	}

	// Parse each file
	var summaries []SessionSummary
	for _, f := range files {
		if s, ok := parseSummaryFromReport(f.path); ok {
			summaries = append(summaries, s)
		}
	}

	// Reverse so result is oldest→newest
	for i, j := 0, len(summaries)-1; i < j; i, j = i+1, j-1 {
		summaries[i], summaries[j] = summaries[j], summaries[i]
	}

	return summaries
}

// parseSummaryFromReport reads a markdown report file and extracts key metrics.
func parseSummaryFromReport(path string) (SessionSummary, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SessionSummary{}, false
	}

	lines := strings.Split(string(data), "\n")

	var s SessionSummary

	for _, line := range lines {
		// Session ID from header: # Session Review: <ID>
		if strings.HasPrefix(line, "# Session Review: ") {
			s.SessionID = strings.TrimPrefix(line, "# Session Review: ")
			s.SessionID = strings.TrimSpace(s.SessionID)
			continue
		}

		// Date from: **Date:** 2026-04-13 15:04:05
		if strings.HasPrefix(line, "**Date:**") {
			raw := strings.TrimPrefix(line, "**Date:**")
			raw = strings.TrimSpace(raw)
			// Take only the date part (first field)
			if parts := strings.Fields(raw); len(parts) > 0 {
				s.Date = parts[0]
			}
			continue
		}

		// Metrics table rows — format: | Metric | Value |
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cols := strings.Split(line, "|")
		if len(cols) < 3 {
			continue
		}
		key := strings.TrimSpace(cols[1])
		val := strings.TrimSpace(cols[2])

		switch key {
		case "Duration":
			s.DurationStr = val

		case "Estimated cost":
			// val looks like "$0.1234"
			clean := strings.TrimPrefix(val, "$")
			if f, err := strconv.ParseFloat(clean, 64); err == nil {
				s.EstimatedCostUSD = f
			} else {
				return SessionSummary{}, false
			}

		case "Tool calls":
			// val looks like "34 total · 2 failed"
			parts := strings.Split(val, "·")
			if len(parts) >= 1 {
				totalPart := strings.TrimSpace(parts[0])
				totalPart = strings.TrimSuffix(totalPart, " total")
				if n, err := strconv.Atoi(strings.TrimSpace(totalPart)); err == nil {
					s.ToolCallsTotal = n
				}
			}
			if len(parts) >= 2 {
				failedPart := strings.TrimSpace(parts[1])
				failedPart = strings.TrimSuffix(failedPart, " failed")
				if n, err := strconv.Atoi(strings.TrimSpace(failedPart)); err == nil {
					s.ToolCallsFailed = n
				}
			}

		case "Total tokens":
			// val looks like "53,333 (↓ 45k input · ↑ 8k output)"
			// take the number before the first space
			beforeSpace := strings.Fields(val)
			if len(beforeSpace) > 0 {
				numStr := strings.ReplaceAll(beforeSpace[0], ",", "")
				if n, err := strconv.ParseInt(numStr, 10, 64); err == nil {
					s.TotalTokens = n
				}
			}
		}
	}

	// Require at least a session ID to be considered valid
	if s.SessionID == "" {
		return SessionSummary{}, false
	}

	return s, true
}
