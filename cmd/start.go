package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/config"
	"github.com/Xwudao/go-timer/internal/ui"
)

var startSchedule string

var startCmd = &cobra.Command{
	Use:   "start <name|script>",
	Short: "Install units and start the timer for a job (or register and start a script)",
	Long: `Installs systemd unit files and starts the timer for a job.

Like pm2, you can pass a script or binary path directly:

  timerd start ./lz-gen-tag.sh          # auto-registers with schedule "daily"
  timerd start ./backup.sh -s "0 3 * * *"

If the job is already registered, the name or filename both work:

  timerd start lz-gen-tag
  timerd start lz-gen-tag.sh`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := args[0]
		name := resolveJobName(arg)

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		// PM2-like: if the argument looks like a file path and no job exists
		// yet, auto-register it so the user never has to run "timerd add".
		if looksLikeFilePath(arg) {
			if _, err := cfg.GetJob(name); err != nil {
				absPath, absErr := filepath.Abs(arg)
				if absErr != nil {
					absPath = arg
				}
				sched := startSchedule
				if sched == "" {
					sched = "daily"
				}
				newJob := &config.JobConfig{
					Command:  absPath,
					Schedule: sched,
				}
				if err := cfg.AddJob(name, newJob); err != nil {
					return fmt.Errorf("auto-registering job %q: %w", name, err)
				}
				if !saveConfig(cfg) {
					return fmt.Errorf("failed to save config")
				}
				ui.Info("registered job %q → %s (schedule: %s)", name, absPath, sched)
			}
		}

		job := mustJob(cfg, name)

		gen := newGenerator()
		mgr := newManager()

		ui.Info("installing unit files …")
		if _, err := gen.Install(name, job, unitDir(), isUserMode()); err != nil {
			return fmt.Errorf("installing units: %w", err)
		}

		if err := mgr.DaemonReload(); err != nil {
			ui.Warn("daemon-reload: %v", err)
		}

		if err := mgr.Start(name); err != nil {
			return fmt.Errorf("starting timer: %w", err)
		}

		// Mark as enabled in config.
		job.Enabled = true
		_ = cfg.UpdateJob(name, job)
		_ = saveConfig(cfg)

		ui.Success("timer %q started", name)
		return nil
	},
}

func init() {
	startCmd.Flags().StringVarP(&startSchedule, "schedule", "s", "", "Cron schedule when auto-registering from a file path (default: daily)")
	rootCmd.AddCommand(startCmd)
}
