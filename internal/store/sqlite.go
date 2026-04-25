//go:build cgo

package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/model"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) init() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TEXT NOT NULL,
			cluster TEXT NOT NULL,
			namespace TEXT NOT NULL,
			file_path TEXT NOT NULL,
			payload_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_namespace_created_at ON snapshots(namespace, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS timeline_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			collected_at TEXT NOT NULL,
			event_time TEXT NOT NULL,
			fingerprint TEXT NOT NULL UNIQUE,
			source TEXT NOT NULL,
			category TEXT NOT NULL,
			severity TEXT NOT NULL,
			namespace TEXT NOT NULL,
			service TEXT,
			resource_kind TEXT,
			resource_name TEXT,
			reason TEXT,
			message TEXT NOT NULL,
			risk_tags_json TEXT NOT NULL,
			payload_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_timeline_events_namespace_time ON timeline_events(namespace, event_time DESC)`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("initialize sqlite schema: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStore) SaveSnapshot(ctx context.Context, snapshot model.Snapshot, path string) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot for sqlite: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO snapshots(created_at, cluster, namespace, file_path, payload_json) VALUES(?, ?, ?, ?, ?)`,
		snapshot.CreatedAt.UTC().Format(time.RFC3339),
		snapshot.Cluster,
		snapshot.Namespace,
		path,
		string(payload),
	)
	if err != nil {
		return fmt.Errorf("insert snapshot into sqlite: %w", err)
	}

	return nil
}

func (s *SQLiteStore) LatestSnapshot(ctx context.Context, namespace string) (*SnapshotRecord, error) {
	query := `SELECT file_path, payload_json FROM snapshots`
	args := make([]any, 0, 1)
	if namespace != "" {
		query += ` WHERE namespace = ?`
		args = append(args, namespace)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT 1`

	row := s.db.QueryRowContext(ctx, query, args...)

	var (
		path    string
		payload string
	)
	if err := row.Scan(&path, &payload); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest snapshot: %w", err)
	}

	var snapshot model.Snapshot
	if err := json.Unmarshal([]byte(payload), &snapshot); err != nil {
		return nil, fmt.Errorf("decode latest snapshot: %w", err)
	}

	return &SnapshotRecord{
		Path:     path,
		Snapshot: snapshot,
	}, nil
}

func (s *SQLiteStore) SaveTimeline(ctx context.Context, timeline model.Timeline) error {
	transaction, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin timeline transaction: %w", err)
	}
	defer func() {
		if transaction != nil {
			_ = transaction.Rollback()
		}
	}()

	statement, err := transaction.PrepareContext(ctx, `INSERT OR IGNORE INTO timeline_events(
		collected_at, event_time, fingerprint, source, category, severity, namespace, service,
		resource_kind, resource_name, reason, message, risk_tags_json, payload_json
	) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare timeline insert: %w", err)
	}
	defer statement.Close()

	for _, event := range timeline.Events {
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal timeline event: %w", err)
		}
		riskTags, err := json.Marshal(event.RiskTags)
		if err != nil {
			return fmt.Errorf("marshal timeline event risk tags: %w", err)
		}

		if _, err := statement.ExecContext(
			ctx,
			timeline.GeneratedAt.UTC().Format(time.RFC3339),
			event.Time.UTC().Format(time.RFC3339),
			fingerprintEvent(event),
			event.Source,
			event.Category,
			event.Severity,
			event.Namespace,
			event.Service,
			event.ResourceKind,
			event.ResourceName,
			event.Reason,
			event.Message,
			string(riskTags),
			string(payload),
		); err != nil {
			return fmt.Errorf("insert timeline event: %w", err)
		}
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit timeline transaction: %w", err)
	}
	transaction = nil
	return nil
}

func fingerprintEvent(event model.TimelineEvent) string {
	sum := sha256.Sum256([]byte(
		event.Time.UTC().Format(time.RFC3339) + "|" +
			event.Source + "|" +
			event.Category + "|" +
			event.Severity + "|" +
			event.Namespace + "|" +
			event.Service + "|" +
			event.ResourceKind + "|" +
			event.ResourceName + "|" +
			event.Reason + "|" +
			event.Message,
	))
	return hex.EncodeToString(sum[:])
}
