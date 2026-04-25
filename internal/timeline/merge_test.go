package timeline

import (
	"testing"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

func TestMergeSortsEventsAndRefreshesSummary(t *testing.T) {
	base := model.Timeline{
		Version:     "v1",
		Cluster:     "prod",
		Namespace:   "prod",
		WindowStart: time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC),
		GeneratedAt: time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC),
		Events: []model.TimelineEvent{
			{
				Time:      time.Date(2026, 4, 25, 14, 20, 0, 0, time.UTC),
				Severity:  "warning",
				Category:  "symptom",
				Namespace: "prod",
				Reason:    "BackOff",
				Message:   "pod is backing off",
			},
		},
	}

	merged := Merge(base, []model.TimelineEvent{
		{
			Time:      time.Date(2026, 4, 25, 14, 10, 0, 0, time.UTC),
			Severity:  "info",
			Category:  "change",
			Namespace: "prod",
			Reason:    "SyncSucceeded",
			Message:   "ArgoCD synced api",
		},
	}, 10)

	if len(merged.Events) != 2 {
		t.Fatalf("expected 2 merged events, got %d", len(merged.Events))
	}
	if merged.Events[0].Reason != "SyncSucceeded" {
		t.Fatalf("expected imported event to be sorted first, got %+v", merged.Events[0])
	}
	if merged.Summary.Changes != 1 || merged.Summary.Symptoms != 1 {
		t.Fatalf("unexpected summary after merge: %+v", merged.Summary)
	}
}
