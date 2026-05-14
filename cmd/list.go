package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all configured jobs and their status",
	Args:    cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		if len(cfg.Jobs) == 0 {
			ui.Info("no jobs defined — run 'timerd add <name>'")
			return nil
		}

		mgr := newManager()

		// Sort names for stable output.
		names := make([]string, 0, len(cfg.Jobs))
		for n := range cfg.Jobs {
			names = append(names, n)
		}
		sort.Strings(names)

		rows := make([][]string, 0, len(names))
		for _, name := range names {
			job := cfg.Jobs[name]

			status, _ := mgr.GetJobStatus(name)

			active := color.New(color.FgHiBlack).Sprint("○ inactive")
			if status != nil {
				active = formatActiveState(status.TimerActive, status.ServiceSubState)
			}

			enabled := color.New(color.FgHiBlack).Sprint("no")
			if mgr.IsEnabled(name) {
				enabled = color.New(color.FgGreen).Sprint("yes")
			}

			nextStr := color.New(color.FgHiBlack).Sprint("—")
			lastStr := color.New(color.FgHiBlack).Sprint("—")

			if status != nil {
				if !status.NextTriggerTime.IsZero() {
					left := time.Until(status.NextTriggerTime)
					nextStr = formatDuration(left)
				}
				if !status.LastTriggerTime.IsZero() {
					ago := time.Since(status.LastTriggerTime)
					lastStr = formatDuration(ago) + " ago"
				}
			}

			rows = append(rows, []string{
				name,
				active,
				enabled,
				truncate(job.Schedule, 22),
				nextStr,
				lastStr,
			})
		}

		ui.Print("")
		ui.Table(
			[]string{"Name", "Status", "Enabled", "Schedule", "Next", "Last"},
			rows,
		)
		ui.Print("")
		ui.Dim("  config : %s", configPath())
		ui.Dim("  mode   : %s", modeLabel())
		return nil
	},
}

func formatActiveState(timerActive, _ string) string {
	switch strings.ToLower(timerActive) {
	case "active":
		return ui.ActiveBadge("active")
	case "failed":
		return ui.ActiveBadge("failed")
	case "activating":
		return ui.ActiveBadge("activating")
	case "":
		return ui.ActiveBadge("inactive")
	default:
		return ui.ActiveBadge(timerActive)
	}
}

func modeLabel() string {
	if isUserMode() {
		return "--user"
	}
	return "--system"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func init() {
	rootCmd.AddCommand(listCmd)
}
