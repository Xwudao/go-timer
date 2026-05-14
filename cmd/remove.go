package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Stop, disable, and remove a job",
	Long: `Stops the timer, disables it, removes the generated unit files,
and deletes the job from config.yml.`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		name := resolveJobName(args[0])

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		if _, err := cfg.GetJob(name); err != nil {
			return fmt.Errorf("job %q not found", name)
		}

		if !removeForce {
			if !ui.Confirm(fmt.Sprintf("Remove job %q (stop, disable, delete unit files)?", name), false) {
				ui.Info("aborted")
				return nil
			}
		}

		mgr := newManager()

		// Stop + disable (ignore errors — unit may already be inactive).
		_ = mgr.Stop(name)
		_ = mgr.Disable(name)

		// Remove unit files.
		gen := newGenerator()
		if err := gen.Remove(name, unitDir()); err != nil {
			ui.Warn("removing unit files: %v", err)
		}

		_ = mgr.DaemonReload()

		// Remove from config.
		if err := cfg.RemoveJob(name); err != nil {
			return err
		}

		if !saveConfig(cfg) {
			return fmt.Errorf("failed to save config")
		}

		ui.Success("job %q removed", name)
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Skip confirmation prompt")
	rootCmd.AddCommand(removeCmd)
}
