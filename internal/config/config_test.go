package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Xwudao/go-timer/internal/config"
)

func TestNewConfig(t *testing.T) {
	cfg := config.NewConfig()
	if cfg.Jobs == nil {
		t.Fatal("NewConfig().Jobs should not be nil")
	}
}

func TestAddAndGetJob(t *testing.T) {
	cfg := config.NewConfig()

	job := &config.JobConfig{
		Command:  "/bin/echo hello",
		Schedule: "daily",
	}

	if err := cfg.AddJob("test", job); err != nil {
		t.Fatalf("AddJob: %v", err)
	}

	got, err := cfg.GetJob("test")
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if got.Command != job.Command {
		t.Errorf("command = %q, want %q", got.Command, job.Command)
	}
}

func TestAddJob_AlreadyExists(t *testing.T) {
	cfg := config.NewConfig()
	job := &config.JobConfig{Command: "/bin/true", Schedule: "daily"}
	_ = cfg.AddJob("dup", job)
	if err := cfg.AddJob("dup", job); err == nil {
		t.Fatal("expected error on duplicate add, got nil")
	}
}

func TestRemoveJob(t *testing.T) {
	cfg := config.NewConfig()
	job := &config.JobConfig{Command: "/bin/true", Schedule: "daily"}
	_ = cfg.AddJob("removeme", job)

	if err := cfg.RemoveJob("removeme"); err != nil {
		t.Fatalf("RemoveJob: %v", err)
	}
	if _, err := cfg.GetJob("removeme"); err == nil {
		t.Fatal("expected error after removal, got nil")
	}
}

func TestRemoveJob_NotFound(t *testing.T) {
	cfg := config.NewConfig()
	if err := cfg.RemoveJob("ghost"); err == nil {
		t.Fatal("expected error for missing job, got nil")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	cfg := config.NewConfig()
	_ = cfg.AddJob("backup", &config.JobConfig{
		Command:     "/bin/backup.sh",
		Schedule:    "hourly",
		Description: "Backup job",
		Env:         map[string]string{"TOKEN": "abc"},
	})

	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not written: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	job, err := loaded.GetJob("backup")
	if err != nil {
		t.Fatalf("GetJob after load: %v", err)
	}
	if job.Command != "/bin/backup.sh" {
		t.Errorf("command = %q, want /bin/backup.sh", job.Command)
	}
	if job.Env["TOKEN"] != "abc" {
		t.Errorf("env TOKEN = %q, want abc", job.Env["TOKEN"])
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.yml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestValidate(t *testing.T) {
	cfg := config.NewConfig()
	_ = cfg.AddJob("missing-cmd", &config.JobConfig{Schedule: "daily"})
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for missing command")
	}

	cfg2 := config.NewConfig()
	_ = cfg2.AddJob("missing-sched", &config.JobConfig{Command: "/bin/true"})
	if err := cfg2.Validate(); err == nil {
		t.Fatal("expected validation error for missing schedule")
	}

	cfg3 := config.NewConfig()
	_ = cfg3.AddJob("valid", &config.JobConfig{Command: "/bin/true", Schedule: "daily"})
	if err := cfg3.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := config.DefaultConfigPath(true)
	if path == "" {
		t.Fatal("user mode path should not be empty")
	}

	sysPath := config.DefaultConfigPath(false)
	if sysPath != "/etc/timerd/config.yml" {
		t.Errorf("system path = %q, want /etc/timerd/config.yml", sysPath)
	}
}
