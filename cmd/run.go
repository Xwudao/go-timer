package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var runCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Trigger a job's service immediately (one-shot)",
	Long:  "Runs the service unit directly via systemctl start, bypassing the timer.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveJobName(args[0])

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		mustJob(cfg, name)
		mgr := newManager()

		ui.Info("running %q …", name)
		if err := mgr.RunOnce(name); err != nil {
			return fmt.Errorf("running job: %w", err)
		}

		ui.Success("job %q triggered", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
