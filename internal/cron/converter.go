// Package cron converts cron expressions to systemd OnCalendar format.
package cron

import (
	"fmt"
	"strconv"
	"strings"
)

// systemd shortcut keywords that can be used directly as OnCalendar values.
var systemdKeywords = map[string]bool{
	"hourly": true, "daily": true, "weekly": true, "monthly": true,
	"yearly": true, "annually": true, "minutely": true,
	"quarterly": true, "semiannually": true,
}

// @-style cron shortcuts mapped to systemd equivalents.
var atShortcuts = map[string]string{
	"@hourly":   "hourly",
	"@daily":    "daily",
	"@midnight": "daily",
	"@weekly":   "weekly",
	"@monthly":  "monthly",
	"@yearly":   "yearly",
	"@annually": "yearly",
}

// dowNames maps cron DOW numbers (0=Sun) to systemd day abbreviations.
var dowNames = [8]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// IsSystemdSchedule reports whether s is already a valid systemd schedule
// (keyword or OnCalendar expression that needs no conversion).
func IsSystemdSchedule(s string) bool {
	return systemdKeywords[strings.ToLower(s)]
}

// IsCronExpression reports whether s looks like a 5-field cron expression.
func IsCronExpression(s string) bool {
	return len(strings.Fields(s)) == 5
}

// ToOnCalendar converts a cron expression or schedule shortcut to a
// systemd OnCalendar value. Systemd keywords are returned unchanged.
func ToOnCalendar(schedule string) (string, error) {
	lower := strings.ToLower(strings.TrimSpace(schedule))

	// Already a systemd keyword.
	if systemdKeywords[lower] {
		return lower, nil
	}

	// @-style cron shortcut.
	if v, ok := atShortcuts[lower]; ok {
		return v, nil
	}

	// @reboot cannot be expressed as a timer.
	if lower == "@reboot" {
		return "", fmt.Errorf("@reboot is not supported; use oneshot=true with no schedule instead")
	}

	// 5-field cron expression: MIN HOUR DOM MONTH DOW
	parts := strings.Fields(schedule)
	if len(parts) != 5 {
		return "", fmt.Errorf("expected 5-field cron expression or systemd keyword, got %q", schedule)
	}

	return convertFiveField(parts[0], parts[1], parts[2], parts[3], parts[4])
}

func convertFiveField(minute, hour, dom, month, dow string) (string, error) {
	var sb strings.Builder

	// Day-of-week prefix.
	if dow != "*" {
		dowStr, err := convertDOW(dow)
		if err != nil {
			return "", fmt.Errorf("day-of-week: %w", err)
		}
		sb.WriteString(dowStr)
		sb.WriteString(" ")
	}

	// Date component: *-MONTH-DOM
	monthStr, err := convertDateField(month, 1, 12)
	if err != nil {
		return "", fmt.Errorf("month: %w", err)
	}
	domStr, err := convertDateField(dom, 1, 31)
	if err != nil {
		return "", fmt.Errorf("day-of-month: %w", err)
	}
	sb.WriteString(fmt.Sprintf("*-%s-%s ", monthStr, domStr))

	// Time component: HOUR:MIN:00
	hourStr, err := convertTimeField(hour, 0, 23)
	if err != nil {
		return "", fmt.Errorf("hour: %w", err)
	}
	minStr, err := convertTimeField(minute, 0, 59)
	if err != nil {
		return "", fmt.Errorf("minute: %w", err)
	}
	sb.WriteString(fmt.Sprintf("%s:%s:00", hourStr, minStr))

	return sb.String(), nil
}

// convertDateField converts a cron field used in dates (month, dom).
// Wildcard returns "*", plain numbers are zero-padded to 2 digits.
func convertDateField(field string, min, max int) (string, error) {
	if field == "*" {
		return "*", nil
	}
	return convertField(field, min, max, true)
}

