package semispacing

import (
	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "semi-spacing"

// Rule is a port of eslint/lib/rules/semi-spacing.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type:       "layout",
		Deprecated: true,
		Docs: eslint.RuleDocs{
			Description: "Enforce consistent spacing before and after semicolons",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/semi-spacing",
		},
		Fixable: "whitespace",
		Messages: map[string]string{
			"unexpectedWhitespaceBefore": "Unexpected whitespace before semicolon.",
			"unexpectedWhitespaceAfter":  "Unexpected whitespace after semicolon.",
			"missingWhitespaceBefore":    "Missing whitespace before semicolon.",
			"missingWhitespaceAfter":     "Missing whitespace after semicolon.",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"before": map[string]any{"type": "boolean", "default": false},
					"after":  map[string]any{"type": "boolean", "default": true},
				},
				"additionalProperties": false,
			},
		},
	},
	Create: create,
}

// isTokenOnSameLine is astUtils.isTokenOnSameLine.
func isTokenOnSameLine(left, right eslint.Node) bool {
	if left == nil || right == nil {
		return false
	}
	leftLine, _ := eslint.LocEnd(left)
	rightLine, _ := eslint.LocStart(right)
	return leftLine == rightLine
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	requireSpaceBefore := false
	requireSpaceAfter := true

	// `typeof config === "object"`: an object option replaces both defaults,
	// and a missing key is `undefined` (falsy) rather than the documented
	// default.
	if config, ok := ctx.Option(0).(map[string]any); ok {
		before, _ := config["before"].(bool)
		after, _ := config["after"].(bool)
		requireSpaceBefore = before
		requireSpaceAfter = after
	}

	// hasLeadingSpace.
	hasLeadingSpace := func(token eslint.Node) bool {
		tokenBefore := sc.GetTokenBefore(token)
		return tokenBefore != nil &&
			isTokenOnSameLine(tokenBefore, token) &&
			sc.IsSpaceBetweenTokens(tokenBefore, token)
	}

	// hasTrailingSpace.
	hasTrailingSpace := func(token eslint.Node) bool {
		tokenAfter := sc.GetTokenAfter(token)
		return tokenAfter != nil &&
			isTokenOnSameLine(token, tokenAfter) &&
			sc.IsSpaceBetweenTokens(token, tokenAfter)
	}

	// isLastTokenInCurrentLine.
	isLastTokenInCurrentLine := func(token eslint.Node) bool {
		tokenAfter := sc.GetTokenAfter(token)
		return !(tokenAfter != nil && isTokenOnSameLine(token, tokenAfter))
	}

	// isFirstTokenInCurrentLine.
	isFirstTokenInCurrentLine := func(token eslint.Node) bool {
		tokenBefore := sc.GetTokenBefore(token)
		return !(tokenBefore != nil && isTokenOnSameLine(token, tokenBefore))
	}

	// isBeforeClosingParen: the next token is `}` or `)`.
	isBeforeClosingParen := func(token eslint.Node) bool {
		nextToken := sc.GetTokenAfter(token)
		return (nextToken != nil && eslint.IsClosingBraceToken(nextToken)) ||
			eslint.IsClosingParenToken(nextToken)
	}

	// checkSemicolonSpacing.
	checkSemicolonSpacing := func(token, node eslint.Node) {
		if token == nil || !eslint.IsSemicolonToken(token) {
			return
		}

		if hasLeadingSpace(token) {
			if !requireSpaceBefore {
				tokenBefore := sc.GetTokenBefore(token)
				startLine, startCol := eslint.LocEnd(tokenBefore)
				endLine, endCol := eslint.LocStart(token)
				loc := eslint.LocOf(startLine, startCol, endLine, endCol)
				from, to := eslint.End(tokenBefore), eslint.Start(token)

				ctx.Report(eslint.Report{
					Node:      node,
					Loc:       loc,
					MessageID: "unexpectedWhitespaceBefore",
					Fix: func(f *eslint.Fixer) *eslint.Fix {
						return f.RemoveRange([2]int{from, to})
					},
				})
			}
		} else if requireSpaceBefore {
			loc := eslint.Loc(token)
			target := token

			ctx.Report(eslint.Report{
				Node:      node,
				Loc:       loc,
				MessageID: "missingWhitespaceBefore",
				Fix: func(f *eslint.Fixer) *eslint.Fix {
					return f.InsertTextBefore(target, " ")
				},
			})
		}

		if !isFirstTokenInCurrentLine(token) && !isLastTokenInCurrentLine(token) && !isBeforeClosingParen(token) {
			if hasTrailingSpace(token) {
				if !requireSpaceAfter {
					tokenAfter := sc.GetTokenAfter(token)
					startLine, startCol := eslint.LocEnd(token)
					endLine, endCol := eslint.LocStart(tokenAfter)
					loc := eslint.LocOf(startLine, startCol, endLine, endCol)
					from, to := eslint.End(token), eslint.Start(tokenAfter)

					ctx.Report(eslint.Report{
						Node:      node,
						Loc:       loc,
						MessageID: "unexpectedWhitespaceAfter",
						Fix: func(f *eslint.Fixer) *eslint.Fix {
							return f.RemoveRange([2]int{from, to})
						},
					})
				}
			} else if requireSpaceAfter {
				loc := eslint.Loc(token)
				target := token

				ctx.Report(eslint.Report{
					Node:      node,
					Loc:       loc,
					MessageID: "missingWhitespaceAfter",
					Fix: func(f *eslint.Fixer) *eslint.Fix {
						return f.InsertTextAfter(target, " ")
					},
				})
			}
		}
	}

	// checkNode: the last token is assumed to be the semicolon.
	checkNode := func(node eslint.Node) {
		checkSemicolonSpacing(sc.GetLastToken(node), node)
	}

	return map[string]func(eslint.Node){
		"VariableDeclaration":      checkNode,
		"ExpressionStatement":      checkNode,
		"BreakStatement":           checkNode,
		"ContinueStatement":        checkNode,
		"DebuggerStatement":        checkNode,
		"DoWhileStatement":         checkNode,
		"ReturnStatement":          checkNode,
		"ThrowStatement":           checkNode,
		"ImportDeclaration":        checkNode,
		"ExportNamedDeclaration":   checkNode,
		"ExportAllDeclaration":     checkNode,
		"ExportDefaultDeclaration": checkNode,
		"PropertyDefinition":       checkNode,
		"ForStatement": func(node eslint.Node) {
			if init := eslint.GetNode(node, "init"); init != nil {
				checkSemicolonSpacing(sc.GetTokenAfter(init), node)
			}
			if test := eslint.GetNode(node, "test"); test != nil {
				checkSemicolonSpacing(sc.GetTokenAfter(test), node)
			}
		},
	}
}
