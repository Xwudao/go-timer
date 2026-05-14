package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/config"
	"github.com/Xwudao/go-timer/internal/ui"
	"github.com/Xwudao/go-timer/internal/watcher"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Watch config directory and auto-reload changed jobs",
	Long: `Starts a background watcher on the timerd config directory.

When config.yml or template files change, timerd automatically:
  1. Re-generates affected unit files (only writes when content changed)
  2. Runs systemctl daemon-reload
  3. Restarts any currently active timers for changed jobs

Press Ctrl+C to stop the daemon.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgDir := config.DefaultConfigDir(isUserMode())

		if err := os.MkdirAll(cfgDir, 0o755); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}

		gen := newGenerator()
		mgr := newManager()
		ud := unitDir()

		handle := func(changedPath string) {
			ui.Info("change detected → %s", changedPath)

			cfg, err := config.Load(configPath())
			if err != nil {
				ui.Error("reloading config: %v", err)
				return
			}

			changed := 0
			for name, job := range cfg.Jobs {
				installed, err := gen.InstallIfChanged(name, job, ud, isUserMode())
				if err != nil {
					ui.Error("job %q: %v", name, err)
					continue
				}
				if installed {
					changed++
					ui.Success("unit files updated: %s", name)
				}
			}

			if changed == 0 {
				ui.Dim("  all units unchanged — skipping daemon-reload")
				return
			}

			if err := mgr.DaemonReload(); err != nil {
				ui.Warn("daemon-reload failed: %v", err)
			} else {
				ui.Dim("  daemon-reload OK")
			}

			for name := range cfg.Jobs {
				if mgr.IsActive(name) {
					if err := mgr.Restart(name); err != nil {
						ui.Warn("restarting %q: %v", name, err)
					} else {
						ui.Success("restarted %q", name)
					}
				}
			}
		}

		w, err := watcher.New(cfgDir, handle)
		if err != nil {
			return fmt.Errorf("starting watcher: %w", err)
		}
		defer w.Stop()

		ui.Info("daemon watching %s", cfgDir)
		ui.Dim("  mode  : %s", modeLabel())
		ui.Dim("  press Ctrl+C to stop")
		fmt.Println()

		go w.Run()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println()
		ui.Info("daemon stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
