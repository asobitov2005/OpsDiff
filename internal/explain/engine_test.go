package explain

import (
	"testing"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

func TestExplainRanksMemoryLimitChangeAboveImageChangeForOOM(t *testing.T) {
	before := model.Snapshot{
		Version:   "v1",
		Cluster:   "prod",
		Namespace: "prod",
		CreatedAt: time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC),
		Resources: []model.Resource{
			{
				Kind:      "Deployment",
				Namespace: "prod",
				Name:      "api",
				Deployment: &model.DeploymentState{
					Containers: []model.ContainerState{
						{
							Name:  "api",
							Image: "api:v1.8.2",
							Resources: model.ResourceState{
								MemoryLimit: "512Mi",
							},
						},
					},
				},
			},
		},
	}

	after := model.Snapshot{
		Version:   "v1",
		Cluster:   "prod",
		Namespace: "prod",
		CreatedAt: time.Date(2026, 4, 25, 14, 5, 0, 0, time.UTC),
		Resources: []model.Resource{
			{
				Kind:      "Deployment",
				Namespace: "prod",
				Name:      "api",
				Deployment: &model.DeploymentState{
					Containers: []model.ContainerState{
						{
							Name:  "api",
							Image: "api:v1.8.3",
							Resources: model.ResourceState{
								MemoryLimit: "256Mi",
							},
						},
					},
				},
			},
		},
	}

	timeline := model.Timeline{
		Version:     "v1",
		Cluster:     "prod",
		Namespace:   "prod",
		WindowStart: time.Date(2026, 4, 25, 13, 30, 0, 0, time.UTC),
		WindowEnd:   time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC),
		Events: []model.TimelineEvent{
			{
				ID:           "evt_001",
				Time:         time.Date(2026, 4, 25, 14, 6, 0, 0, time.UTC),
				Source:       "kubernetes",
				Category:     "symptom",
				Severity:     "critical",
				Namespace:    "prod",
				Service:      "api",
				ResourceKind: "Pod",
				ResourceName: "api-79fdb46b7f-r2k7d",
				Reason:       "OOMKilled",
				Message:      "container api was OOMKilled",
				RiskTags:     []string{"oomkilled", "restart-evidence"},
			},
			{
				ID:           "evt_002",
				Time:         time.Date(2026, 4, 25, 14, 7, 0, 0, time.UTC),
				Source:       "kubernetes",
				Category:     "evidence",
				Severity:     "critical",
				Namespace:    "prod",
				Service:      "api",
				ResourceKind: "Pod",
				ResourceName: "api-79fdb46b7f-r2k7d",
				Reason:       "ContainerRestarted",
				Message:      "container api restarted 4 times",
				RiskTags:     []string{"restart-evidence"},
			},
		},
	}
	timeline.Summary = model.TimelineSummary{
		Total:    2,
		Critical: 2,
		Symptoms: 1,
		Evidence: 1,
		Restarts: 1,
		OOMKills: 1,
	}

	result := NewEngine().Explain(before, after, timeline, "before.json", "after.json", 5)
	if len(result.Candidates) < 2 {
		t.Fatalf("expected at least 2 candidates, got %d", len(result.Candidates))
	}

	top := result.Candidates[0]
	if top.Change.Path != "spec.template.spec.containers.api.resources.limits.memory" {
		t.Fatalf("expected top candidate to be memory limit change, got %s", top.Change.Path)
	}
	if top.Score <= result.Candidates[1].Score {
		t.Fatalf("expected top candidate score %d to be greater than second %d", top.Score, result.Candidates[1].Score)
	}
	if top.Likelihood != "high" {
		t.Fatalf("expected high likelihood, got %s", top.Likelihood)
	}
	if len(top.MatchedEvents) == 0 {
		t.Fatalf("expected matched events for top candidate")
	}
}
