package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

func WriteSnapshot(path string, snapshot model.Snapshot) error {
	if path == "" {
		return fmt.Errorf("snapshot output path is required")
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}

	return nil
}

func ReadSnapshot(path string) (model.Snapshot, error) {
	var snapshot model.Snapshot

	data, err := os.ReadFile(path)
	if err != nil {
		return snapshot, fmt.Errorf("read snapshot %q: %w", path, err)
	}

	if err := json.Unmarshal(data, &snapshot); err != nil {
		return snapshot, fmt.Errorf("decode snapshot %q: %w", path, err)
	}

	return snapshot, nil
}
