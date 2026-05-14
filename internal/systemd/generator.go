// Package systemd generates and manages systemd unit files for timerd.
package systemd

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Xwudao/go-timer/internal/config"
	"github.com/Xwudao/go-timer/internal/cron"
)

//go:embed service.tmpl
var defaultServiceTmpl string

//go:embed timer.tmpl
var defaultTimerTmpl string

// ServiceData holds the data passed to the service template.
type ServiceData struct {
	Description string
	WorkDir     string
	ExecStart   string
	Env         map[string]string
	Restart     string
	RestartSec  string
	Timeout     string
	User        string
	OneShot     bool
	After       []string
	Wants       []string
	Requires    []string
	UserMode    bool
}

// TimerData holds the data passed to the timer template.
type TimerData struct {
	Description string
	OnCalendar  string
	Persistent  bool
	ServiceName string
}

// UnitPair holds a generated service and timer unit content.
type UnitPair struct {
	ServiceName string
	TimerName   string
	Service     string
	Timer       string
}

// Generator generates systemd unit files from job configs.
type Generator struct {
	// CustomTemplateDIr overrides the embedded templates when set.
	CustomTemplateDir string
}

// NewGenerator creates a new Generator.
func NewGenerator(customTemplateDir string) *Generator {
	return &Generator{CustomTemplateDir: customTemplateDir}
}

// UnitName returns the systemd unit base name for a job.
func UnitName(jobName string) string {
	return "timerd-" + jobName
}

// ServiceFileName returns the .service filename for a job.
func ServiceFileName(jobName string) string {
	return UnitName(jobName) + ".service"
}

// TimerFileName returns the .timer filename for a job.
func TimerFileName(jobName string) string {
	return UnitName(jobName) + ".timer"
}

// Generate produces the service and timer unit content for a job.
func (g *Generator) Generate(name string, job *config.JobConfig, userMode bool) (*UnitPair, error) {
	onCalendar, err := scheduleToOnCalendar(job.Schedule)
	if err != nil {
		return nil, fmt.Errorf("converting schedule: %w", err)
	}

	execStart := buildExecStart(job)
	description := job.Description
	if description == "" {
		description = fmt.Sprintf("timerd job: %s", name)
	}

	serviceData := &ServiceData{
		Description: description,
		WorkDir:     job.WorkDir,
		ExecStart:   execStart,
		Env:         job.Env,
		Restart:     job.Restart,
		RestartSec:  job.RestartSec,
		Timeout:     job.Timeout,
		User:        job.User,
		OneShot:     job.OneShot,
		After:       job.After,
		Wants:       job.Wants,
		Requires:    job.Requires,
		UserMode:    userMode,
	}

	timerData := &TimerData{
		Description: description,
		OnCalendar:  onCalendar,
		Persistent:  job.Persistent,
		ServiceName: ServiceFileName(name),
	}

	serviceTmpl, err := g.loadTemplate("service.tmpl", defaultServiceTmpl)
	if err != nil {
		return nil, fmt.Errorf("loading service template: %w", err)
	}
	timerTmpl, err := g.loadTemplate("timer.tmpl", defaultTimerTmpl)
	if err != nil {
		return nil, fmt.Errorf("loading timer template: %w", err)
	}

	serviceContent, err := renderTemplate(serviceTmpl, serviceData)
	if err != nil {
		return nil, fmt.Errorf("rendering service template: %w", err)
	}
	timerContent, err := renderTemplate(timerTmpl, timerData)
	if err != nil {
		return nil, fmt.Errorf("rendering timer template: %w", err)
	}

	return &UnitPair{
		ServiceName: ServiceFileName(name),
		TimerName:   TimerFileName(name),
		Service:     serviceContent,
		Timer:       timerContent,
	}, nil
}

// Install writes unit files to the target directory.
func (g *Generator) Install(name string, job *config.JobConfig, unitDir string, userMode bool) (*UnitPair, error) {
	pair, err := g.Generate(name, job, userMode)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating unit directory %s: %w", unitDir, err)
	}

	if err := os.WriteFile(filepath.Join(unitDir, pair.ServiceName), []byte(pair.Service), 0o644); err != nil {
		return nil, fmt.Errorf("writing service unit: %w", err)
	}
	if err := os.WriteFile(filepath.Join(unitDir, pair.TimerName), []byte(pair.Timer), 0o644); err != nil {
		return nil, fmt.Errorf("writing timer unit: %w", err)
	}

	return pair, nil
}

// Remove deletes unit files from the target directory.
func (g *Generator) Remove(name, unitDir string) error {
	servicePath := filepath.Join(unitDir, ServiceFileName(name))
	timerPath := filepath.Join(unitDir, TimerFileName(name))

	var errs []string
	for _, p := range []string{servicePath, timerPath} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("removing unit files: %s", strings.Join(errs, "; "))
	}
	return nil
}

// loadTemplate loads a template from a custom directory or uses the embedded default.
func (g *Generator) loadTemplate(name, embedded string) (string, error) {
	if g.CustomTemplateDir != "" {
		path := filepath.Join(g.CustomTemplateDir, name)
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data), nil
		}
		// Fall through to embedded template if the custom one is missing.
	}
	return embedded, nil
}

// renderTemplate executes a template string with the given data.
func renderTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("unit").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	// Normalise blank lines left by conditional blocks.
	return normaliseBlankLines(buf.String()), nil
}

// normaliseBlankLines collapses multiple consecutive blank lines into one.
func normaliseBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	prev := false
	for _, l := range lines {
		blank := strings.TrimSpace(l) == ""
		if blank && prev {
			continue
		}
		out = append(out, l)
		prev = blank
	}
	return strings.Join(out, "\n")
}

// buildExecStart assembles the ExecStart value from command + args.
func buildExecStart(job *config.JobConfig) string {
	if len(job.Args) == 0 {
		return job.Command
	}
	quoted := make([]string, len(job.Args))
	for i, a := range job.Args {
		if strings.ContainsAny(a, " \t") {
			quoted[i] = fmt.Sprintf("%q", a)
		} else {
			quoted[i] = a
		}
	}
	return job.Command + " " + strings.Join(quoted, " ")
}

// scheduleToOnCalendar converts a schedule string (cron or systemd keyword)
// to a systemd OnCalendar value.
func scheduleToOnCalendar(schedule string) (string, error) {
	return cron.ToOnCalendar(schedule)
}
