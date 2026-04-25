package argocd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadTimelineEventsFromApplicationList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "argocd.json")
	data := `[{"app":"api","time":"2026-04-25T14:08:00Z","revision":"abc123","syncStatus":"Synced","healthStatus":"Healthy","operationPhase":"Succeeded","destinationNamespace":"prod"}]`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	events, err := LoadTimelineEvents(path, time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC), time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Source != "argocd" || events[0].Category != "change" || events[0].Service != "api" {
		t.Fatalf("unexpected normalized event: %+v", events[0])
	}
}
