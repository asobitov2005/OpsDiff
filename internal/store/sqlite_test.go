//go:build cgo

package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

func TestSQLiteStorePersistsSnapshotAndTimeline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opsdiff.db")
	store, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	snapshot := model.Snapshot{
		Version:   "v1",
		Cluster:   "prod",
		Namespace: "prod",
		CreatedAt: time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC),
		Resources: []model.Resource{
			{Kind: "Deployment", Namespace: "prod", Name: "api"},
		},
	}

	if err := store.SaveSnapshot(ctx, snapshot, "/tmp/before.json"); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	record, err := store.LatestSnapshot(ctx, "prod")
	if err != nil {
		t.Fatalf("latest snapshot: %v", err)
	}
	if record == nil || record.Path != "/tmp/before.json" || record.Snapshot.Cluster != "prod" {
		t.Fatalf("unexpected snapshot record: %+v", record)
	}

	timeline := model.Timeline{
		Version:     "v1",
		Cluster:     "prod",
		Namespace:   "prod",
		WindowStart: time.Date(2026, 4, 25, 13, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC),
		GeneratedAt: time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC),
		Events: []model.TimelineEvent{
			{
				Time:         time.Date(2026, 4, 25, 14, 12, 0, 0, time.UTC),
				Source:       "kubernetes",
				Category:     "symptom",
				Severity:     "critical",
				Namespace:    "prod",
				Service:      "api",
				ResourceKind: "Pod",
				ResourceName: "api-123",
				Reason:       "OOMKilled",
				Message:      "container api was OOMKilled",
				RiskTags:     []string{"oomkilled"},
			},
		},
	}

	if err := store.SaveTimeline(ctx, timeline); err != nil {
		t.Fatalf("save timeline: %v", err)
	}
	if err := store.SaveTimeline(ctx, timeline); err != nil {
		t.Fatalf("save deduped timeline: %v", err)
	}
}
