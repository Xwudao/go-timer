package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var restartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Restart the timer for a job",
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

		// Reinstall units (in case config changed).
		if _, err := gen.Install(name, job, unitDir(), isUserMode()); err != nil {
			return fmt.Errorf("installing units: %w", err)
		}

		if err := mgr.DaemonReload(); err != nil {
			ui.Warn("daemon-reload: %v", err)
		}

		if err := mgr.Restart(name); err != nil {
			return fmt.Errorf("restarting timer: %w", err)
		}

		ui.Success("timer %q restarted", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
