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
	Type  FindingType `json:"type"`
	Scope Scope       `json:"scope"`
	Text  string      `json:"text"`
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
