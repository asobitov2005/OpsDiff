package model

import "time"

type ExplainResult struct {
	BeforePath      string             `json:"before_path"`
	AfterPath       string             `json:"after_path"`
	Namespace       string             `json:"namespace"`
	GeneratedAt     time.Time          `json:"generated_at"`
	ChangeWindow    TimeWindow         `json:"change_window"`
	CompareSummary  Summary            `json:"compare_summary"`
	TimelineSummary TimelineSummary    `json:"timeline_summary"`
	Summary         ExplainSummary     `json:"summary"`
	Candidates      []ExplainCandidate `json:"candidates"`
}

type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type ExplainSummary struct {
	TotalChanges       int `json:"total_changes"`
	RankedCandidates   int `json:"ranked_candidates"`
	CriticalSymptoms   int `json:"critical_symptoms"`
	WarningSymptoms    int `json:"warning_symptoms"`
	SupportingEvidence int `json:"supporting_evidence"`
}

type ExplainCandidate struct {
	Rank            int                 `json:"rank"`
	Score           int                 `json:"score"`
	Likelihood      string              `json:"likelihood"`
	Service         string              `json:"service,omitempty"`
	Change          Change              `json:"change"`
	Evidence        []string            `json:"evidence,omitempty"`
	SuggestedChecks []string            `json:"suggested_checks,omitempty"`
	MatchedEvents   []ExplainEventMatch `json:"matched_events,omitempty"`
}

type ExplainEventMatch struct {
	Time         time.Time `json:"time"`
	Severity     string    `json:"severity"`
	Category     string    `json:"category"`
	Reason       string    `json:"reason,omitempty"`
	Message      string    `json:"message"`
	ResourceKind string    `json:"resource_kind,omitempty"`
	ResourceName string    `json:"resource_name,omitempty"`
	Contribution int       `json:"contribution"`
}
