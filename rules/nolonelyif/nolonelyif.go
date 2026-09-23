// Package nolonelyif is a port of eslint/lib/rules/no-lonely-if.js.
//
// The rule reports an `if` that is the only statement of an `else` block and
// rewrites it into an `else if` chain — unless the rewrite would change
// semantics (comments in the way, or ASI hazards), in which case the report
// carries no fix at all.
package nolonelyif

import (
	"regexp"
	"strings"
	"unicode"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-lonely-if"

// Rule is a port of eslint/lib/rules/no-lonely-if.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow `if` statements as the only statement in `else` blocks",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-lonely-if",
		},
		Fixable:  "code",
		Schema:   []any{},
		Messages: map[string]string{"unexpectedLonelyIf": "Unexpected if as the only statement in an else block."},
	},
	Create: create,
}

// unsafeTokenStart is the JS /^[([/+`-]/u ASI-hazard test.
var unsafeTokenStart = regexp.MustCompile("^[([/+`-]")

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode

	return map[string]func(eslint.Node){
		"IfStatement": func(node eslint.Node) {
			parent := eslint.Parent(node)
			grandparent := eslint.Parent(parent)

			if parent != nil && eslint.NodeType(parent) == "BlockStatement" &&
				len(eslint.GetNodes(parent, "body")) == 1 && grandparent != nil &&
				eslint.NodeType(grandparent) == "IfStatement" &&
				eslint.SameNode(parent, eslint.GetNode(grandparent, "alternate")) {

				ctx.Report(eslint.Report{
					Node:      node,
					MessageID: "unexpectedLonelyIf",
					Fix: func(f *eslint.Fixer) *eslint.Fix {
						openingElseCurly := sc.GetFirstToken(parent)
						closingElseCurly := sc.GetLastToken(parent)
						elseKeyword := sc.GetTokenBefore(openingElseCurly)
						tokenAfterElseBlock := sc.GetTokenAfter(closingElseCurly)
						consequent := eslint.GetNode(node, "consequent")
						lastIfToken := sc.GetLastToken(consequent)
						sourceText := sc.Text()

						// Don't fix when non-whitespace (e.g. comments) is in
						// the way.
						if jsTrim(sourceText[eslint.End(openingElseCurly):eslint.Start(node)]) != "" ||
							jsTrim(sourceText[eslint.End(node):eslint.Start(closingElseCurly)]) != "" {
							return nil
						}

						if eslint.NodeType(consequent) != "BlockStatement" &&
							eslint.TokenValue(lastIfToken) != ";" &&
							tokenAfterElseBlock != nil {

							consequentEndLine, _ := eslint.LocEnd(consequent)
							tokenAfterStartLine, _ := eslint.LocStart(tokenAfterElseBlock)
							lastValue := eslint.TokenValue(lastIfToken)

							if consequentEndLine == tokenAfterStartLine ||
								unsafeTokenStart.MatchString(eslint.TokenValue(tokenAfterElseBlock)) ||
								lastValue == "++" || lastValue == "--" {

								/*
								 * The `if` has no block and is not followed by a
								 * semicolon: fixing could change semantics
								 * through ASI, so don't.
								 */
								return nil
							}
						}

						prefix := ""
						if eslint.End(elseKeyword) == eslint.Start(openingElseCurly) {
							prefix = " "
						}
						return f.ReplaceTextRange(
							[2]int{eslint.Start(openingElseCurly), eslint.End(closingElseCurly)},
							prefix+sc.GetText(node))
					},
				})
			}
		},
	}
}

// jsTrim is JS String.prototype.trim: it strips WhiteSpace and LineTerminator
// (which includes U+00A0 and U+FEFF, and excludes U+0085).
func jsTrim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		switch r {
		case '\t', '\n', '\v', '\f', '\r', ' ', 0x00a0, 0xfeff:
			return true
		}
		return unicode.Is(unicode.Zs, r)
	})
}
