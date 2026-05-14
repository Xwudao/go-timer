package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/systemd"
	"github.com/Xwudao/go-timer/internal/ui"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <name>",
	Short: "Show full details: config, generated units, systemd status, and recent logs",
	Long: `Like 'docker inspect' or 'pm2 describe', shows everything about a job:
  - YAML configuration
  - Generated .service and .timer unit files
  - Live systemd status and key properties
  - Recent journal logs`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveJobName(args[0])

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		job := mustJob(cfg, name)
		gen := newGenerator()
		mgr := newManager()

		// ── YAML config ──────────────────────────────────────────────────────
		ui.Header(fmt.Sprintf("── config (%s) ──────────────────────────────", name))
		ui.Dim("  command     : %s", job.Command)
		if len(job.Args) > 0 {
			ui.Dim("  args        : %v", job.Args)
		}
		ui.Dim("  schedule    : %s", job.Schedule)
		if job.WorkDir != "" {
			ui.Dim("  workdir     : %s", job.WorkDir)
		}
		if job.Description != "" {
			ui.Dim("  description : %s", job.Description)
		}
		if job.Shell {
			ui.Dim("  shell       : true")
		}
		inheritStr := "true (default)"
		if job.InheritEnv != nil && !*job.InheritEnv {
			inheritStr = "false"
		}
		ui.Dim("  inherit_env : %s", inheritStr)
		if job.Restart != "" {
			ui.Dim("  restart     : %s", job.Restart)
		}
		if job.RestartSec != "" {
			ui.Dim("  restart_sec : %s", job.RestartSec)
		}
		if len(job.Env) > 0 {
			keys := make([]string, 0, len(job.Env))
			for k := range job.Env {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				ui.Dim("  env         : %s=%s", k, job.Env[k])
			}
		}

		// ── Generated units ──────────────────────────────────────────────────
		ui.Print("")
		pair, err := gen.Generate(name, job, isUserMode())
		if err != nil {
			ui.Warn("could not generate units: %v", err)
		} else {
			ui.Header(fmt.Sprintf("── %s ──────────────────", pair.ServiceName))
			ui.Print("%s", pair.Service)
			ui.Header(fmt.Sprintf("── %s ──────────────────", pair.TimerName))
			ui.Print("%s", pair.Timer)
		}

		// ── systemd status ───────────────────────────────────────────────────
		ui.Print("")
		ui.Header("── systemd status ──────────────────────────────")
		out, _ := mgr.Status(name)
		ui.Print("%s", out)

		// ── Key systemd properties ───────────────────────────────────────────
		ui.Header("── systemd properties ──────────────────────────")
		svcProps, err := mgr.ShowProperties(systemd.ServiceFileName(name))
		if err == nil {
			interesting := []string{
				"ActiveState", "SubState", "LoadState",
				"ExecMainStartTimestamp", "ExecMainExitTimestamp",
				"NRestarts", "Result",
			}
			for _, k := range interesting {
				if v, ok := svcProps[k]; ok && v != "" {
					ui.Dim("  %-30s %s", k, v)
				}
			}
		}
		tmrProps, err := mgr.ShowProperties(systemd.TimerFileName(name))
		if err == nil {
			for _, k := range []string{"ActiveState", "NextElapseUSecRealtime", "LastTriggerUSec"} {
				if v, ok := tmrProps[k]; ok && v != "" {
					ui.Dim("  %-30s %s", k, v)
				}
			}
		}

		// ── Recent logs ──────────────────────────────────────────────────────
		ui.Print("")
		ui.Header("── recent logs (last 20 lines) ─────────────────")
		jArgs := []string{}
		if isUserMode() {
			jArgs = append(jArgs, "--user")
		}
		jArgs = append(jArgs, "-u", systemd.ServiceFileName(name), "-n", "20", "--no-pager")
		c := exec.Command("journalctl", jArgs...) //nolint:gosec
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Run() //nolint:errcheck

		return nil
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
