package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var enableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a timer to start on boot / login",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveJobName(args[0])

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		job := mustJob(cfg, name)
		mgr := newManager()

		if err := mgr.Enable(name); err != nil {
			return fmt.Errorf("enabling timer: %w", err)
		}

		job.Enabled = true
		_ = cfg.UpdateJob(name, job)
		_ = saveConfig(cfg)

		ui.Success("timer %q enabled", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(enableCmd)
}
