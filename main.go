// Command timerd manages systemd timers via a YAML config.
package main

import "github.com/Xwudao/go-timer/cmd"

// Version metadata — injected at build time via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersion(version, commit, date)
	cmd.Execute()
}
