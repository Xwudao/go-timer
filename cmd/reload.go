package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Regenerate all unit files from config and reload systemd",
	Long: `Reads config.yml, regenerates every .service and .timer unit file,
then runs systemctl daemon-reload. Active timers are restarted so
the new config takes effect immediately.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		for name, job := range cfg.Jobs {
			ui.Info("generating units for %q …", name)
			if flagDryRun {
				ui.DryRunNotice()
				continue
			}
			if _, err := gen.Install(name, job, ud, isUserMode()); err != nil {
				ui.Error("job %q: %v", name, err)
				errs++
			}
		}

		if !flagDryRun {
			if err := mgr.DaemonReload(); err != nil {
				ui.Warn("daemon-reload: %v", err)
			}

			// Restart any currently active timers to pick up changes.
			for name := range cfg.Jobs {
				if mgr.IsActive(name) {
					if err := mgr.Restart(name); err != nil {
						ui.Warn("restarting %q: %v", name, err)
					}
				}
			}
		}

		if errs > 0 {
			return fmt.Errorf("%d job(s) failed to reload", errs)
		}

		ui.Success("reload complete (%d job(s))", len(cfg.Jobs))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reloadCmd)
}
