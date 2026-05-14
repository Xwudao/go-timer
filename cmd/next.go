package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var nextCmd = &cobra.Command{
	Use:   "next <name>",
	Short: "Show the next scheduled trigger time for a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveJobName(args[0])

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		mustJob(cfg, name)
		mgr := newManager()

		next, err := mgr.NextTrigger(name)
		if err != nil {
			return fmt.Errorf("querying next trigger: %w", err)
		}

		ui.Print("%-20s  %s", name, next)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(nextCmd)
}
