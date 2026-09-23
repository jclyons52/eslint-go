package nomixedspacesandtabs

import (
	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-mixed-spaces-and-tabs"

// Rule is a port of eslint/lib/rules/no-mixed-spaces-and-tabs.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "layout",
		Docs: eslint.RuleDocs{
			Description: "Disallow mixed spaces and tabs for indentation",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-mixed-spaces-and-tabs",
		},
		Deprecated: true,
		Schema: []any{
			map[string]any{"enum": []any{"smart-tabs", true, false}},
		},
		Messages: map[string]string{
			"mixedSpacesAndTabs": "Mixed spaces and tabs.",
		},
	},
	Create: create,
}

// matchMixedLength returns the length of the match the original rule's
// indentation regex would produce on line, or 0 when it does not match.
//
// The two patterns are
//
//	default:     /^(?=( +|\t+))\1(?:\t| )/u
//	smart-tabs:  /^(?=(\t*))\1(?=( +))\2\t/u
//
// Go's regexp package has neither lookahead nor backreferences, so both are
// rewritten as an explicit scan. Both patterns reduce to "a maximal leading run
// of one whitespace character, followed by the opposite whitespace character"
// (default) or "leading tabs, then a run of spaces, then a tab" (smart-tabs),
// because the greedy `+`/`*` makes the captured run maximal and the following
// literal therefore has to be the other character.
func matchMixedLength(line string, smartTabs bool) int {
	if smartTabs {
		// (\t*)( +)\t
		tabs := 0
		for tabs < len(line) && line[tabs] == '\t' {
			tabs++
		}
		spaces := 0
		for tabs+spaces < len(line) && line[tabs+spaces] == ' ' {
			spaces++
		}
		if spaces >= 1 && tabs+spaces < len(line) && line[tabs+spaces] == '\t' {
			return tabs + spaces + 1
		}
		return 0
	}

	if len(line) == 0 {
		return 0
	}
	first := line[0]
	if first != ' ' && first != '\t' {
		return 0
	}
	run := 0
	for run < len(line) && line[run] == first {
		run++
	}
	// The maximal run is followed by the opposite whitespace character.
	if run < len(line) && line[run] != first && (line[run] == ' ' || line[run] == '\t') {
		return run + 1
	}
	return 0
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode

	smartTabs := false
	switch v := ctx.Option(0).(type) {
	case bool:
		smartTabs = v
	case string:
		smartTabs = v == "smart-tabs"
	}

	return map[string]func(eslint.Node){
		"Program:exit": func(node eslint.Node) {
			lines := sc.Lines()
			ignoredCommentLines := map[int]bool{}

			// Add all lines except the first ones.
			for _, comment := range sc.GetAllComments() {
				startLine, _ := eslint.LocStart(comment)
				endLine, _ := eslint.LocEnd(comment)
				for i := startLine + 1; i <= endLine; i++ {
					ignoredCommentLines[i] = true
				}
			}

			for i, line := range lines {
				length := matchMixedLength(line, smartTabs)
				if length == 0 {
					continue
				}

				lineNumber := i + 1
				loc := eslint.LocOf(lineNumber, length-2, lineNumber, length)

				if ignoredCommentLines[lineNumber] {
					continue
				}

				containingNode := sc.GetNodeByRangeIndex(sc.GetIndexFromLoc(lineNumber, length-2))
				if containingNode != nil {
					t := eslint.NodeType(containingNode)
					if t == "Literal" || t == "TemplateElement" {
						continue
					}
				}

				ctx.Report(eslint.Report{
					Node:      node,
					Loc:       loc,
					MessageID: "mixedSpacesAndTabs",
				})
			}
		},
	}
}
