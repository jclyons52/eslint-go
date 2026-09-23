package formatters

import (
	"fmt"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// formatters.go — the formatter registry, mirroring ESLint's --format names for
// the formatters this port implements.

// Supported formatter names (the ones this port implements).
const (
	NameStylish = "stylish"
	NameJSON    = "json"
	NameCompact = "compact"
	NameUnix    = "unix"
)

// FormatResults renders results with the named formatter. colorLevel is chalk's
// color level (0 = no color, 1 = basic), matching ESLint's --color handling.
func FormatResults(name string, results []*eslint.LintResult, colorLevel int) (string, error) {
	switch strings.TrimSpace(name) {
	case "", NameStylish:
		return Stylish(results, colorLevel), nil
	case NameJSON:
		return JSON(results)
	case NameCompact:
		return Compact(results, colorLevel), nil
	case NameUnix:
		return Unix(results, colorLevel), nil
	default:
		return "", fmt.Errorf("there was a problem loading formatter: %s (supported: stylish, json, compact, unix)", name)
	}
}

// Compact is eslint's compact formatter.
func Compact(results []*eslint.LintResult, colorLevel int) string {
	chalk := NewChalk(colorLevel)
	var b strings.Builder
	errorCount, warningCount := 0, 0
	total := 0

	for _, result := range results {
		messages := result.Messages
		if len(messages) == 0 {
			continue
		}
		errorCount += result.ErrorCount
		warningCount += result.WarningCount

		// Last unignored message for the file drives the summary line, exactly
		// as the original does.
		last := messages[len(messages)-1]
		for _, m := range messages {
			var msgType string
			if m.Fatal || m.Severity == 2 {
				msgType = chalk.Red("Error")
			} else {
				msgType = chalk.Yellow("Warning")
			}
			line := fmt.Sprintf("%s: line %s, col %s, %s - %s",
				result.FilePath, itoa(m.Line), itoa(m.Column), msgType, m.Message)
			if m.RuleID != "" {
				line += " (" + m.RuleID + ")"
			}
			b.WriteString(line + "\n")
		}
		if last.Fatal || last.Severity == 2 {
			b.WriteString("\n" + chalk.RedBold(itoa(result.ErrorCount)) +
				chalk.Red(" problem"+pluralize("", result.ErrorCount)) + "\n")
		} else {
			b.WriteString("\n" + chalk.YellowBold(itoa(result.WarningCount)) +
				chalk.Yellow(" problem"+pluralize("", result.WarningCount)) + "\n")
		}
		total++
	}

	if total > 0 {
		b.WriteString("\n" + chalk.RedBold(itoa(errorCount+warningCount)) + " problems\n")
		return chalk.Reset("\n" + b.String())
	}
	return ""
}

// Unix is eslint's unix formatter (one line per message, no summary).
func Unix(results []*eslint.LintResult, colorLevel int) string {
	chalk := NewChalk(colorLevel)
	var b strings.Builder
	errorCount, warningCount := 0, 0
	total := 0

	for _, result := range results {
		messages := result.Messages
		if len(messages) == 0 {
			continue
		}
		errorCount += result.ErrorCount
		warningCount += result.WarningCount
		for _, m := range messages {
			msgType := chalk.Yellow("warning")
			if m.Fatal || m.Severity == 2 {
				msgType = chalk.Red("error")
			}
			line := fmt.Sprintf("%s:%s:%s: %s - %s",
				result.FilePath, itoa(m.Line), itoa(m.Column), m.Message, msgType)
			if m.RuleID != "" {
				line += " (" + m.RuleID + ")"
			}
			b.WriteString(line + "\n")
		}
		total++
	}

	if total > 0 {
		b.WriteString("\n" + itoa(errorCount+warningCount) + " problems\n")
		return chalk.Reset(b.String())
	}
	return ""
}
