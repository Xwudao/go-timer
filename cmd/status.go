package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show systemd status for a job (or all jobs)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		mgr := newManager()

		if len(args) == 1 {
			name := resolveJobName(args[0])
			mustJob(cfg, name)
			out, err := mgr.Status(name)
			if err != nil {
				return fmt.Errorf("getting status: %w", err)
			}
			fmt.Fprintln(os.Stdout, out)
			return nil
		}

		// All jobs.
		if len(cfg.Jobs) == 0 {
			ui.Info("no jobs defined — run 'timerd add <name>'")
			return nil
		}

		for name := range cfg.Jobs {
			ui.Header(fmt.Sprintf("── %s ──────────────────────────────", name))
			out, _ := mgr.Status(name)
			fmt.Fprintln(os.Stdout, out)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
