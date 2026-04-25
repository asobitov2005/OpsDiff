package app

import (
	"os"
	"path/filepath"
)

func defaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".opsdiff"
	}
	return filepath.Join(home, ".opsdiff")
}

func defaultSQLitePath() string {
	return filepath.Join(defaultStateDir(), "opsdiff.db")
}

func defaultSnapshotDir() string {
	return filepath.Join(defaultStateDir(), "snapshots")
}
