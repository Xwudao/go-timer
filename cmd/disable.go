package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var disableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a timer (do not start on boot / login)",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		name := resolveJobName(args[0])

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		job := mustJob(cfg, name)
		mgr := newManager()

		if err := mgr.Disable(name); err != nil {
			return fmt.Errorf("disabling timer: %w", err)
		}

		job.Enabled = false
		_ = cfg.UpdateJob(name, job)
		_ = saveConfig(cfg)

		ui.Success("timer %q disabled", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(disableCmd)
}
