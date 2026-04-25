package prometheus

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadTimelineEventsFromAlertEnvelope(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alerts.json")
	data := `{"alerts":[{"status":"firing","startsAt":"2026-04-25T14:10:00Z","labels":{"alertname":"HighLatency","severity":"critical","namespace":"prod","service":"api"},"annotations":{"summary":"API latency is high"}}]}`
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
	if events[0].Source != "prometheus" || events[0].Severity != "critical" || events[0].Service != "api" {
		t.Fatalf("unexpected normalized event: %+v", events[0])
	}
}
