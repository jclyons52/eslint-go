package formatters

import (
	"fmt"
	"regexp"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// stylish.go — a port of eslint 8.57's lib/cli-engine/formatters/stylish.js.
//
// It reproduces the exact string, including the leading newline, the
// line:column dimming pass over the rendered table, the trailing-period
// stripping, and the reset wrapper — because `eslint-go --format stylish` must
// diff clean against `eslint --format stylish`.

// pluralize appends an "s" unless the count is 1.
func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}

// trailingPeriod strips a period that follows a non-space character (the JS
// `message.replace(/([^ ])\.$/u, "$1")`).
var trailingPeriod = regexp.MustCompile(`([^ ])\.$`)

// lineColRe matches the "line column" pair the formatter rewrites to
// "line:column" in dim style.
var lineColRe = regexp.MustCompile(`(\d+)\s+(\d+)`)

// Stylish renders results the way ESLint's stylish formatter does.
func Stylish(results []*eslint.LintResult, colorLevel int) string {
	chalk := NewChalk(colorLevel)

	output := "\n"
	errorCount, warningCount := 0, 0
	fixableErrorCount, fixableWarningCount := 0, 0
	summaryColor := "yellow"

	for _, result := range results {
		messages := result.Messages
		if len(messages) == 0 {
			continue
		}

		errorCount += result.ErrorCount
		warningCount += result.WarningCount
		fixableErrorCount += result.FixableErrorCount
		fixableWarningCount += result.FixableWarningCount

		output += chalk.Underline(result.FilePath) + "\n"

		rows := make([][]string, 0, len(messages))
		for _, message := range messages {
			var messageType string
			if message.Fatal || message.Severity == 2 {
				messageType = chalk.Red("error")
				summaryColor = "red"
			} else {
				messageType = chalk.Yellow("warning")
			}
			ruleID := ""
			if message.RuleID != "" {
				ruleID = message.RuleID
			}
			rows = append(rows, []string{
				"",
				itoa(message.Line),
				itoa(message.Column),
				messageType,
				trailingPeriod.ReplaceAllString(message.Message, "$1"),
				chalk.Dim(ruleID),
			})
		}

		table := textTable(rows, tableOptions{
			align:        []string{"", "r", "l"},
			stringLength: func(s string) int { return visibleLen(s) },
		})

		dimmed := lineColRe.ReplaceAllStringFunc(table, func(m string) string {
			sub := lineColRe.FindStringSubmatch(m)
			return chalk.Dim(sub[1] + ":" + sub[2])
		})

		output += dimmed + "\n\n"
	}

	total := errorCount + warningCount
	if total > 0 {
		if summaryColor == "red" {
			output += chalk.RedBold(strings.Join([]string{
				"\u2716 ", itoa(total), pluralize(" problem", total),
				" (", itoa(errorCount), pluralize(" error", errorCount), ", ",
				itoa(warningCount), pluralize(" warning", warningCount), ")\n",
			}, ""))
		} else {
			output += chalk.YellowBold(strings.Join([]string{
				"\u2716 ", itoa(total), pluralize(" problem", total),
				" (", itoa(errorCount), pluralize(" error", errorCount), ", ",
				itoa(warningCount), pluralize(" warning", warningCount), ")\n",
			}, ""))
		}

		if fixableErrorCount > 0 || fixableWarningCount > 0 {
			line := strings.Join([]string{
				"  ", itoa(fixableErrorCount), pluralize(" error", fixableErrorCount), " and ",
				itoa(fixableWarningCount), pluralize(" warning", fixableWarningCount),
				" potentially fixable with the `--fix` option.\n",
			}, "")
			if summaryColor == "red" {
				output += chalk.RedBold(line)
			} else {
				output += chalk.YellowBold(line)
			}
		}
	}

	if total > 0 {
		return chalk.Reset(output)
	}
	return ""
}

// JSON renders results the way ESLint's json formatter does: the results array
// serialized with the same keys and key order the CLI produces.
func JSON(results []*eslint.LintResult) (string, error) {
	var b strings.Builder
	b.WriteString("[")
	for i, r := range results {
		if i > 0 {
			b.WriteString(",")
		}
		s, err := jsonResult(r)
		if err != nil {
			return "", err
		}
		b.WriteString(s)
	}
	b.WriteString("]")
	return b.String(), nil
}

func jsonResult(r *eslint.LintResult) (string, error) {
	var b strings.Builder
	b.WriteString("{")
	b.WriteString(`"filePath":` + jsonString(r.FilePath))
	b.WriteString(`,"messages":[`)
	for i, m := range r.Messages {
		if i > 0 {
			b.WriteString(",")
		}
		s, err := jsonMessage(m)
		if err != nil {
			return "", err
		}
		b.WriteString(s)
	}
	b.WriteString("]")
	b.WriteString(`,"suppressedMessages":[]`)
	b.WriteString(`,"errorCount":` + itoa(r.ErrorCount))
	b.WriteString(`,"fatalErrorCount":` + itoa(r.FatalErrorCount()))
	b.WriteString(`,"warningCount":` + itoa(r.WarningCount))
	b.WriteString(`,"fixableErrorCount":` + itoa(r.FixableErrorCount))
	b.WriteString(`,"fixableWarningCount":` + itoa(r.FixableWarningCount))
	if r.Fixed && r.Output != "" {
		b.WriteString(`,"output":` + jsonString(r.Output))
	} else if r.Source != "" && r.ErrorCount+r.WarningCount > 0 {
		// eslint includes the source text when a file has problems and was not
		// rewritten by --fix (cli-engine creates the `source` key then).
		b.WriteString(`,"source":` + jsonString(r.Source))
	}
	b.WriteString(`,"usedDeprecatedRules":[`)
	for i, d := range r.UsedDeprecatedRules {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"ruleId":` + jsonString(d.RuleID) + `,"replacedBy":[`)
		for j, rb := range d.ReplacedBy {
			if j > 0 {
				b.WriteString(",")
			}
			b.WriteString(jsonString(rb))
		}
		b.WriteString("]}")
	}
	b.WriteString("]")
	b.WriteString("}")
	return b.String(), nil
}

func jsonMessage(m eslint.Message) (string, error) {
	b, err := m.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("marshal message: %w", err)
	}
	return string(b), nil
}
