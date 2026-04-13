package debate

// FindingType categorises a finding.
type FindingType string

const (
	Strength    FindingType = "strength"
	Improvement FindingType = "improvement"
)

// Scope determines where a finding is injected.
type Scope string

const (
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
)

// Finding is a single consensus-approved observation.
type Finding struct {
	Type            FindingType      `json:"type"`
	Scope           Scope            `json:"scope"`
	Text            string           `json:"text"`
	// SuggestedHook is optional. When set, the finding is actionable enough
	// to warrant a Claude Code hook (e.g. a lint check after file edits).
	// The command must be a safe, read-only or lint-style shell command.
	SuggestedHook   *HookSuggestion  `json:"suggested_hook,omitempty"`
}

// HookSuggestion describes a hookify rule to create.
// Hookify rules are written as .claude/hookify.{name}.local.md files and
// activated automatically by the hookify Claude Code plugin.
type HookSuggestion struct {
	Event   string `json:"event"`          // "bash" | "file" | "prompt" | "stop"
	Pattern string `json:"pattern"`        // Python regex pattern to match
	Action  string `json:"action"`         // "warn" | "block"
	Field   string `json:"field,omitempty"` // "command" | "file_path" | "new_text" | "user_prompt"
	Message string `json:"message"`        // message shown to Claude when pattern matches
}

// ConsensusStatus is returned by the consensus agent each round.
type ConsensusStatus string

const (
	StatusContinue  ConsensusStatus = "CONTINUE"
	StatusReached   ConsensusStatus = "CONSENSUS_REACHED"
)

// ConsensusOutput is the structured JSON the consensus agent returns.
type ConsensusOutput struct {
	Status   ConsensusStatus `json:"status"`
	Findings []Finding       `json:"findings"`
	Rationale string         `json:"rationale,omitempty"`
}

// DebateRound holds the outputs of all 4 agents for one round.
type DebateRound struct {
	Round       int
	AdvocateOut string
	CriticOut   string
	ChallengerOut string
	Consensus   ConsensusOutput
}

// Result is the final output of a completed debate.
type Result struct {
	Rounds     []DebateRound
	Findings   []Finding
	RoundCount int
}
