package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var editCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Edit the config file in $EDITOR",
	Long: `Opens the timerd config file in your default editor ($EDITOR or vi).
If a job name is provided it is shown as a hint — the whole config is opened.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		cfgPath := configPath()

		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			return fmt.Errorf("config not found at %s; run 'timerd init' first", cfgPath)
		}

		if len(args) == 1 {
			ui.Info("opening config for job %q …", args[0])
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			editor = "vi"
		}

		execCmd := exec.Command(editor, cfgPath) //nolint:gosec
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		if err := execCmd.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		ui.Info("config saved; run 'timerd reload' to apply changes")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
