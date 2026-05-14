package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <name>",
	Short: "Show full details: config, generated units, and systemd status",
	Args:  cobra.ExactArgs(1),
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
		ui.Header(fmt.Sprintf("── config.yml → jobs.%s ──────────────────", name))
		ui.Dim("  command    : %s", job.Command)
		if len(job.Args) > 0 {
			ui.Dim("  args       : %v", job.Args)
		}
		ui.Dim("  schedule   : %s", job.Schedule)
		if job.WorkDir != "" {
			ui.Dim("  workdir    : %s", job.WorkDir)
		}
		if job.Description != "" {
			ui.Dim("  description: %s", job.Description)
		}
		for k, v := range job.Env {
			ui.Dim("  env        : %s=%s", k, v)
		}

		// ── Generated units ──────────────────────────────────────────────────
		ui.Print("")
		pair, err := gen.Generate(name, job, isUserMode())
		if err != nil {
			ui.Warn("could not generate units: %v", err)
		} else {
			ui.Header(fmt.Sprintf("── %s ──────────────────", pair.ServiceName))
			ui.Print(pair.Service)
			ui.Header(fmt.Sprintf("── %s ──────────────────", pair.TimerName))
			ui.Print(pair.Timer)
		}

		// ── systemd status ───────────────────────────────────────────────────
		ui.Print("")
		ui.Header(fmt.Sprintf("── systemd status ──────────────────────"))
		out, _ := mgr.Status(name)
		ui.Print(out)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