// convertTimeField converts a cron field used in times (hour, minute).
// Wildcard returns "*", plain numbers are zero-padded to 2 digits.
func convertTimeField(field string, min, max int) (string, error) {
	if field == "*" {
		return "*", nil
	}
	return convertField(field, min, max, true)
}

// convertField is the generic field converter supporting */n, n-m, n,m,o and plain numbers.
func convertField(field string, min, max int, pad bool) (string, error) {
	// Step: */n  or  start/n
	if before, after, ok := strings.Cut(field, "/"); ok {
		base := before
		stepStr := after
		step, err := strconv.Atoi(stepStr)
		if err != nil || step <= 0 {
			return "", fmt.Errorf("invalid step value %q", stepStr)
		}
		if base == "*" {
			return fmt.Sprintf("0/%d", step), nil
		}
		start, err := strconv.Atoi(base)
		if err != nil {
			return "", fmt.Errorf("invalid step base %q", base)
		}
		return fmt.Sprintf("%d/%d", start, step), nil
	}

	// List: a,b,c  (systemd also uses commas, so pass through)
	if strings.Contains(field, ",") {
		// Validate each element.
		parts := strings.Split(field, ",")
		results := make([]string, 0, len(parts))
		for _, p := range parts {
			r, err := convertField(p, min, max, pad)
			if err != nil {
				return "", err
			}
			results = append(results, r)
		}
		return strings.Join(results, ","), nil
	}

	// Range: n-m  → systemd uses n..m
	if strings.Contains(field, "-") {
		halves := strings.SplitN(field, "-", 2)
		start, err := strconv.Atoi(halves[0])
		if err != nil {
			return "", fmt.Errorf("invalid range start %q", halves[0])
		}
		end, err := strconv.Atoi(halves[1])
		if err != nil {
			return "", fmt.Errorf("invalid range end %q", halves[1])
		}
		return fmt.Sprintf("%d..%d", start, end), nil
	}

	// Plain integer.
	n, err := strconv.Atoi(field)
	if err != nil {
		return "", fmt.Errorf("invalid value %q", field)
	}
	_ = min
	_ = max
	if pad {
		return fmt.Sprintf("%02d", n), nil
	}
	return strconv.Itoa(n), nil
}

// convertDOW converts a cron day-of-week field to systemd abbreviation(s).
func convertDOW(dow string) (string, error) {
	// List
	if strings.Contains(dow, ",") {
		parts := strings.Split(dow, ",")
		names := make([]string, 0, len(parts))
		for _, p := range parts {
			name, err := parseSingleDOW(strings.TrimSpace(p))
			if err != nil {
				return "", err
			}
			names = append(names, name)
		}
		return strings.Join(names, ","), nil
	}

	// Range  n-m  →  Mon..Fri
	if strings.Contains(dow, "-") {
		halves := strings.SplitN(dow, "-", 2)
		start, err := parseSingleDOW(halves[0])
		if err != nil {
			return "", err
		}
		end, err := parseSingleDOW(halves[1])
		if err != nil {
			return "", err
		}
		return start + ".." + end, nil
	}

	return parseSingleDOW(dow)
}

// parseSingleDOW converts a single cron day-of-week token to a systemd abbreviation.
func parseSingleDOW(s string) (string, error) {
	s = strings.TrimSpace(s)

	// Numeric (0-7, where 0 and 7 = Sunday)
	if n, err := strconv.Atoi(s); err == nil {
		if n < 0 || n > 7 {
			return "", fmt.Errorf("day-of-week number must be 0-7, got %d", n)
		}
		return dowNames[n], nil
	}

	// Abbreviated name (Mon, Tue, …)
	abbrs := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for _, a := range abbrs {
		if strings.EqualFold(s, a) {
			return a, nil
		}
	}

	// Full name
	full := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	for i, f := range full {
		if strings.EqualFold(s, f) {
			return abbrs[i], nil
		}
	}

	return "", fmt.Errorf("unrecognised day-of-week %q", s)
}
