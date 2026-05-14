package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Xwudao/go-timer/internal/ui"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		ui.Header("timerd")
		fmt.Printf("  version : %s\n", appVersion)
		fmt.Printf("  commit  : %s\n", appCommit)
		fmt.Printf("  built   : %s\n", appDate)
		fmt.Printf("  mode    : %s\n", modeLabel())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
