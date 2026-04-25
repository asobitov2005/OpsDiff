//go:build !cgo

package store

import (
	"context"
	"fmt"

	"github.com/asobitov2005/OpsDiff/internal/model"
)

type SQLiteStore struct{}

func OpenSQLite(path string) (*SQLiteStore, error) {
	return nil, fmt.Errorf("sqlite watch mode requires a cgo-enabled build")
}

func (s *SQLiteStore) Close() error {
	return nil
}

func (s *SQLiteStore) SaveSnapshot(ctx context.Context, snapshot model.Snapshot, path string) error {
	return fmt.Errorf("sqlite watch mode requires a cgo-enabled build")
}

func (s *SQLiteStore) LatestSnapshot(ctx context.Context, namespace string) (*SnapshotRecord, error) {
	return nil, fmt.Errorf("sqlite watch mode requires a cgo-enabled build")
}

func (s *SQLiteStore) SaveTimeline(ctx context.Context, timeline model.Timeline) error {
	return fmt.Errorf("sqlite watch mode requires a cgo-enabled build")
}
