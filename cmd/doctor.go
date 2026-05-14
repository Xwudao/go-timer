package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/systemd"
	"github.com/Xwudao/go-timer/internal/ui"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system compatibility and configuration",
	Long:  "Runs a comprehensive series of environment checks and reports issues.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Header("timerd doctor")
		ui.Separator()

		ok := color.New(color.FgGreen, color.Bold).Sprint("✔")
		warn := color.New(color.FgYellow, color.Bold).Sprint("⚠")
		fail := color.New(color.FgRed, color.Bold).Sprint("✘")
		dim := color.New(color.FgHiBlack)

		check := func(label string, pass bool, note string) {
			mark := ok
			if !pass {
				mark = fail
			}
			if note != "" {
				fmt.Printf("  %s  %-44s %s\n", mark, label, dim.Sprint(note))
			} else {
				fmt.Printf("  %s  %s\n", mark, label)
			}
		}
		checkWarn := func(label string, pass bool, note string) {
			mark := ok
			if !pass {
				mark = warn
			}
			if note != "" {
				fmt.Printf("  %s  %-44s %s\n", mark, label, dim.Sprint(note))
			} else {
				fmt.Printf("  %s  %s\n", mark, label)
			}
		}

		// ── OS / Platform ───────────────────────────────────────────────────
		fmt.Println()
		ui.Dim("  OS / Platform")

		check("Linux runtime", runtime.GOOS == "linux", runtime.GOOS)

		isWSL := systemd.IsWSL()
		if isWSL {
			fmt.Printf("  %s  %-44s %s\n", warn, "WSL detected",
				color.New(color.FgYellow).Sprint("extra setup may be needed — see README"))
		} else {
			check("Not running inside WSL", true, "")
		}

		if runtime.GOOS == "linux" {
			osr := systemd.ReadOsRelease()
			distroLabel := osr.Name
			if osr.Version != "" {
				distroLabel += " " + osr.Version
			}
			check("Linux distribution detected", osr.Name != "Unknown", distroLabel)
		}

		// ── systemd ──────────────────────────────────────────────────────────
		fmt.Println()
		ui.Dim("  systemd")

		hasSd := systemd.IsSystemdAvailable()
		check("systemd available", hasSd, "")

		hasCtl := systemd.IsSystemctlAvailable()
		check("systemctl on PATH", hasCtl, "")

		if hasCtl {
			userAvail := systemd.IsUserModeAvailable()
			checkWarn("systemctl --user works", userAvail, "")
		}

		if isUserMode() {
			dbusOK := systemd.IsDBusUserSessionAvailable()
			checkWarn("systemd user D-Bus session socket exists", dbusOK,
				fmt.Sprintf("/run/user/%d/bus", os.Getuid()))
		}

		// ── User mode / linger ───────────────────────────────────────────────
		fmt.Println()
		ui.Dim("%s", "  Mode: "+modeLabel())

		if isUserMode() {
			u, _ := currentUsername()
			lingerOn := systemd.IsLingerEnabled(u)
			if !lingerOn {
				fmt.Printf("  %s  %-44s %s\n", warn, "Linger enabled",
					color.New(color.FgYellow).Sprintf("run: loginctl enable-linger %s", u))
			} else {
				check("Linger enabled (services survive logout)", true, "")
			}
		} else {
			isRoot := systemd.IsRoot()
			check("Running as root (required for --system)", isRoot, "")
		}

		// ── XDG / config directories ─────────────────────────────────────────
		fmt.Println()
		ui.Dim("  Directories")

		cfgPath := configPath()
		ud := unitDir()

		check("config file readable", fileReadable(cfgPath), cfgPath)
		checkWarn("unit directory writable", systemd.UnitDirPermissionsOK(ud), ud)

		xdgRuntime := fmt.Sprintf("/run/user/%d", os.Getuid())
		if isUserMode() {
			_, xdgErr := os.Stat(xdgRuntime)
			checkWarn("XDG_RUNTIME_DIR exists", xdgErr == nil, xdgRuntime)
		}

		// ── Config / jobs ────────────────────────────────────────────────────
		fmt.Println()
		ui.Dim("  Config")

		cfg, cfgErr := loadConfigErr()
		if cfgErr != nil {
			check("config.yml parseable", false, cfgErr.Error())
		} else {
			check("config.yml parseable", true, cfgPath)
			fmt.Printf("  %s  %-44s %s\n", ok, "jobs defined",
				dim.Sprintf("%d job(s)", len(cfg.Jobs)))
		}

		// ── Failed / broken units ─────────────────────────────────────────────
		fmt.Println()
		ui.Dim("  Unit health")

		failed := systemd.ListFailedUnits(isUserMode())
		if len(failed) == 0 {
			check("No failed timerd units", true, "")
		} else {
			for _, u := range failed {
				fmt.Printf("  %s  %-44s %s\n", fail, "failed unit: "+u,
					dim.Sprint("run: timerd status <name>"))
			}
		}

		// ── Shell / environment ──────────────────────────────────────────────
		fmt.Println()
		ui.Dim("  Shell / Environment")

		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "(not set)"
		}
		check("SHELL set", shell != "(not set)", shell)

		pathVal := os.Getenv("PATH")
		pathOK := pathVal != ""
		checkWarn("PATH non-empty", pathOK, fmt.Sprintf("%d entries", countPathEntries(pathVal)))

		// ── Required tools ───────────────────────────────────────────────────
		fmt.Println()
		ui.Dim("  Required tools")

		for _, bin := range []string{"systemctl", "journalctl", "loginctl"} {
			p, binErr := exec.LookPath(bin)
			hint := ""
			if binErr == nil {
				hint = p
			}
			checkWarn(bin+" on PATH", binErr == nil, hint)
		}

		// ── Summary ──────────────────────────────────────────────────────────
		fmt.Println()

		if !hasSd || !hasCtl {
			ui.Warn("systemd is not available — timerd requires systemd")
			if isWSL {
				ui.Info("WSL tip: enable systemd in /etc/wsl.conf:\n  [boot]\n  systemd=true")
			}
		} else if runtime.GOOS != "linux" {
			ui.Warn("timerd is designed for Linux; some features may not work on %s", runtime.GOOS)
		} else {
			ui.Success("system looks healthy")
		}

		return nil
	},
}

func currentUsername() (string, error) {
	out, err := exec.Command("id", "-un").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// loadConfigSilent loads the config without printing errors (used in doctor).
func loadConfigSilent() (*struct{ Jobs map[string]any }, error) {
	cfg, err := loadConfigErr()
	if err != nil {
		return nil, err
	}
	jobs := make(map[string]any)
	for k := range cfg.Jobs {
		jobs[k] = struct{}{}
	}
	return &struct{ Jobs map[string]any }{Jobs: jobs}, nil
}

func fileReadable(path string) bool {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func countPathEntries(path string) int {
	if path == "" {
		return 0
	}
	return len(strings.Split(path, ":"))
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
