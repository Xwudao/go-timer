package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsLines  int
)

var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show journal logs for a job",
	Long:  "Equivalent to: journalctl -u timerd-<name>.service [--follow] [-n N]",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveJobName(args[0])

		cfg, ok := loadConfig()
		if !ok {
			return fmt.Errorf("could not load config")
		}
		mustJob(cfg, name) // validate name

		unit := "timerd-" + name + ".service"

		jArgs := []string{}
		if isUserMode() {
			jArgs = append(jArgs, "--user")
		}
		jArgs = append(jArgs, "-u", unit)
		jArgs = append(jArgs, "-n", strconv.Itoa(logsLines))
		if logsFollow {
			jArgs = append(jArgs, "--follow")
		}
		jArgs = append(jArgs, "--no-pager")

		c := exec.Command("journalctl", jArgs...) //nolint:gosec
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin

		if err := c.Run(); err != nil {
			// journalctl exits non-zero on empty log or missing unit; don't treat as fatal.
			return nil
		}
		return nil
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "Number of recent lines to show")
	rootCmd.AddCommand(logsCmd)
}
