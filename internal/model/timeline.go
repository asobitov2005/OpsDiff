package model

import "time"

type Timeline struct {
	Version     string          `json:"version"`
	Cluster     string          `json:"cluster"`
	Namespace   string          `json:"namespace"`
	WindowStart time.Time       `json:"window_start"`
	WindowEnd   time.Time       `json:"window_end"`
	GeneratedAt time.Time       `json:"generated_at"`
	Summary     TimelineSummary `json:"summary"`
	Events      []TimelineEvent `json:"events"`
}

type TimelineEvent struct {
	ID           string    `json:"id"`
	Time         time.Time `json:"time"`
	Source       string    `json:"source"`
	Category     string    `json:"category"`
	Severity     string    `json:"severity"`
	Namespace    string    `json:"namespace"`
	Service      string    `json:"service,omitempty"`
	ResourceKind string    `json:"resource_kind,omitempty"`
	ResourceName string    `json:"resource_name,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	Message      string    `json:"message"`
	RiskTags     []string  `json:"risk_tags,omitempty"`
}

type TimelineSummary struct {
	Total      int `json:"total"`
	Critical   int `json:"critical"`
	Warning    int `json:"warning"`
	Info       int `json:"info"`
	Changes    int `json:"changes"`
	Symptoms   int `json:"symptoms"`
	Evidence   int `json:"evidence"`
	Restarts   int `json:"restarts"`
	OOMKills   int `json:"oomkills"`
	CrashLoops int `json:"crashloops"`
}
