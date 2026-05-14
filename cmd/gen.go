package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var genCmd = &cobra.Command{
	Use:   "gen [name]",
	Short: "Generate unit files but do not install them",
	Long:  "Prints the generated .service and .timer content to stdout. Does not write any files.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		gen := newGenerator()

		genOne := func(name string) error {
			job, err := cfg.GetJob(name)
			if err != nil {
				return err
			}
			pair, err := gen.Generate(name, job, isUserMode())
			if err != nil {
				return fmt.Errorf("generating units for %q: %w", name, err)
			}
			ui.Header(fmt.Sprintf("── %s ──", pair.ServiceName))
			ui.Print("%s", pair.Service)
			ui.Header(fmt.Sprintf("── %s ──", pair.TimerName))
			ui.Print("%s", pair.Timer)
			return nil
		}

		if len(args) == 1 {
			return genOne(args[0])
		}

		for name := range cfg.Jobs {
			if err := genOne(name); err != nil {
				ui.Error("%v", err)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(genCmd)
}
