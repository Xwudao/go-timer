package systemd

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

// TimerInfo holds parsed information about a systemd timer.
type TimerInfo struct {
	Name   string
	Next   time.Time
	Left   string
	Last   time.Time
	Passed string
	Unit   string
	Active bool
}

// JobStatus holds the status of a job's service and timer.
type JobStatus struct {
	ServiceActive   string
	ServiceState    string
	ServiceSubState string
	TimerActive     string
	TimerState      string
	LastTrigger     string
	NextTrigger     string
}

// Manager wraps systemctl to manage timerd units.
type Manager struct {
	UserMode bool
	DryRun   bool
	Verbose  bool
}

// NewManager creates a new Manager.
func NewManager(userMode, dryRun, verbose bool) *Manager {
	return &Manager{UserMode: userMode, DryRun: dryRun, Verbose: verbose}
}

// baseArgs returns the base systemctl arguments for the current mode.
func (m *Manager) baseArgs() []string {
	if m.UserMode {
		return []string{"--user"}
	}
	return nil
}

// run executes a systemctl command.
func (m *Manager) run(args ...string) (string, error) {
	full := append([]string{"systemctl"}, m.baseArgs()...)
	full = append(full, args...)

	if m.Verbose {
		slog.Debug("executing", "cmd", strings.Join(full, " "))
	}

	if m.DryRun {
		fmt.Printf("[dry-run] %s\n", strings.Join(full, " "))
		return "", nil
	}

	cmd := exec.Command(full[0], full[1:]...) //nolint:gosec
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// runOutput executes a command and returns its combined output even on error.
func (m *Manager) runOutput(args ...string) (string, error) {
	full := append([]string{"systemctl"}, m.baseArgs()...)
	full = append(full, args...)

	if m.Verbose {
		slog.Debug("executing", "cmd", strings.Join(full, " "))
	}

	cmd := exec.Command(full[0], full[1:]...) //nolint:gosec
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// DaemonReload runs systemctl daemon-reload.
func (m *Manager) DaemonReload() error {
	_, err := m.run("daemon-reload")
	return err
}

// Start starts a timer unit.
func (m *Manager) Start(name string) error {
	_, err := m.run("start", TimerFileName(name))
	return err
}

// Stop stops a timer unit.
func (m *Manager) Stop(name string) error {
	_, err := m.run("stop", TimerFileName(name))
	return err
}

// Restart restarts a timer unit.
func (m *Manager) Restart(name string) error {
	_, err := m.run("restart", TimerFileName(name))
	return err
}

// Enable enables a timer unit.
func (m *Manager) Enable(name string) error {
	_, err := m.run("enable", TimerFileName(name))
	return err
}

// Disable disables a timer unit.
func (m *Manager) Disable(name string) error {
	_, err := m.run("disable", TimerFileName(name))
	return err
}

// RunOnce triggers the service unit immediately (systemctl start .service).
func (m *Manager) RunOnce(name string) error {
	_, err := m.run("start", ServiceFileName(name))
	return err
}

// IsActive reports whether the timer unit is active.
func (m *Manager) IsActive(name string) bool {
	out, err := m.runOutput("is-active", TimerFileName(name))
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "active"
}

// IsEnabled reports whether the timer unit is enabled.
func (m *Manager) IsEnabled(name string) bool {
	out, err := m.runOutput("is-enabled", TimerFileName(name))
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "enabled"
}

// Status returns the human-readable status of both service and timer units.
func (m *Manager) Status(name string) (string, error) {
	out, _ := m.runOutput("status", TimerFileName(name), ServiceFileName(name))
	return out, nil
}

// ShowProperties returns the parsed properties of a unit.
func (m *Manager) ShowProperties(unitName string) (map[string]string, error) {
	full := append([]string{"systemctl"}, m.baseArgs()...)
	full = append(full, "show", unitName)

	if m.Verbose {
		slog.Debug("executing", "cmd", strings.Join(full, " "))
	}

	cmd := exec.Command(full[0], full[1:]...) //nolint:gosec
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("systemctl show %s: %w", unitName, err)
	}

	props := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		props[line[:idx]] = line[idx+1:]
	}
	return props, nil
}

