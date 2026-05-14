package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/config"
	"github.com/Xwudao/go-timer/internal/ui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise the timerd config directory",
	Long:  "Creates the config directory and a starter config.yml if they don't exist.",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		cfgPath := configPath()
		cfgDir := filepath.Dir(cfgPath)

		if _, err := os.Stat(cfgPath); err == nil {
			ui.Info("already initialised at %s", cfgPath)
			return nil
		}

		if err := os.MkdirAll(cfgDir, 0o750); err != nil {
			return fmt.Errorf("creating config dir %s: %w", cfgDir, err)
		}

		// Also create the systemd user unit directory.
		ud := unitDir()
		if err := os.MkdirAll(ud, 0o750); err != nil {
			return fmt.Errorf("creating unit dir %s: %w", ud, err)
		}

		starter := config.NewConfig()
		if err := config.Save(cfgPath, starter); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}

		ui.Success("timerd initialised")
		ui.Dim("  config : %s", cfgPath)
		ui.Dim("  units  : %s", ud)
		ui.Print("")
		ui.Info("Add your first job with: timerd add <name>")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
