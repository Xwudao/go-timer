package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var nextAllFlag bool

var nextCmd = &cobra.Command{
	Use:     "next [name]",
	Aliases: []string{"timers"},
	Short:   "Show next/last trigger times for a job (or all jobs)",
	Long: `Display the next and last trigger times for one job or every configured job.

Examples:
  timerd next backup       # single job
  timerd next              # all jobs`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		mgr := newManager()

		var names []string
		if len(args) == 1 {
			name := resolveJobName(args[0])
			mustJob(cfg, name)
			names = []string{name}
		} else {
			for n := range cfg.Jobs {
				names = append(names, n)
			}
			sort.Strings(names)
		}

		now := time.Now()
		rows := make([][]string, 0, len(names))

		for _, name := range names {
			status, err := mgr.GetJobStatus(name)
			if err != nil {
				rows = append(rows, []string{name, "error", "—", "—"})
				continue
			}

			nextStr := "—"
			leftStr := "—"
			lastStr := "—"

			if !status.NextTriggerTime.IsZero() {
				nextStr = status.NextTriggerTime.Format("2006-01-02 15:04:05")
				left := time.Until(status.NextTriggerTime)
				leftStr = formatDuration(left)
			}

			if !status.LastTriggerTime.IsZero() {
				lastStr = status.LastTriggerTime.Format("2006-01-02 15:04:05")
			}

			_ = now
			rows = append(rows, []string{name, nextStr, leftStr, lastStr})
		}

		fmt.Println()
		ui.Table(
			[]string{"Name", "Next trigger", "In", "Last trigger"},
			rows,
		)
		fmt.Println()
		return nil
	},
}

// formatDuration returns a human-readable duration string (e.g. "2h 15m").
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "overdue"
	}
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

func init() {
	nextCmd.Flags().BoolVarP(&nextAllFlag, "all", "a", false, "Show all jobs (default when no name given)")
	rootCmd.AddCommand(nextCmd)
}
