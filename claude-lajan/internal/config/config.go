package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	ModelAdvocate   = "claude-sonnet-4-5"
	ModelCritic     = "claude-sonnet-4-5"
	ModelChallenger = "claude-sonnet-4-5"
	ModelConsensus  = "claude-opus-4-6"

	MaxDebateRounds = 5
	MaxTokens       = 2048
	DigestMaxItems  = 20
	DigestInjectTop = 5
)

// ReviewerDir returns the runtime data directory (~/.claude-lajan/).
func ReviewerDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude-lajan")
}

func QueueFile() string {
	return filepath.Join(ReviewerDir(), "queue.txt")
}

func ReportsDir() string {
	return filepath.Join(ReviewerDir(), "reports")
}

func DigestFile() string {
	return filepath.Join(ReviewerDir(), "digest.md")
}

func BinDir() string {
	return filepath.Join(ReviewerDir(), "bin")
}

// ClaudeDir returns ~/.claude/
func ClaudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// GlobalClaudeMD returns ~/.claude/CLAUDE.md
func GlobalClaudeMD() string {
	return filepath.Join(ClaudeDir(), "CLAUDE.md")
}

// GlobalSettings returns ~/.claude/settings.json
func GlobalSettings() string {
	return filepath.Join(ClaudeDir(), "settings.json")
}

// ProjectMemoryFile returns the feedback_learnings.md path for a given project cwd.
// Claude Code encodes the path by replacing / with -.
func ProjectMemoryFile(cwd string) string {
	encoded := encodeProjectPath(cwd)
	return filepath.Join(ClaudeDir(), "projects", encoded, "memory", "feedback_learnings.md")
}

// encodeProjectPath mirrors Claude Code's project dir encoding: replace / with -
func encodeProjectPath(path string) string {
	return strings.ReplaceAll(path, "/", "-")
}

func AnthropicAPIKey() string {
	return os.Getenv("ANTHROPIC_API_KEY")
}
