package report

import (
	"strings"
	"testing"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

func TestRenderIncidentHTMLIncludesCandidateAndTimeline(t *testing.T) {
	result := model.ExplainResult{
		BeforePath:  "before.json",
		AfterPath:   "after.json",
		Namespace:   "prod",
		GeneratedAt: time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC),
		ChangeWindow: model.TimeWindow{
			Start: time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 4, 25, 14, 5, 0, 0, time.UTC),
		},
		Summary: model.ExplainSummary{
			RankedCandidates:   1,
			CriticalSymptoms:   1,
			SupportingEvidence: 1,
		},
		Candidates: []model.ExplainCandidate{
			{
				Rank:       1,
				Score:      94,
				Likelihood: "high",
				Change: model.Change{
					ResourceKind: "Deployment",
					ResourceName: "api",
					Path:         "spec.template.spec.containers.api.resources.limits.memory",
					Before:       "512Mi",
					After:        "256Mi",
				},
				Evidence: []string{"memory-related runtime failure directly matches the changed field"},
			},
		},
	}

	timeline := model.Timeline{
		Namespace: "prod",
		Summary: model.TimelineSummary{
			Total:    1,
			Critical: 1,
			Symptoms: 1,
			OOMKills: 1,
		},
		Events: []model.TimelineEvent{
			{
				Time:      time.Date(2026, 4, 25, 14, 12, 0, 0, time.UTC),
				Severity:  "critical",
				Source:    "kubernetes",
				Service:   "api",
				Reason:    "OOMKilled",
				Message:   "container api was OOMKilled",
				Namespace: "prod",
			},
		},
	}

	html, err := RenderIncidentHTML(result, timeline)
	if err != nil {
		t.Fatalf("render html: %v", err)
	}

	for _, fragment := range []string{"OpsDiff Incident Report", "94/100", "Deployment/api", "OOMKilled"} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("expected HTML to contain %q", fragment)
		}
	}
}
