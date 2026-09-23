package notrailingspaces

import (
	"regexp"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-trailing-spaces"

// Rule is a port of eslint/lib/rules/no-trailing-spaces.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "layout",
		Docs: eslint.RuleDocs{
			Description: "Disallow trailing whitespace at the end of lines",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-trailing-spaces",
		},
		Fixable: "whitespace",
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"skipBlankLines":  map[string]any{"type": "boolean", "default": false},
					"ignoreComments":  map[string]any{"type": "boolean", "default": false},
					"additionalProps": nil,
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{"trailingSpace": "Trailing spaces not allowed."},
	},
	Create: create,
}

// The character class of "blank" characters the rule treats as trailing space
// (source: no-trailing-spaces.js BLANK_CLASS, with Go regexp \x{} escapes).
const blankClass = `[ \t\x{00a0}\x{2000}-\x{200b}\x{3000}]`

var nonBlankRe = regexp.MustCompile(blankClass + `+$`)

var skipBlankRe = regexp.MustCompile(`^` + blankClass + `*$`)

var linebreakRe = regexp.MustCompile(`\r\n|[\r\n\x{2028}\x{2029}]`)

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	options := ctx.OptionMap(0)
	skipBlankLines := false
	ignoreComments := false
	if options != nil {
		skipBlankLines, _ = options["skipBlankLines"].(bool)
		ignoreComments, _ = options["ignoreComments"].(bool)
	}

	return map[string]func(eslint.Node){
		"Program": func(node eslint.Node) {
			lines := sc.Lines()
			linebreaks := linebreakRe.FindAllString(sc.Text(), -1)
			commentLineNumbers := map[int]bool{}
			for _, comment := range sc.GetAllComments() {
				startLine, _ := eslint.LocStart(comment)
				endLine := startLine
				if eslint.NodeType(comment) == eslint.CommentBlock {
					endLine, _ = eslint.LocEnd(comment)
					endLine--
				} else {
					endLine, _ = eslint.LocEnd(comment)
				}
				for i := startLine; i <= endLine; i++ {
					commentLineNumbers[i] = true
				}
			}

			totalLength := 0
			for i, line := range lines {
				lineNumber := i + 1
				linebreakLength := 1
				if i < len(linebreaks) && linebreaks[i] != "" {
					linebreakLength = len(linebreaks[i])
				}
				lineLength := len(line) + linebreakLength

				loc := nonBlankRe.FindStringIndex(line)
				if loc != nil {
					location := eslint.LocOf(lineNumber, loc[0], lineNumber, lineLength-linebreakLength)
					rangeStart := totalLength + loc[0]
					rangeEnd := totalLength + lineLength - linebreakLength
					containing := sc.GetNodeByRangeIndex(rangeStart)

					if eslint.NodeType(containing) == "TemplateElement" {
						parent := eslint.Parent(containing)
						if parent != nil && rangeStart > eslint.Start(parent) && rangeEnd < eslint.End(parent) {
							totalLength += lineLength
							continue
						}
					}

					if skipBlankLines && skipBlankRe.MatchString(line) {
						totalLength += lineLength
						continue
					}

					if !ignoreComments || !commentLineNumbers[lineNumber] {
						fixRange := [2]int{rangeStart, rangeEnd}
						ctx.Report(eslint.Report{
							Node:      node,
							Loc:       location,
							MessageID: "trailingSpace",
							Fix: func(f *eslint.Fixer) *eslint.Fix {
								return f.RemoveRange(fixRange)
							},
						})
					}
				}

				totalLength += lineLength
			}
		},
	}
}
