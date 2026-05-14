package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/systemd"
)

var (
	logsFollow  bool
	logsLines   int
	logsSince   string
	logsTimer   bool
	logsService bool
)

var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show journal logs for a job",
	Long: `Show systemd journal logs for a job's service (default) or timer unit.

Examples:
  timerd logs backup              # last 50 lines
  timerd logs backup -f           # follow
  timerd logs backup -n 100       # last 100 lines
  timerd logs backup --since "1h" # last hour
  timerd logs backup --timer      # timer unit logs instead of service`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveJobName(args[0])

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}
		mustJob(cfg, name) // validate name

		// Decide which unit to follow.
		var unit string
		switch {
		case logsTimer:
			unit = systemd.TimerFileName(name)
		default: // logsService or default
			unit = systemd.ServiceFileName(name)
		}

		jArgs := []string{}
		if isUserMode() {
			jArgs = append(jArgs, "--user")
		}
		jArgs = append(jArgs, "-u", unit)
		jArgs = append(jArgs, "-n", strconv.Itoa(logsLines))

		if logsSince != "" {
			jArgs = append(jArgs, "--since", logsSince)
		}
		if logsFollow {
			jArgs = append(jArgs, "--follow")
		}
		jArgs = append(jArgs, "--no-pager")

		c := exec.Command("journalctl", jArgs...) //nolint:gosec
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin

		if err := c.Run(); err != nil {
			// journalctl exits non-zero on empty log or missing unit; treat as soft error.
			return nil
		}
		return nil
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output (like tail -f)")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "Number of recent lines to show")
	logsCmd.Flags().StringVar(&logsSince, "since", "", `Show logs since this time, e.g. "1h", "2024-01-01", "yesterday"`)
	logsCmd.Flags().BoolVar(&logsTimer, "timer", false, "Show timer unit logs instead of service logs")
	logsCmd.Flags().BoolVar(&logsService, "service", false, "Show service unit logs (default)")
	rootCmd.AddCommand(logsCmd)
}
