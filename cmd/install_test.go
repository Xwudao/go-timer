package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallBinary_MovesBinaryToTargetPath(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "timerd-dev")
	targetPath := filepath.Join(tempDir, "bin", "timerd")

	const wantContent = "#!/bin/sh\necho timerd\n"
	if err := os.WriteFile(sourcePath, []byte(wantContent), 0o700); err != nil {
		t.Fatalf("os.WriteFile(source): %v", err)
	}

	if err := installBinary(sourcePath, targetPath); err != nil {
		t.Fatalf("installBinary returned error: %v", err)
	}

	gotContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("os.ReadFile(target): %v", err)
	}
	if string(gotContent) != wantContent {
		t.Fatalf("target content = %q, want %q", string(gotContent), wantContent)
	}

	if _, err := os.Stat(sourcePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source path still exists, stat err = %v", err)
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("os.Stat(target): %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("target mode = %v, want executable bit set", info.Mode().Perm())
	}
}

func TestInstallBinary_ReturnsAlreadyInstalledForSamePath(t *testing.T) {
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "timerd")

	if err := os.WriteFile(binaryPath, []byte("timerd"), 0o755); err != nil {
		t.Fatalf("os.WriteFile(binary): %v", err)
	}

	err := installBinary(binaryPath, binaryPath)
	if !errors.Is(err, errAlreadyInstalled) {
		t.Fatalf("installBinary error = %v, want %v", err, errAlreadyInstalled)
	}
}
