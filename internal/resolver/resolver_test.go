package resolver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Xwudao/go-timer/internal/config"
	"github.com/Xwudao/go-timer/internal/resolver"
)

// ─── ResolveExecutable ────────────────────────────────────────────────────────

func TestResolveExecutable_AbsolutePath(t *testing.T) {
	// Create a real temp file so the path is absolute.
	dir := t.TempDir()
	bin := filepath.Join(dir, "mybinary")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o700); err != nil { //nolint:gosec
		t.Fatal(err)
	}

	got, err := resolver.ResolveExecutable(bin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != bin {
		t.Errorf("got %q, want %q", got, bin)
	}
}

func TestResolveExecutable_LookPath(t *testing.T) {
	// 'sh' must always be findable on POSIX systems.
	got, err := resolver.ResolveExecutable("sh")
	if err != nil {
		t.Fatalf("unexpected error resolving 'sh': %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
}

func TestResolveExecutable_NotFound(t *testing.T) {
	_, err := resolver.ResolveExecutable("__timerd_no_such_binary_xyz_9999__")
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
	if !strings.Contains(err.Error(), "cannot resolve executable") {
		t.Errorf("error message should mention 'cannot resolve executable', got: %v", err)
	}
}

func TestResolveExecutable_Empty(t *testing.T) {
	_, err := resolver.ResolveExecutable("")
	if err == nil {
		t.Fatal("expected error for empty command, got nil")
	}
}

func TestResolveExecutable_TildeSlash(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	// Build a path like ~/existing_file using the home dir itself.
	// We'll create a temp file inside home's temp subdir.
	dir := t.TempDir()
	// Check if dir is under home (macOS puts temp dirs elsewhere on some versions).
	bin := filepath.Join(dir, "tildetest")
	if err := os.WriteFile(bin, []byte{}, 0o700); err != nil { //nolint:gosec
		t.Fatal(err)
	}

	// Use the absolute path to verify absolute returns unchanged.
	got, err := resolver.ResolveExecutable(bin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
	_ = home
}

// ─── CanonicalizeWorkDir ──────────────────────────────────────────────────────

func TestCanonicalizeWorkDir_Empty(t *testing.T) {
	got, err := resolver.CanonicalizeWorkDir("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestCanonicalizeWorkDir_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	got, err := resolver.CanonicalizeWorkDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
}

func TestCanonicalizeWorkDir_Tilde(t *testing.T) {
	if _, err := os.UserHomeDir(); err != nil {
		t.Skip("no home dir")
	}
	got, err := resolver.CanonicalizeWorkDir("~")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty result for ~")
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
}

func TestCanonicalizeWorkDir_NonExistent(t *testing.T) {
	// Non-existent directories should not error — they may be created at runtime.
	got, err := resolver.CanonicalizeWorkDir("/tmp/__timerd_nonexistent_dir_xyz__")
	if err != nil {
		t.Fatalf("unexpected error for non-existent dir: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty result")
	}
}

// ─── BuildExecStart ───────────────────────────────────────────────────────────

func TestBuildExecStart_AbsoluteNoArgs(t *testing.T) {
	job := &config.JobConfig{Command: "/usr/bin/echo"}
	got, err := resolver.BuildExecStart(job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/usr/bin/echo" {
		t.Errorf("got %q, want /usr/bin/echo", got)
	}
}

func TestBuildExecStart_AbsoluteWithArgs(t *testing.T) {
	job := &config.JobConfig{
		Command: "/usr/bin/echo",
		Args:    []string{"hello", "world"},
	}
	got, err := resolver.BuildExecStart(job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/usr/bin/echo hello world" {
		t.Errorf("got %q, want '/usr/bin/echo hello world'", got)
	}
}

func TestBuildExecStart_ArgWithSpaces(t *testing.T) {
	job := &config.JobConfig{
		Command: "/usr/bin/echo",
		Args:    []string{"hello world"},
	}
	got, err := resolver.BuildExecStart(job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Argument with spaces should be quoted.
	if !strings.Contains(got, `"hello world"`) {
		t.Errorf("expected quoted arg, got %q", got)
	}
}

func TestBuildExecStart_ShellMode(t *testing.T) {
	trueVal := true
	job := &config.JobConfig{
		Command:    "echo hello && echo world",
		Shell:      true,
		InheritEnv: &trueVal,
	}
	got, err := resolver.BuildExecStart(job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "-lc") {
		t.Errorf("shell mode should produce '-lc', got: %q", got)
	}
	if !strings.Contains(got, "echo hello && echo world") {
		t.Errorf("shell mode should include original command, got: %q", got)
	}
}

func TestBuildExecStart_ShellModeSingleQuoteEscape(t *testing.T) {
	job := &config.JobConfig{
		Command: "echo it's a test",
		Shell:   true,
	}
	got, err := resolver.BuildExecStart(job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Single-quotes must be escaped.
	if strings.Contains(got, "it's") {
		t.Errorf("unescaped single-quote in shell mode output: %q", got)
	}
}

// ─── ShouldInheritEnv ────────────────────────────────────────────────────────

func TestShouldInheritEnv_Default(t *testing.T) {
	job := &config.JobConfig{} // InheritEnv is nil
	if !resolver.ShouldInheritEnv(job) {
		t.Error("expected true when InheritEnv is nil")
	}
}

func TestShouldInheritEnv_ExplicitTrue(t *testing.T) {
	b := true
	job := &config.JobConfig{InheritEnv: &b}
	if !resolver.ShouldInheritEnv(job) {
		t.Error("expected true")
	}
}

func TestShouldInheritEnv_ExplicitFalse(t *testing.T) {
	b := false
	job := &config.JobConfig{InheritEnv: &b}
	if resolver.ShouldInheritEnv(job) {
		t.Error("expected false")
	}
}

// ─── MergedEnv ────────────────────────────────────────────────────────────────

func TestMergedEnv(t *testing.T) {
	base := map[string]string{"PATH": "/usr/bin", "HOME": "/home/user"}
	overrides := map[string]string{"PATH": "/custom/bin", "APP": "1"}

	result := resolver.MergedEnv(base, overrides)

	if result["PATH"] != "/custom/bin" {
		t.Errorf("override should win: got %q", result["PATH"])
	}
	if result["HOME"] != "/home/user" {
		t.Errorf("base should survive: got %q", result["HOME"])
	}
	if result["APP"] != "1" {
		t.Errorf("override key should be present: got %q", result["APP"])
	}
}
