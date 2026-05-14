package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop the timer for a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		name := resolveJobName(args[0])

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		job := mustJob(cfg, name)
		mgr := newManager()

		if err := mgr.Stop(name); err != nil {
			return fmt.Errorf("stopping timer: %w", err)
		}

		// Update enabled state in config.
		job.Enabled = false
		_ = cfg.UpdateJob(name, job)
		_ = saveConfig(cfg)

		ui.Success("timer %q stopped", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
