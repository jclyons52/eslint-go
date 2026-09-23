package eqeqeq

import eslint "github.com/jclyons52/eslint-go"

// Name is the ESLint rule id.
const Name = "eqeqeq"

// Rule is a port of eslint/lib/rules/eqeqeq.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Require the use of `===` and `!==`",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/eqeqeq",
		},
		Fixable: "code",
		Messages: map[string]string{
			"unexpected": "Expected '{{expectedOperator}}' and instead saw '{{actualOperator}}'.",
		},
		// The original schema is `{ anyOf: [ <always-variant>, <smart-variant> ] }`.
		// RuleMeta.Schema holds an array, so the two anyOf variants are stored
		// directly (the declaration is documentation only, never validated).
		Schema: []any{
			map[string]any{
				"type": "array",
				"items": []any{
					map[string]any{"enum": []any{"always"}},
					map[string]any{
						"type": "object",
						"properties": map[string]any{
							"null": map[string]any{"enum": []any{"always", "never", "ignore"}},
						},
						"additionalProperties": false,
					},
				},
				"additionalItems": false,
			},
			map[string]any{
				"type": "array",
				"items": []any{
					map[string]any{"enum": []any{"smart", "allow-null"}},
				},
				"additionalItems": false,
			},
		},
	},
	Create: create,
}

// isTypeOf mirrors the rule's isTypeOf.
func isTypeOf(node eslint.Node) bool {
	return eslint.NodeType(node) == "UnaryExpression" && eslint.GetString(node, "operator") == "typeof"
}

// isTypeOfBinary mirrors the rule's isTypeOfBinary.
func isTypeOfBinary(node eslint.Node) bool {
	return isTypeOf(eslint.GetNode(node, "left")) || isTypeOf(eslint.GetNode(node, "right"))
}

// literalTypeof is JS `typeof <literal>.value`, needed by
// areLiteralsAndSameType. The port's AST stores a BigInt literal's value as its
// digit string (with a separate `bigint` field), whereas JS holds a real
// BigInt — so the bigint case has to be detected explicitly, otherwise `1n`
// would look like a string and `1n == "1"` would be (wrongly) treated as a
// same-type comparison and fixed.
func literalTypeof(node eslint.Node) string {
	if s, ok := node["bigint"].(string); ok && s != "" {
		return "bigint"
	}
	switch eslint.Get(node, "value").(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	}
	return "object" // null, regex literals, and anything else non-primitive
}

// areLiteralsAndSameType mirrors the rule's helper.
func areLiteralsAndSameType(node eslint.Node) bool {
	left := eslint.GetNode(node, "left")
	right := eslint.GetNode(node, "right")
	return eslint.NodeType(left) == "Literal" && eslint.NodeType(right) == "Literal" &&
		literalTypeof(left) == literalTypeof(right)
}

// isNullCheck mirrors the rule's helper.
func isNullCheck(node eslint.Node) bool {
	return eslint.IsNullLiteral(eslint.GetNode(node, "right")) || eslint.IsNullLiteral(eslint.GetNode(node, "left"))
}

// firstTokenBetween is SourceCode.getFirstTokenBetween(left, right, filter) —
// the first token in [left.end, right.start] matching the filter (comments are
// not tokens, so they are skipped).
func firstTokenBetween(sc *eslint.SourceCode, left, right eslint.Node, filter func(eslint.Node) bool) eslint.Node {
	tokens := sc.GetTokensBetween(left, right, eslint.TokenOpt{Filter: filter})
	if len(tokens) == 0 {
		return nil
	}
	return tokens[0]
}

// jsFalsy mirrors JS `!value` for the JSON value shapes a rule option can be.
func jsFalsy(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case bool:
		return !t
	case float64:
		return t == 0
	case int:
		return t == 0
	case string:
		return t == ""
	}
	return false
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode

	configValue := ctx.Option(0)
	if jsFalsy(configValue) {
		configValue = "always"
	}
	config, _ := configValue.(string)

	// `config === "always" ? (options.null || "always") : "ignore"`
	nullOption := "ignore"
	if config == "always" {
		nullOption = "always"
		if m := ctx.OptionMap(1); m != nil {
			if s, ok := m["null"].(string); ok && s != "" {
				nullOption = s
			}
		}
	}
	enforceRuleForNull := nullOption == "always"
	enforceInverseRuleForNull := nullOption == "never"

	report := func(node eslint.Node, expectedOperator string) {
		operatorToken := firstTokenBetween(sc, eslint.GetNode(node, "left"), eslint.GetNode(node, "right"),
			func(token eslint.Node) bool {
				return eslint.TokenValue(token) == eslint.GetString(node, "operator")
			})

		ctx.Report(eslint.Report{
			Node:      node,
			Loc:       eslint.Loc(operatorToken),
			MessageID: "unexpected",
			Data: map[string]any{
				"expectedOperator": expectedOperator,
				"actualOperator":   eslint.GetString(node, "operator"),
			},
			Fix: func(f *eslint.Fixer) *eslint.Fix {
				// If the comparison is a `typeof` comparison or both sides are
				// literals with the same type, then it's safe to fix.
				if operatorToken != nil && (isTypeOfBinary(node) || areLiteralsAndSameType(node)) {
					return f.ReplaceText(operatorToken, expectedOperator)
				}
				return nil
			},
		})
	}

	return map[string]func(eslint.Node){
		"BinaryExpression": func(node eslint.Node) {
			isNull := isNullCheck(node)
			operator := eslint.GetString(node, "operator")

			if operator != "==" && operator != "!=" {
				if enforceInverseRuleForNull && isNull {
					report(node, operator[:len(operator)-1])
				}
				return
			}

			if config == "smart" && (isTypeOfBinary(node) || areLiteralsAndSameType(node) || isNull) {
				return
			}

			if !enforceRuleForNull && isNull {
				return
			}

			report(node, operator+"=")
		},
	}
}
