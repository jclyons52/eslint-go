package nounsafenegation

import eslint "github.com/jclyons52/eslint-go"

// Name is the ESLint rule id.
const Name = "no-unsafe-negation"

// Rule is a port of eslint/lib/rules/no-unsafe-negation.js.
//
// NOTE (core gap): the original sets `meta.hasSuggestions: true` and reports a
// two-entry `suggest` array on every report. RuleMeta has no HasSuggestions
// field and eslint.Message has no `suggestions` field, so this port produces
// the same message/messageId/loc for every case but cannot emit the
// suggestions array — see the package test.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Disallow negating the left operand of relational operators",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-unsafe-negation",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enforceForOrderingRelations": map[string]any{"type": "boolean", "default": false},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			"unexpected":                   "Unexpected negating the left operand of '{{operator}}' operator.",
			"suggestNegatedExpression":     "Negate '{{operator}}' expression instead of its left operand. This changes the current behavior.",
			"suggestParenthesisedNegation": "Wrap negation in '()' to make the intention explicit. This preserves the current behavior.",
		},
	},
	Create: create,
}

// isInOrInstanceOfOperator mirrors the rule's helper.
func isInOrInstanceOfOperator(op string) bool {
	return op == "in" || op == "instanceof"
}

// isOrderingRelationalOperator mirrors the rule's helper.
func isOrderingRelationalOperator(op string) bool {
	return op == "<" || op == ">" || op == ">=" || op == "<="
}

// isNegation mirrors the rule's helper.
func isNegation(node eslint.Node) bool {
	return eslint.NodeType(node) == "UnaryExpression" && eslint.GetString(node, "operator") == "!"
}

// isParenthesised mirrors astUtils.isParenthesised(sourceCode, node): the
// tokens immediately before and after the node are `(` and `)` and they
// enclose it. The core's IsParenthesized is a different helper (it takes a
// repetition count), so the original is implemented here privately.
func isParenthesised(sc *eslint.SourceCode, node eslint.Node) bool {
	previousToken := sc.GetTokenBefore(node)
	nextToken := sc.GetTokenAfter(node)

	return previousToken != nil && nextToken != nil &&
		eslint.TokenValue(previousToken) == "(" && eslint.End(previousToken) <= eslint.Start(node) &&
		eslint.TokenValue(nextToken) == ")" && eslint.Start(nextToken) >= eslint.End(node)
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode

	// `options.enforceForOrderingRelations === true`
	enforceForOrderingRelations := false
	if m := ctx.OptionMap(0); m != nil {
		if v, ok := m["enforceForOrderingRelations"].(bool); ok {
			enforceForOrderingRelations = v
		}
	}

	return map[string]func(eslint.Node){
		"BinaryExpression": func(node eslint.Node) {
			operator := eslint.GetString(node, "operator")
			orderingRelationRuleApplies := enforceForOrderingRelations && isOrderingRelationalOperator(operator)
			left := eslint.GetNode(node, "left")

			if (isInOrInstanceOfOperator(operator) || orderingRelationRuleApplies) &&
				isNegation(left) && !isParenthesised(sc, left) {
				negationToken := sc.GetFirstToken(left)
				nodeEnd := eslint.End(node)
				leftText := sc.GetText(left)
				ctx.Report(eslint.Report{
					Node:      node,
					Loc:       eslint.Loc(left),
					MessageID: "unexpected",
					Data:      map[string]any{"operator": operator},
					Suggest: []eslint.Suggestion{
						{
							MessageID: "suggestNegatedExpression",
							Data:      map[string]any{"operator": operator},
							Fix: func(f *eslint.Fixer) *eslint.Fix {
								if negationToken == nil {
									return nil
								}
								fixRange := [2]int{eslint.End(negationToken), nodeEnd}
								return f.ReplaceTextRange(fixRange, "("+sc.GetTextRange(fixRange)+")")
							},
						},
						{
							MessageID: "suggestParenthesisedNegation",
							Fix: func(f *eslint.Fixer) *eslint.Fix {
								return f.ReplaceText(left, "("+leftText+")")
							},
						},
					},
				})
			}
		},
	}
}
