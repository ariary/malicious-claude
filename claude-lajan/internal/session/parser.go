package session

import (
	"bufio"
	"encoding/json"
	"os"
	"time"
)

// Entry represents a single line in a Claude Code JSONL transcript.
type Entry struct {
	Type       string    `json:"type"`
	UUID       string    `json:"uuid"`
	ParentUUID string    `json:"parentUuid"`
	SessionID  string    `json:"sessionId"`
	Timestamp  time.Time `json:"timestamp"`
	CWD        string    `json:"cwd"`
	Message    *Message  `json:"message,omitempty"`

	// tool result fields (type == "user" with tool results)
	ToolUseResult *ToolUseResult `json:"toolUseResult,omitempty"`
}

type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
	Model   string         `json:"model,omitempty"`
	Usage   *EntryUsage    `json:"usage,omitempty"`
}

type EntryUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Name      string          `json:"name,omitempty"`        // tool_use
	Input     json.RawMessage `json:"input,omitempty"`       // tool_use
	ToolUseID string          `json:"tool_use_id,omitempty"` // tool_result
	Content   json.RawMessage `json:"content,omitempty"`     // tool_result
	IsError   *bool           `json:"is_error,omitempty"`    // tool_result error flag
}

type ToolUseResult struct {
	Success     bool   `json:"success"`
	CommandName string `json:"commandName,omitempty"`
}

// Metrics holds computed statistics for a session.
type Metrics struct {
	Duration           time.Duration
	TotalInputTokens   int64
	TotalOutputTokens  int64
	TotalCacheCreation int64
	TotalCacheRead     int64
	EstimatedCostUSD   float64
	ToolCallsTotal     int
	ToolCallsFailed    int
	ToolCallsByName    map[string]int
}

// Session is the parsed, structured form of a JSONL transcript.
type Session struct {
	ID        string
	CWD       string
	StartTime time.Time
	EndTime   time.Time
	Entries   []Entry

	// Derived stats
	UserTurns      int
	AssistantTurns int
	ToolCalls      []ToolCall
	FilesTouched   []string
	BashCommands   []string
	Metrics        Metrics
}

type ToolCall struct {
	Name  string
	Input json.RawMessage
}

// Parse reads a JSONL file and returns a structured Session.
func Parse(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	// Increase buffer for large lines
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return derive(path, entries), nil
}

func derive(_ string, entries []Entry) *Session {
	s := &Session{
		Entries: entries,
		Metrics: Metrics{ToolCallsByName: make(map[string]int)},
	}
	seen := map[string]bool{}
	model := ""

	for _, e := range entries {
		if s.ID == "" && e.SessionID != "" {
			s.ID = e.SessionID
		}
		if s.CWD == "" && e.CWD != "" {
			s.CWD = e.CWD
		}
		if !e.Timestamp.IsZero() {
			if s.StartTime.IsZero() || e.Timestamp.Before(s.StartTime) {
				s.StartTime = e.Timestamp
			}
			if e.Timestamp.After(s.EndTime) {
				s.EndTime = e.Timestamp
			}
		}
		if e.Type == "user" && e.Message != nil && e.Message.Role == "user" {
			s.UserTurns++
			// Count failed tool results in user messages
			for _, block := range e.Message.Content {
				if block.Type == "tool_result" && block.IsError != nil && *block.IsError {
					s.Metrics.ToolCallsFailed++
				}
			}
		}
		if e.Type == "assistant" && e.Message != nil && e.Message.Role == "assistant" {
			s.AssistantTurns++
			// Capture model from first assistant message
			if model == "" && e.Message.Model != "" {
				model = e.Message.Model
			}
			// Accumulate token usage
			if e.Message.Usage != nil {
				u := e.Message.Usage
				s.Metrics.TotalInputTokens += u.InputTokens
				s.Metrics.TotalOutputTokens += u.OutputTokens
				s.Metrics.TotalCacheCreation += u.CacheCreationInputTokens
				s.Metrics.TotalCacheRead += u.CacheReadInputTokens
			}
			for _, block := range e.Message.Content {
				if block.Type == "tool_use" {
					s.ToolCalls = append(s.ToolCalls, ToolCall{Name: block.Name, Input: block.Input})
					s.Metrics.ToolCallsByName[block.Name]++
					// Extract file paths from Read/Edit/Write/Glob calls
					if block.Name == "Read" || block.Name == "Edit" || block.Name == "Write" {
						var inp struct {
							FilePath string `json:"file_path"`
						}
						if json.Unmarshal(block.Input, &inp) == nil && inp.FilePath != "" && !seen[inp.FilePath] {
							s.FilesTouched = append(s.FilesTouched, inp.FilePath)
							seen[inp.FilePath] = true
						}
					}
					// Extract bash commands
					if block.Name == "Bash" {
						var inp struct {
							Command string `json:"command"`
						}
						if json.Unmarshal(block.Input, &inp) == nil && inp.Command != "" {
							s.BashCommands = append(s.BashCommands, inp.Command)
						}
					}
				}
			}
		}
	}

	// Compute derived metrics
	s.Metrics.ToolCallsTotal = len(s.ToolCalls)
	if !s.StartTime.IsZero() && !s.EndTime.IsZero() {
		s.Metrics.Duration = s.EndTime.Sub(s.StartTime)
	}
	s.Metrics.EstimatedCostUSD = CalculateCost(
		model,
		s.Metrics.TotalInputTokens,
		s.Metrics.TotalOutputTokens,
		s.Metrics.TotalCacheCreation,
		s.Metrics.TotalCacheRead,
	)

	return s
}
