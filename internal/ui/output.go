// Package ui provides colored, formatted terminal output for timerd.
package ui

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
)

var (
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	warnColor    = color.New(color.FgYellow, color.Bold)
	infoColor    = color.New(color.FgCyan)
	headerColor  = color.New(color.FgWhite, color.Bold)
	dimColor     = color.New(color.FgHiBlack)
	accentColor  = color.New(color.FgMagenta, color.Bold)
)

// Success prints a green success message.
func Success(format string, args ...any) {
	fmt.Fprint(os.Stdout, successColor.Sprint("✔ ")+fmt.Sprintf(format, args...)+"\n")
}

// Error prints a red error message to stderr.
func Error(format string, args ...any) {
	fmt.Fprint(os.Stderr, errorColor.Sprint("✘ ")+fmt.Sprintf(format, args...)+"\n")
}

// Warn prints a yellow warning message.
func Warn(format string, args ...any) {
	fmt.Fprint(os.Stdout, warnColor.Sprint("⚠ ")+fmt.Sprintf(format, args...)+"\n")
}

// Info prints a cyan informational message.
func Info(format string, args ...any) {
	fmt.Fprint(os.Stdout, infoColor.Sprint("ℹ ")+fmt.Sprintf(format, args...)+"\n")
}

// Header prints a bold white header.
func Header(text string) {
	fmt.Fprintln(os.Stdout, headerColor.Sprint(text))
}

// Dim prints a dimmed/muted message.
func Dim(format string, args ...any) {
	fmt.Fprint(os.Stdout, dimColor.Sprintf(format, args...)+"\n")
}

// Accent prints an accented (magenta) message.
func Accent(format string, args ...any) {
	fmt.Fprint(os.Stdout, accentColor.Sprintf(format, args...)+"\n")
}

// Print is a plain print.
func Print(format string, args ...any) {
	fmt.Fprint(os.Stdout, fmt.Sprintf(format, args...)+"\n")
}

// StatusBadge returns a colored status badge string.
func StatusBadge(active bool) string {
	if active {
		return successColor.Sprint("● active")
	}
	return dimColor.Sprint("○ inactive")
}

// ActiveBadge returns active/inactive badge.
func ActiveBadge(status string) string {
	switch strings.ToLower(status) {
	case "active", "running":
		return successColor.Sprint("● " + status)
	case "failed":
		return errorColor.Sprint("✘ " + status)
	case "activating":
		return warnColor.Sprint("◌ " + status)
	default:
		return dimColor.Sprint("○ " + status)
	}
}

// Table prints a formatted table with headers and rows.
func Table(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header row
	headerStrs := make([]string, len(headers))
	for i, h := range headers {
		headerStrs[i] = headerColor.Sprint(strings.ToUpper(h))
	}
	fmt.Fprintln(w, strings.Join(headerStrs, "\t"))

	// Separator
	seps := make([]string, len(headers))
	for i, h := range headers {
		seps[i] = strings.Repeat("─", len(h))
	}
	fmt.Fprintln(w, dimColor.Sprint(strings.Join(seps, "\t")))

	// Data rows
	for _, row := range rows {
		cells := make([]string, len(row))
		copy(cells, row)
		fmt.Fprintln(w, strings.Join(cells, "\t"))
	}

	w.Flush()
}

// Prompt displays an interactive prompt and returns the user's input.
func Prompt(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Fprint(os.Stdout, accentColor.Sprint("? ")+label+dimColor.Sprintf(" [%s]", defaultVal)+": ")
	} else {
		fmt.Fprint(os.Stdout, accentColor.Sprint("? ")+label+": ")
	}
	var input string
	_, _ = fmt.Scanln(&input)
	if input == "" {
		return defaultVal
	}
	return input
}

// PromptRequired displays a prompt that cannot be empty.
func PromptRequired(label string) (string, error) {
	for {
		fmt.Fprint(os.Stdout, accentColor.Sprint("? ")+label+warnColor.Sprint(" (required)")+": ")
		var input string
		_, _ = fmt.Scanln(&input)
		input = strings.TrimSpace(input)
		if input != "" {
			return input, nil
		}
		Warn("this field is required")
	}
}

// Confirm displays a yes/no prompt and returns true if the user confirms.
func Confirm(label string, defaultYes bool) bool {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}
	fmt.Fprint(os.Stdout, accentColor.Sprint("? ")+label+" "+dimColor.Sprint(hint)+": ")
	var input string
	_, _ = fmt.Scanln(&input)
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

// DryRunNotice prints a notice that the command is in dry-run mode.
func DryRunNotice() {
	warnColor.Println("⚡ Dry-run mode — no changes will be made")
}

// Separator prints a horizontal separator line.
func Separator() {
	dimColor.Println(strings.Repeat("─", 60))
}
