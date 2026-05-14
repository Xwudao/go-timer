// Package resolver provides runtime environment resolution for timerd.
//
// It resolves executable paths (bare names → absolute via PATH, ~ expansion,
// absolute paths trusted as-is), canonicalises working directories, and
// manages environment variable inheritance so that systemd units contain
// fully-qualified paths and a sane PATH — just like PM2 does on Node.
package resolver

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Xwudao/go-timer/internal/config"
)

// ResolveExecutable resolves a command string to its absolute executable path.
//
// Resolution order:
//  1. Expand a leading ~ to the user home directory.
//  2. If the result is an absolute path, return it directly (no stat check —
//     let systemd surface the error if the file is missing at runtime).
//  3. Otherwise search the current process PATH via exec.LookPath so the
//     same binary the user can run interactively is embedded in the unit.
func ResolveExecutable(command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("command is empty")
	}

	expanded, err := expandTilde(command)
	if err != nil {
		return "", fmt.Errorf("expanding path %q: %w", command, err)
	}

	if filepath.IsAbs(expanded) {
		return expanded, nil
	}

	abs, err := exec.LookPath(expanded)
	if err != nil {
		return "", fmt.Errorf(
			"cannot resolve executable %q: not found in PATH\n"+
				"  hint: use an absolute path or add its parent directory to PATH",
			expanded,
		)
	}
	return abs, nil
}

// CanonicalizeWorkDir converts a working directory to an absolute,
// symlink-resolved path. An empty string is returned unchanged.
// If EvalSymlinks fails (directory not yet created), the plain absolute
// path is returned without error.
func CanonicalizeWorkDir(dir string) (string, error) {
	if dir == "" {
		return "", nil
	}

	expanded, err := expandTilde(dir)
	if err != nil {
		return "", fmt.Errorf("expanding workdir %q: %w", dir, err)
	}

	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolving workdir %q: %w", dir, err)
	}

	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	return abs, nil
}

// InheritedPATH returns the PATH from the current process environment.
func InheritedPATH() string {
	return os.Getenv("PATH")
}

// ShouldInheritEnv reports whether the current process PATH should be
// injected into the unit's Environment= directives. Defaults to true
// when the job field is nil (not set in config).
func ShouldInheritEnv(job *config.JobConfig) bool {
	if job.InheritEnv == nil {
		return true
	}
	return *job.InheritEnv
}

// MergedEnv returns a new map that is the union of base and overrides,
// with overrides winning on key conflicts.
func MergedEnv(base, overrides map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(overrides))
	maps.Copy(out, base)
	maps.Copy(out, overrides)
	return out
}

// BuildExecStart constructs the ExecStart= value for a systemd service unit.
//
//   - shell == false (default): resolves command to an absolute path via
//     ResolveExecutable and appends any args.
//   - shell == true: wraps the full command string in bash -lc '...' so that
//     shell constructs (pipes, &&, variable expansion, etc.) work as expected.
func BuildExecStart(job *config.JobConfig) (string, error) {
	if job.Shell {
		return buildShellExecStart(job)
	}
	return buildDirectExecStart(job)
}

// ─── private helpers ─────────────────────────────────────────────────────────

func buildDirectExecStart(job *config.JobConfig) (string, error) {
	resolved, err := ResolveExecutable(job.Command)
	if err != nil {
		return "", err
	}

	if len(job.Args) == 0 {
		return resolved, nil
	}

	parts := make([]string, 0, 1+len(job.Args))
	parts = append(parts, resolved)
	for _, a := range job.Args {
		if strings.ContainsAny(a, " \t") {
			parts = append(parts, fmt.Sprintf("%q", a))
		} else {
			parts = append(parts, a)
		}
	}
	return strings.Join(parts, " "), nil
}

func buildShellExecStart(job *config.JobConfig) (string, error) {
	// Prefer bash; silently fall back to /bin/sh.
	shell, err := ResolveExecutable("bash")
	if err != nil {
		shell = "/bin/sh"
	}

	fullCmd := job.Command
	if len(job.Args) > 0 {
		fullCmd += " " + strings.Join(job.Args, " ")
	}

	// Escape embedded single-quotes for safe shell embedding.
	safe := strings.ReplaceAll(fullCmd, "'", "'\\''")
	return fmt.Sprintf("%s -lc '%s'", shell, safe), nil
}

func expandTilde(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
