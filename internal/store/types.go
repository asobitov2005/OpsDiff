package store

import "github.com/asobitov2005/OpsDiff/internal/model"

type SnapshotRecord struct {
	Path     string
	Snapshot model.Snapshot
}
