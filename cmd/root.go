// Package cmd implements the timerd CLI.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/config"
	"github.com/Xwudao/go-timer/internal/systemd"
	"github.com/Xwudao/go-timer/internal/ui"
)

var (
	flagSystem  bool
	flagUser    bool
	flagDryRun  bool
	flagVerbose bool

	appVersion = "dev"
	appCommit  = "none"
	appDate    = "unknown"
)

// SetVersion sets version metadata injected at build time.
func SetVersion(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
}

var rootCmd = &cobra.Command{
	Use:   "timerd",
	Short: "A systemd timer/service manager — like PM2, powered by systemd",
	Long: `timerd lets you manage scheduled tasks as systemd timers without
writing unit files by hand. Define jobs in config.yml and timerd
handles the rest.

Documentation: https://github.com/Xwudao/go-timer`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagSystem, "system", false, "Use systemd system mode (requires root)")
	rootCmd.PersistentFlags().BoolVar(&flagUser, "user", false, "Use systemd user mode (default)")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Print actions without executing them")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose output")
}

// isUserMode returns true when running in user mode (default).
func isUserMode() bool {
	return !flagSystem
}

// configPath returns the active config file path.
func configPath() string {
	return config.DefaultConfigPath(isUserMode())
}

// unitDir returns the active systemd unit directory.
func unitDir() string {
	return config.DefaultUnitDir(isUserMode())
}

// newManager creates a Manager from the current flag state.
func newManager() *systemd.Manager {
	return systemd.NewManager(isUserMode(), flagDryRun, flagVerbose)
}

// newGenerator creates a Generator. customDir may be empty.
func newGenerator() *systemd.Generator {
	return systemd.NewGenerator("")
}

// loadConfig loads the config or prints an error and returns nil.
func loadConfig() (*config.Config, bool) {
	cfg, err := config.Load(configPath())
	if err != nil {
		ui.Error("%v", err)
		return nil, false
	}
	return cfg, true
}

// loadConfigErr loads the config and returns the error directly.
func loadConfigErr() (*config.Config, error) {
	return config.Load(configPath())
}

// saveConfig saves the config or prints an error.
func saveConfig(cfg *config.Config) bool {
	if err := config.Save(configPath(), cfg); err != nil {
		ui.Error("saving config: %v", err)
		return false
	}
	return true
}

// mustJob resolves a job by name or exits.
func mustJob(cfg *config.Config, name string) *config.JobConfig {
	job, err := cfg.GetJob(name)
	if err != nil {
		ui.Error("%v", err)
		fmt.Fprintf(os.Stderr, "Available jobs: %s\n", jobNames(cfg))
		os.Exit(1)
	}
	return job
}

func jobNames(cfg *config.Config) string {
	names := make([]string, 0, len(cfg.Jobs))
	for n := range cfg.Jobs {
		names = append(names, n)
	}
	if len(names) == 0 {
		return "(none)"
	}
	return joinStrings(names, ", ")
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

// scriptExts is the list of extensions stripped when resolving a job name from a file path.
var scriptExts = []string{".sh", ".bash", ".zsh", ".py", ".rb", ".js", ".ts", ".pl", ".php"}

// looksLikeFilePath reports whether arg appears to be a file path rather than a plain job name.
// A bare name like "backup" is a job name; "./backup.sh" or "scripts/backup.sh" is a file path.
func looksLikeFilePath(arg string) bool {
	if strings.ContainsRune(arg, '/') {
		return true
	}
	for _, ext := range scriptExts {
		if strings.HasSuffix(arg, ext) {
			return true
		}
	}
	return false
}

// resolveJobName derives a canonical job name from a user-supplied argument.
// If arg looks like a file path, the base filename is used with the extension
// stripped and then sanitised — mirroring pm2's behaviour:
//
//	"timerd start ./lz-gen-tag.sh"  →  job name "lz-gen-tag"
//	"timerd stop  lz-gen-tag.sh"    →  job name "lz-gen-tag"
func resolveJobName(arg string) string {
	base := filepath.Base(arg)
	for _, ext := range scriptExts {
		if strings.HasSuffix(base, ext) {
			base = strings.TrimSuffix(base, ext)
			break
		}
	}
	return sanitiseName(base)
}
