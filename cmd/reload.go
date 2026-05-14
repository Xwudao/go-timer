package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Regenerate changed unit files from config and reload systemd",
	Long: `Reads config.yml and regenerates .service / .timer unit files.
Only files whose content actually changed are rewritten. systemctl
daemon-reload is called only when at least one file changed, and only
active timers for modified jobs are restarted.`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		if len(cfg.Jobs) == 0 {
			ui.Info("no jobs defined in config — nothing to reload")
			return nil
		}

		gen := newGenerator()
		mgr := newManager()
		ud := unitDir()
		errs := 0
		changed := 0
		var changedNames []string

		for name, job := range cfg.Jobs {
			if flagDryRun {
				ui.DryRunNotice()
				ui.Info("would regenerate units for %q", name)
				continue
			}
			installed, err := gen.InstallIfChanged(name, job, ud, isUserMode())
			if err != nil {
				ui.Error("job %q: %v", name, err)
				errs++
				continue
			}
			if installed {
				changed++
				changedNames = append(changedNames, name)
				ui.Success("updated units: %s", name)
			} else {
				ui.Dim("  unchanged: %s", name)
			}
		}

		if flagDryRun {
			return nil
		}

		if changed == 0 && errs == 0 {
			ui.Info("all units up-to-date — nothing to reload")
			return nil
		}

		if changed > 0 {
			if err := mgr.DaemonReload(); err != nil {
				ui.Warn("daemon-reload: %v", err)
			}

			for _, name := range changedNames {
				if mgr.IsActive(name) {
					if err := mgr.Restart(name); err != nil {
						ui.Warn("restarting %q: %v", name, err)
					} else {
						ui.Success("restarted %q", name)
					}
				}
			}
		}

		if errs > 0 {
			return fmt.Errorf("%d job(s) failed to reload", errs)
		}

		ui.Success("reload complete (%d changed, %d total)", changed, len(cfg.Jobs))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reloadCmd)
}
