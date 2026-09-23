package eollast

import (
	"regexp"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "eol-last"

// Rule is a port of eslint/lib/rules/eol-last.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "layout",
		Docs: eslint.RuleDocs{
			Description: "Require or disallow newline at the end of files",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/eol-last",
		},
		Fixable: "whitespace",
		Schema: []any{
			map[string]any{
				"enum": []any{"always", "never", "unix", "windows"},
			},
		},
		Messages: map[string]string{
			"missing":    "Newline required at end of file but not found.",
			"unexpected": "Newline not allowed at end of file.",
		},
	},
	Create: create,
}

// finalEOLs is the rule's `/(?:\r?\n)+$/u` (the trailing run of linebreaks the
// "never" fixer removes).
var finalEOLs = regexp.MustCompile(`(?:\r?\n)+$`)

const (
	lf   = "\n"
	crlf = "\r\n"
)

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	return map[string]func(eslint.Node){
		"Program": func(node eslint.Node) {
			src := sc.Text()
			lines := sc.Lines()
			lastLine := lines[len(lines)-1]
			// A bare {line, column} point: report-translator turns it into
			// {start, end: null}, so the message carries no endLine/endColumn.
			location := eslint.StartLoc(len(lines), len(lastLine))
			endsWithNewline := strings.HasSuffix(src, lf)

			// Empty source is always valid: there is no content to terminate.
			if len(src) == 0 {
				return
			}

			mode := "always"
			if s, ok := ctx.Option(0).(string); ok && s != "" {
				mode = s
			}
			appendCRLF := false

			if mode == "unix" {
				// `"unix"` behaves exactly as `"always"`.
				mode = "always"
			}
			if mode == "windows" {
				// `"windows"` behaves as `"always"`, but appends CRLF in the fixer.
				mode = "always"
				appendCRLF = true
			}

			if mode == "always" && !endsWithNewline {
				eol := lf
				if appendCRLF {
					eol = crlf
				}
				end := len(src)
				ctx.Report(eslint.Report{
					Node:      node,
					Loc:       location,
					MessageID: "missing",
					Fix: func(f *eslint.Fixer) *eslint.Fix {
						return f.InsertTextAfterRange([2]int{0, end}, eol)
					},
				})
			} else if mode == "never" && endsWithNewline {
				secondLastLine := lines[len(lines)-2]
				ctx.Report(eslint.Report{
					Node: node,
					Loc: eslint.LocOf(len(lines)-1, len(secondLastLine),
						len(lines), 0),
					MessageID: "unexpected",
					Fix: func(f *eslint.Fixer) *eslint.Fix {
						match := finalEOLs.FindStringIndex(src)
						if match == nil {
							return nil
						}
						return f.ReplaceTextRange([2]int{match[0], len(src)}, "")
					},
				})
			}
		},
	}
}
