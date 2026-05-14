package cmd

import (
	"path/filepath"
	"testing"
)

func TestNewAutoRegisteredJob_SetsCommandWorkDirAndDefaultSchedule(t *testing.T) {
	job, err := newAutoRegisteredJob("./lz-gen-tag.sh", "")
	if err != nil {
		t.Fatalf("newAutoRegisteredJob returned error: %v", err)
	}

	wantCommand := filepath.Join(mustAbs(t, "."), "lz-gen-tag.sh")
	if job.Command != wantCommand {
		t.Fatalf("command = %q, want %q", job.Command, wantCommand)
	}

	wantWorkDir := filepath.Dir(wantCommand)
	if job.WorkDir != wantWorkDir {
		t.Fatalf("workdir = %q, want %q", job.WorkDir, wantWorkDir)
	}

	if job.Schedule != "daily" {
		t.Fatalf("schedule = %q, want daily", job.Schedule)
	}
}

func TestNewAutoRegisteredJob_PreservesExplicitSchedule(t *testing.T) {
	job, err := newAutoRegisteredJob("./lz-gen-tag.sh", "0 3 * * *")
	if err != nil {
		t.Fatalf("newAutoRegisteredJob returned error: %v", err)
	}

	if job.Schedule != "0 3 * * *" {
		t.Fatalf("schedule = %q, want explicit schedule", job.Schedule)
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()

	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", path, err)
	}

	return absPath
}
