package cmd

import (
	"fmt"
	"os/exec"
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
	Long:  "Runs a series of environment checks and reports any issues.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Header("timerd doctor")
		ui.Separator()

		ok := color.New(color.FgGreen).Sprint("✔")
		warn := color.New(color.FgYellow).Sprint("⚠")
		fail := color.New(color.FgRed).Sprint("✘")

		check := func(label string, pass bool, note string) {
			mark := ok
			if !pass {
				mark = fail
			}
			if note != "" {
				fmt.Printf("  %s  %-40s %s\n", mark, label, color.New(color.FgHiBlack).Sprint(note))
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
				fmt.Printf("  %s  %-40s %s\n", mark, label, color.New(color.FgHiBlack).Sprint(note))
			} else {
				fmt.Printf("  %s  %s\n", mark, label)
			}
		}

		fmt.Println()
		ui.Dim("  OS / Platform")
		check("Linux runtime", runtime.GOOS == "linux", runtime.GOOS)
		isWSL := systemd.IsWSL()
		if isWSL {
			fmt.Printf("  %s  %-40s %s\n", warn, "WSL detected",
				color.New(color.FgYellow).Sprint("systemd user mode may need extra setup"))
		} else {
			check("Not WSL", true, "")
		}

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

		fmt.Println()
		ui.Dim("  Current mode: " + modeLabel())
		if isUserMode() {
			u, _ := currentUsername()
			lingerOn := systemd.IsLingerEnabled(u)
			if !lingerOn {
				fmt.Printf("  %s  %-40s %s\n", warn, "Linger enabled",
					color.New(color.FgYellow).Sprint("run: loginctl enable-linger "+u))
			} else {
				check("Linger enabled (user services survive logout)", true, "")
			}
		} else {
			isRoot := systemd.IsRoot()
			check("Running as root (required for --system)", isRoot, "")
		}

		fmt.Println()
		ui.Dim("  Config")
		cfgPath := configPath()
		cfg, err := loadConfigSilent()
		if err != nil {
			check("config.yml readable at "+cfgPath, false, err.Error())
		} else {
			check("config.yml readable", true, cfgPath)
			fmt.Printf("  %s  %-40s %s\n", ok, "jobs defined",
				color.New(color.FgHiBlack).Sprintf("%d", len(cfg.Jobs)))
		}

		fmt.Println()
		ui.Dim("  Tools")
		for _, bin := range []string{"journalctl", "loginctl"} {
			_, binErr := exec.LookPath(bin)
			checkWarn(bin+" on PATH", binErr == nil, "")
		}

		fmt.Println()

		if !hasSd || !hasCtl {
			ui.Warn("systemd is not available on this system")
			if isWSL {
				ui.Info("WSL tip: enable systemd in /etc/wsl.conf:\n  [boot]\n  systemd=true")
			}
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
func loadConfigSilent() (*struct{ Jobs map[string]interface{} }, error) {
	cfg, err := loadConfigErr()
	if err != nil {
		return nil, err
	}
	jobs := make(map[string]interface{})
	for k := range cfg.Jobs {
		jobs[k] = struct{}{}
	}
	return &struct{ Jobs map[string]interface{} }{Jobs: jobs}, nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
