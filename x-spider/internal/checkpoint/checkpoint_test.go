package checkpoint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckpointManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cpPath := filepath.Join(tempDir, "cp.json")
	mgr := NewManager(cpPath)

	if mgr.Exists() {
		t.Fatalf("checkpoint should not exist initially")
	}

	state := State{
		Query:        "golang since:2026-01-01",
		TargetFile:   "tweets-data/test.csv",
		ExportFormat: "csv",
		LastTweetID:  "987654",
		TotalSaved:   45,
		TargetLimit:  100,
	}

	if err := mgr.Save(state); err != nil {
		t.Fatalf("failed to save checkpoint: %v", err)
	}

	if !mgr.Exists() {
		t.Fatalf("checkpoint should exist after saving")
	}

	loaded, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to load checkpoint: %v", err)
	}

	if loaded.LastTweetID != "987654" || loaded.TotalSaved != 45 {
		t.Errorf("unexpected loaded state: %+v", loaded)
	}

	if err := mgr.Clear(); err != nil {
		t.Fatalf("failed to clear checkpoint: %v", err)
	}

	if mgr.Exists() {
		t.Fatalf("checkpoint should not exist after clear")
	}
}
