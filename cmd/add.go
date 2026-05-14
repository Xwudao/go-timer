package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/config"
	"github.com/Xwudao/go-timer/internal/ui"
)

var (
	addCommand     string
	addSchedule    string
	addWorkDir     string
	addDescription string
	addStart       bool
	addEnable      bool
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new job to the config (interactive wizard)",
	Long: `Creates a new scheduled job. If flags are not provided, an interactive
wizard will prompt for the required values.

Schedule examples:
  hourly          daily           weekly
  "*/5 * * * *"   "0 9 * * 1-5"   "0 2 * * *"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := sanitiseName(args[0])
		if name == "" {
			return fmt.Errorf("job name must not be empty")
		}

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("run 'timerd init' first")
		}

		if _, err := cfg.GetJob(name); err == nil {
			return fmt.Errorf("job %q already exists; use 'timerd edit %s' to modify it", name, name)
		}

		job, err := collectJobConfig(cmd, name)
		if err != nil {
			return err
		}

		if err := cfg.AddJob(name, job); err != nil {
			return err
		}

		if !saveConfig(cfg) {
			return fmt.Errorf("failed to save config")
		}

		ui.Success("job %q added to config", name)

		if addEnable || addStart {
			gen := newGenerator()
			mgr := newManager()

			ui.Info("installing systemd units …")
			if _, err := gen.Install(name, job, unitDir(), isUserMode()); err != nil {
				ui.Error("installing units: %v", err)
			} else {
				_ = mgr.DaemonReload()
				if addEnable {
					if err := mgr.Enable(name); err != nil {
						ui.Warn("enabling timer: %v", err)
					}
				}
				if addStart {
					if err := mgr.Start(name); err != nil {
						ui.Warn("starting timer: %v", err)
					} else {
						ui.Success("timer %q started", name)
					}
				}
			}
		} else {
			ui.Dim("  run 'timerd start %s' to activate", name)
		}

		return nil
	},
}

func collectJobConfig(cmd *cobra.Command, name string) (*config.JobConfig, error) {
	job := &config.JobConfig{}

	// Use flags if provided, otherwise interactive prompt.
	cmdVal := addCommand
	if cmdVal == "" && !cmd.Flags().Changed("command") {
		var err error
		cmdVal, err = ui.PromptRequired(fmt.Sprintf("Command for %q", name))
		if err != nil {
			return nil, err
		}
	}
	job.Command = cmdVal

	schedVal := addSchedule
	if schedVal == "" && !cmd.Flags().Changed("schedule") {
		schedVal = promptWithExamples()
	}
	job.Schedule = schedVal

	if addWorkDir != "" {
		job.WorkDir = addWorkDir
	} else if !cmd.Flags().Changed("workdir") {
		job.WorkDir = ui.Prompt("Working directory", "")
	}

	if addDescription != "" {
		job.Description = addDescription
	} else if !cmd.Flags().Changed("description") {
		job.Description = ui.Prompt("Description", "")
	}

	return job, nil
}

func promptWithExamples() string {
	fmt.Println()
	ui.Dim("  Schedule examples:")
	ui.Dim("    hourly | daily | weekly | monthly")
	ui.Dim("    */5 * * * *   (every 5 minutes)")
	ui.Dim("    0 9 * * 1-5   (weekdays at 09:00)")
	fmt.Println()

	for {
		val := strings.TrimSpace(ui.Prompt("Schedule", "daily"))
		if val != "" {
			return val
		}
		ui.Warn("schedule is required")
	}
}

// sanitiseName strips dangerous characters from a job name.
func sanitiseName(s string) string {
	var out strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' ||
			r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' ||
			r == '-' || r == '_' {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func init() {
	addCmd.Flags().StringVarP(&addCommand, "command", "c", "", "Command to run")
	addCmd.Flags().StringVarP(&addSchedule, "schedule", "s", "", "Schedule (cron or systemd keyword)")
	addCmd.Flags().StringVarP(&addWorkDir, "workdir", "w", "", "Working directory")
	addCmd.Flags().StringVarP(&addDescription, "description", "d", "", "Job description")
	addCmd.Flags().BoolVar(&addStart, "start", false, "Start the timer immediately after adding")
	addCmd.Flags().BoolVar(&addEnable, "enable", false, "Enable the timer on boot after adding")
	rootCmd.AddCommand(addCmd)
}
