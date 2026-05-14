package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/systemd"
	"github.com/Xwudao/go-timer/internal/ui"
)

var runFollow bool
var runLines int

var runCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Trigger a job's service immediately (one-shot) and optionally follow logs",
	Long: `Runs the service unit directly via systemctl start, bypassing the timer.

Like PM2's 'start', the job runs immediately. Use --follow to watch
the output in real time via journalctl.

Examples:
  timerd run backup              # trigger and exit
  timerd run backup --follow     # trigger then follow logs
  timerd run backup -f -n 100    # follow, show last 100 lines`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveJobName(args[0])

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		mustJob(cfg, name)
		mgr := newManager()

		ui.Info("triggering %q …", name)
		if err := mgr.RunOnce(name); err != nil {
			return fmt.Errorf("running job: %w", err)
		}

		ui.Success("job %q triggered", name)

		if runFollow {
			unit := systemd.ServiceFileName(name)
			jArgs := []string{}
			if isUserMode() {
				jArgs = append(jArgs, "--user")
			}
			jArgs = append(jArgs, "-u", unit, "-n", strconv.Itoa(runLines), "--follow", "--no-pager")

			c := exec.Command("journalctl", jArgs...) //nolint:gosec
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Stdin = os.Stdin
			c.Run() //nolint:errcheck
		}

		return nil
	},
}

func init() {
	runCmd.Flags().BoolVarP(&runFollow, "follow", "f", false, "Follow service logs after triggering")
	runCmd.Flags().IntVarP(&runLines, "lines", "n", 50, "Number of log lines to show when following")
	rootCmd.AddCommand(runCmd)
}
