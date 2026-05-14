package cron_test

import (
	"testing"

	"github.com/Xwudao/go-timer/internal/cron"
)

func TestToOnCalendar_Keywords(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hourly", "hourly"},
		{"daily", "daily"},
		{"weekly", "weekly"},
		{"monthly", "monthly"},
		{"yearly", "yearly"},
		{"annually", "annually"},
		{"@hourly", "hourly"},
		{"@daily", "daily"},
		{"@weekly", "weekly"},
		{"@monthly", "monthly"},
		{"@yearly", "yearly"},
	}

	for _, tc := range tests {
		got, err := cron.ToOnCalendar(tc.input)
		if err != nil {
			t.Errorf("ToOnCalendar(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ToOnCalendar(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestToOnCalendar_CronExpressions(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"* * * * *", "*-*-* *:*:00"},
		{"*/5 * * * *", "*-*-* *:0/5:00"},
		{"0 * * * *", "*-*-* *:00:00"},
		{"0 0 * * *", "*-*-* 00:00:00"},
		{"30 6 * * *", "*-*-* 06:30:00"},
		{"0 9 1 * *", "*-*-01 09:00:00"},
		{"0 9 * 1 *", "*-01-* 09:00:00"},
		{"0 */2 * * *", "*-*-* 0/2:00:00"},
		{"0 0 * * 0", "Sun *-*-* 00:00:00"},
		{"0 0 * * 7", "Sun *-*-* 00:00:00"},
		{"0 9 * * 1", "Mon *-*-* 09:00:00"},
		{"0 9 * * 1-5", "Mon..Fri *-*-* 09:00:00"},
		{"0 9 * * 1,3,5", "Mon,Wed,Fri *-*-* 09:00:00"},
		{"*/15 * * * *", "*-*-* *:0/15:00"},
	}

	for _, tc := range tests {
		got, err := cron.ToOnCalendar(tc.input)
		if err != nil {
			t.Errorf("ToOnCalendar(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ToOnCalendar(%q)\n  got  %q\n  want %q", tc.input, got, tc.want)
		}
	}
}

func TestToOnCalendar_Errors(t *testing.T) {
	tests := []string{
		"@reboot",
		"not-valid",
		"1 2 3",         // only 3 fields
		"1 2 3 4 5 6 7", // 7 fields
	}

	for _, tc := range tests {
		_, err := cron.ToOnCalendar(tc)
		if err == nil {
			t.Errorf("ToOnCalendar(%q) expected error, got nil", tc)
		}
	}
}

func TestIsSystemdSchedule(t *testing.T) {
	trueCases := []string{"hourly", "daily", "weekly", "monthly", "yearly"}
	for _, tc := range trueCases {
		if !cron.IsSystemdSchedule(tc) {
			t.Errorf("IsSystemdSchedule(%q) = false, want true", tc)
		}
	}

	falseCases := []string{"*/5 * * * *", "cron", "@daily"}
	for _, tc := range falseCases {
		if cron.IsSystemdSchedule(tc) {
			t.Errorf("IsSystemdSchedule(%q) = true, want false", tc)
		}
	}
}
