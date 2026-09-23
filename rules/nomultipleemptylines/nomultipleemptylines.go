package nomultipleemptylines

import (
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-multiple-empty-lines"

// Rule is a port of eslint/lib/rules/no-multiple-empty-lines.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "layout",
		Docs: eslint.RuleDocs{
			Description: "Disallow multiple empty lines",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-multiple-empty-lines",
		},
		Fixable: "whitespace",
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"max":    map[string]any{"type": "integer", "minimum": 0},
					"maxEOF": map[string]any{"type": "integer", "minimum": 0},
					"maxBOF": map[string]any{"type": "integer", "minimum": 0},
				},
				"required":             []any{"max"},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			blankBeginningOfFile: "Too many blank lines at the beginning of file. Max of {{max}} allowed.",
			blankEndOfFile:       "Too many blank lines at the end of file. Max of {{max}} allowed.",
			consecutiveBlank:     "More than {{max}} blank {{pluralizedLines}} not allowed.",
		},
	},
	Create: create,
}

const (
	blankBeginningOfFile = "blankBeginningOfFile"
	blankEndOfFile       = "blankEndOfFile"
	consecutiveBlank     = "consecutiveBlank"
)

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	// Use options.max or 2 as default.
	max := 2
	maxEOF := max
	maxBOF := max

	if opts := ctx.OptionMap(0); opts != nil {
		max = intOption(opts["max"], max)
		maxEOF = max
		if v, ok := opts["maxEOF"]; ok {
			maxEOF = intOption(v, max)
		}
		maxBOF = max
		if v, ok := opts["maxBOF"]; ok {
			maxBOF = intOption(v, max)
		}
	}

	sc := ctx.SourceCode
	lines := sc.Lines()

	// Swallow the final newline, as some editors add it automatically and we
	// don't want it to cause an issue.
	allLines := lines
	if lines[len(lines)-1] == "" {
		allLines = lines[:len(lines)-1]
	}
	templateLiteralLines := map[int]bool{}

	return map[string]func(eslint.Node){
		"TemplateLiteral": func(node eslint.Node) {
			for _, quasi := range eslint.GetNodes(node, "quasis") {
				// Empty lines have a semantic meaning if they're inside
				// template literals. Don't count these as empty lines.
				startLine, _ := eslint.LocStart(quasi)
				endLine, _ := eslint.LocEnd(quasi)
				for ignoredLine := startLine; ignoredLine < endLine; ignoredLine++ {
					templateLiteralLines[ignoredLine] = true
				}
			}
		},
		"Program:exit": func(node eslint.Node) {
			// First, the line numbers that are non-empty.
			nonEmptyLineNumbers := make([]int, 0, len(allLines))
			for index, line := range allLines {
				if jsTrim(line) != "" || templateLiteralLines[index+1] {
					nonEmptyLineNumbers = append(nonEmptyLineNumbers, index+1)
				}
			}
			// Add a value at the end to allow trailing empty lines to be checked.
			nonEmptyLineNumbers = append(nonEmptyLineNumbers, len(allLines)+1)

			lastLineNumber := 0
			for _, lineNumber := range nonEmptyLineNumbers {
				var messageID string
				var maxAllowed int

				switch {
				case lastLineNumber == 0:
					messageID = blankBeginningOfFile
					maxAllowed = maxBOF
				case lineNumber == len(allLines)+1:
					messageID = blankEndOfFile
					maxAllowed = maxEOF
				default:
					messageID = consecutiveBlank
					maxAllowed = max
				}

				if lineNumber-lastLineNumber-1 > maxAllowed {
					ln, ma, lnum := lastLineNumber, maxAllowed, lineNumber
					ctx.Report(eslint.Report{
						Node: node,
						Loc: eslint.LocOf(
							ln+ma+1, 0,
							lnum, 0,
						),
						MessageID: messageID,
						Data: map[string]any{
							"max":             ma,
							"pluralizedLines": pluralizedLines(ma),
						},
						Fix: func(f *eslint.Fixer) *eslint.Fix {
							rangeStart := sc.GetIndexFromLoc(ln+1, 0)

							// The end of the removal range is usually the
							// start index of the next line. However, at the
							// end of the file there is no next line, so the
							// end of the range is just the length of the text.
							lineNumberAfterRemovedLines := lnum - ma
							rangeEnd := 0
							if lineNumberAfterRemovedLines <= len(allLines) {
								rangeEnd = sc.GetIndexFromLoc(lineNumberAfterRemovedLines, 0)
							} else {
								rangeEnd = len(sc.Text())
							}

							return f.RemoveRange([2]int{rangeStart, rangeEnd})
						},
					})
				}

				lastLineNumber = lineNumber
			}
		},
	}
}

func pluralizedLines(maxAllowed int) string {
	if maxAllowed == 1 {
		return "line"
	}
	return "lines"
}

func intOption(v any, def int) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return def
}

// jsTrim trims exactly the characters JS's String.prototype.trim removes
// (WhiteSpace + LineTerminator), which is not the same set as Go's
// unicode.IsSpace (Go also treats U+0085 NEL as space; JS does not).
func jsTrim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		switch r {
		case '\t', '\n', '\v', '\f', '\r', ' ',
			0x00A0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
			return true
		}
		return r >= 0x2000 && r <= 0x200A
	})
}