// GetJobStatus returns a structured status for a job.
func (m *Manager) GetJobStatus(name string) (*JobStatus, error) {
	svcProps, err := m.ShowProperties(ServiceFileName(name))
	if err != nil {
		// Unit may not exist yet.
		return &JobStatus{ServiceActive: "not-found"}, nil
	}
	tmrProps, _ := m.ShowProperties(TimerFileName(name))

	js := &JobStatus{
		ServiceActive:   svcProps["ActiveState"],
		ServiceState:    svcProps["LoadState"],
		ServiceSubState: svcProps["SubState"],
	}
	if tmrProps != nil {
		js.TimerActive = tmrProps["ActiveState"]
		js.TimerState = tmrProps["LoadState"]
		js.NextTrigger = tmrProps["NextElapseUSecRealtime"]
		js.LastTrigger = tmrProps["LastTriggerUSec"]
	}
	return js, nil
}

// ListTimers returns all timerd-* timers visible to systemd.
func (m *Manager) ListTimers() ([]TimerInfo, error) {
	full := append([]string{"systemctl"}, m.baseArgs()...)
	full = append(full, "list-timers", "--all", "--no-legend", "timerd-*")

	cmd := exec.Command(full[0], full[1:]...) //nolint:gosec
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing timers: %w", err)
	}

	var timers []TimerInfo
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Output columns: NEXT LEFT LAST PASSED UNIT ACTIVATES
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		// The unit name is in the 5th field (index 4) in most systemd versions.
		// We look for the field ending in .timer.
		var unitName string
		for _, f := range fields {
			if strings.HasSuffix(f, ".timer") {
				unitName = f
				break
			}
		}
		if unitName == "" {
			continue
		}
		timers = append(timers, TimerInfo{
			Name: strings.TrimPrefix(strings.TrimSuffix(unitName, ".timer"), "timerd-"),
			Unit: unitName,
		})
	}
	return timers, nil
}

// NextTrigger returns the next trigger time string for a timer unit.
func (m *Manager) NextTrigger(name string) (string, error) {
	full := append([]string{"systemctl"}, m.baseArgs()...)
	full = append(full, "list-timers", "--all", "--no-legend", TimerFileName(name))

	cmd := exec.Command(full[0], full[1:]...) //nolint:gosec
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("listing timer: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if strings.Contains(line, TimerFileName(name)) {
			fields := strings.Fields(line)
			// Format: NEXT LEFT LAST PASSED UNIT ACTIVATES
			// "NEXT" is a datetime that spans multiple fields.
			// We reconstruct from fields before "left".
			var nextParts []string
			for i, f := range fields {
				if strings.Contains(strings.ToLower(f), "left") || strings.Contains(strings.ToLower(f), "ago") {
					// Everything before this was the next trigger time.
					if i > 0 {
						nextParts = fields[:i]
					}
					break
				}
			}
			if len(nextParts) > 0 {
				return strings.Join(nextParts, " "), nil
			}
			if len(fields) > 0 {
				return fields[0], nil
			}
		}
	}
	return "n/a", nil
}

// EnableLinger enables systemd linger for the current user so user services
// survive logout.
func EnableLinger(username string) error {
	if username == "" {
		u, err := currentUser()
		if err != nil {
			return err
		}
		username = u
	}
	cmd := exec.Command("loginctl", "enable-linger", username) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("loginctl enable-linger: %w: %s", err, string(out))
	}
	return nil
}

// IsLingerEnabled reports whether linger is enabled for the current user.
func IsLingerEnabled(username string) bool {
	if username == "" {
		u, err := currentUser()
		if err != nil {
			return false
		}
		username = u
	}
	path := fmt.Sprintf("/var/lib/systemd/linger/%s", username)
	_, err := os.Stat(path)
	return err == nil
}

// IsSystemdAvailable reports whether systemd is the init system.
func IsSystemdAvailable() bool {
	_, err := os.Stat("/run/systemd/private")
	if err == nil {
		return true
	}
	_, err = os.Stat("/sys/fs/cgroup/systemd")
	return err == nil
}

// IsSystemctlAvailable reports whether the systemctl binary is on PATH.
func IsSystemctlAvailable() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

// IsWSL reports whether we're running inside Windows Subsystem for Linux.
func IsWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}

// IsUserModeAvailable reports whether systemctl --user works.
func IsUserModeAvailable() bool {
	cmd := exec.Command("systemctl", "--user", "is-system-running") //nolint:gosec
	err := cmd.Run()
	// is-system-running exits 1 in degraded state but still works.
	return err == nil || (cmd.ProcessState != nil && cmd.ProcessState.ExitCode() < 4)
}

// IsRoot reports whether the process is running as root.
func IsRoot() bool {
	return os.Getuid() == 0
}

func currentUser() (string, error) {
	cmd := exec.Command("id", "-un")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting current user: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
