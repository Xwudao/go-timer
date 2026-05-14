package systemd_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Xwudao/go-timer/internal/config"
	"github.com/Xwudao/go-timer/internal/systemd"
)

func TestUnitName(t *testing.T) {
	if got := systemd.UnitName("backup"); got != "timerd-backup" {
		t.Errorf("UnitName = %q, want timerd-backup", got)
	}
}

func TestServiceFileName(t *testing.T) {
	if got := systemd.ServiceFileName("backup"); got != "timerd-backup.service" {
		t.Errorf("ServiceFileName = %q, want timerd-backup.service", got)
	}
}

func TestTimerFileName(t *testing.T) {
	if got := systemd.TimerFileName("backup"); got != "timerd-backup.timer" {
		t.Errorf("TimerFileName = %q, want timerd-backup.timer", got)
	}
}

func TestGenerate_Basic(t *testing.T) {
	gen := systemd.NewGenerator("")
	job := &config.JobConfig{
		Command:     "/home/tim/scripts/backup.sh",
		Schedule:    "hourly",
		Description: "Backup task",
		WorkDir:     "/home/tim/scripts",
	}

	pair, err := gen.Generate("backup", job, true)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Service checks
	if !strings.Contains(pair.Service, "ExecStart=/home/tim/scripts/backup.sh") {
		t.Errorf("service missing ExecStart:\n%s", pair.Service)
	}
	if !strings.Contains(pair.Service, "WorkingDirectory=/home/tim/scripts") {
		t.Errorf("service missing WorkingDirectory:\n%s", pair.Service)
	}
	if !strings.Contains(pair.Service, "default.target") {
		t.Errorf("service should use default.target in user mode:\n%s", pair.Service)
	}
	if !strings.Contains(pair.Service, "Backup task") {
		t.Errorf("service missing description:\n%s", pair.Service)
	}

	// Timer checks
	if !strings.Contains(pair.Timer, "OnCalendar=hourly") {
		t.Errorf("timer missing OnCalendar:\n%s", pair.Timer)
	}
	if !strings.Contains(pair.Timer, "timerd-backup.service") {
		t.Errorf("timer missing Unit reference:\n%s", pair.Timer)
	}
	if !strings.Contains(pair.Timer, "WantedBy=timers.target") {
		t.Errorf("timer missing WantedBy:\n%s", pair.Timer)
	}
}

func TestGenerate_SystemMode(t *testing.T) {
	gen := systemd.NewGenerator("")
	job := &config.JobConfig{
		Command:  "/usr/local/bin/sync",
		Schedule: "daily",
	}

	pair, err := gen.Generate("sync", job, false) // system mode
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(pair.Service, "multi-user.target") {
		t.Errorf("system mode should use multi-user.target:\n%s", pair.Service)
	}
}

func TestGenerate_WithCronSchedule(t *testing.T) {
	gen := systemd.NewGenerator("")
	job := &config.JobConfig{
		Command:  "/usr/bin/python3 sync.py",
		Schedule: "*/5 * * * *",
	}

	pair, err := gen.Generate("sync", job, true)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(pair.Timer, "*:0/5:00") {
		t.Errorf("timer should have converted cron schedule:\n%s", pair.Timer)
	}
}

func TestGenerate_WithEnv(t *testing.T) {
	gen := systemd.NewGenerator("")
	job := &config.JobConfig{
		Command:  "/usr/bin/app",
		Schedule: "daily",
		Env:      map[string]string{"APP_ENV": "production"},
	}

	pair, err := gen.Generate("app", job, true)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(pair.Service, `Environment="APP_ENV=production"`) {
		t.Errorf("service missing Environment:\n%s", pair.Service)
	}
}

func TestGenerate_WithArgs(t *testing.T) {
	gen := systemd.NewGenerator("")
	job := &config.JobConfig{
		Command:  "/usr/bin/rsync",
		Args:     []string{"-avz", "/src/", "/dst/"},
		Schedule: "daily",
	}

	pair, err := gen.Generate("rsync", job, true)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(pair.Service, "/usr/bin/rsync -avz /src/ /dst/") {
		t.Errorf("service ExecStart incorrect:\n%s", pair.Service)
	}
}

func TestGenerate_InvalidSchedule(t *testing.T) {
	gen := systemd.NewGenerator("")
	job := &config.JobConfig{
		Command:  "/bin/true",
		Schedule: "not-valid-cron",
	}
	_, err := gen.Generate("bad", job, true)
	if err == nil {
		t.Fatal("expected error for invalid schedule, got nil")
	}
}

func TestInstallAndRemove(t *testing.T) {
	dir := t.TempDir()
	gen := systemd.NewGenerator("")
	job := &config.JobConfig{
		Command:  "/bin/true",
		Schedule: "daily",
	}

	pair, err := gen.Install("mytest", job, dir, true)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Files should exist
	checkFileExists(t, dir, pair.ServiceName)
	checkFileExists(t, dir, pair.TimerName)

	// Remove
	if err := gen.Remove("mytest", dir); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Files should be gone
	checkFileNotExists(t, dir, pair.ServiceName)
	checkFileNotExists(t, dir, pair.TimerName)
}

func checkFileExists(t *testing.T, dir, name string) {
	t.Helper()
	path := dir + "/" + name
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file %s to exist: %v", path, err)
	}
}

func checkFileNotExists(t *testing.T, dir, name string) {
	t.Helper()
	path := dir + "/" + name
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected file %s to not exist after removal", path)
	}
}
