package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var exportDir string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export all generated unit files to a directory",
	Long:  "Generates .service and .timer files for all jobs and writes them to the target directory.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}

		if len(cfg.Jobs) == 0 {
			ui.Info("no jobs to export")
			return nil
		}

		dest := exportDir
		if dest == "" {
			dest = "./timerd-units"
		}

		if err := os.MkdirAll(dest, 0o755); err != nil {
			return fmt.Errorf("creating export dir: %w", err)
		}

		gen := newGenerator()
		errs := 0

		for name, job := range cfg.Jobs {
			pair, err := gen.Generate(name, job, isUserMode())
			if err != nil {
				ui.Error("job %q: %v", name, err)
				errs++
				continue
			}

			svcPath := filepath.Join(dest, pair.ServiceName)
			tmrPath := filepath.Join(dest, pair.TimerName)

			if err := os.WriteFile(svcPath, []byte(pair.Service), 0o644); err != nil {
				ui.Error("writing %s: %v", pair.ServiceName, err)
				errs++
				continue
			}
			if err := os.WriteFile(tmrPath, []byte(pair.Timer), 0o644); err != nil {
				ui.Error("writing %s: %v", pair.TimerName, err)
				errs++
				continue
			}

			ui.Success("  %s  %s", pair.ServiceName, pair.TimerName)
		}

		if errs > 0 {
			return fmt.Errorf("%d job(s) failed to export", errs)
		}

		ui.Print("")
		ui.Success("exported %d job(s) to %s", len(cfg.Jobs), dest)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportDir, "output", "o", "", "Output directory (default: ./timerd-units)")
	rootCmd.AddCommand(exportCmd)
}
