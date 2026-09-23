package useisnan

import eslint "github.com/jclyons52/eslint-go"

// Name is the ESLint rule id.
const Name = "use-isnan"

// Rule is a port of eslint/lib/rules/use-isnan.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Require calls to `isNaN()` when checking for `NaN`",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/use-isnan",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enforceForSwitchCase": map[string]any{"type": "boolean", "default": true},
					"enforceForIndexOf":    map[string]any{"type": "boolean", "default": false},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			"comparisonWithNaN": "Use the isNaN function to compare with NaN.",
			"switchNaN":         "'switch(NaN)' can never match a case clause. Use Number.isNaN instead of the switch.",
			"caseNaN":           "'case NaN' can never match. Use Number.isNaN before the switch.",
			"indexOfNaN":        "Array prototype method '{{ methodName }}' cannot find NaN.",
		},
	},
	Create: create,
}

// isSpecificId mirrors astUtils.isSpecificId for the string-name form.
func isSpecificId(node eslint.Node, name string) bool {
	return eslint.NodeType(node) == "Identifier" && eslint.GetString(node, "name") == name
}

// isSpecificMemberAccess mirrors astUtils.isSpecificMemberAccess for the
// string-name form: the node is (optionally chained) `objectName.propertyName`.
func isSpecificMemberAccess(node eslint.Node, objectName, propertyName string) bool {
	checkNode := eslint.SkipChainExpression(node)

	if eslint.NodeType(checkNode) != "MemberExpression" {
		return false
	}
	if objectName != "" && !isSpecificId(eslint.GetNode(checkNode, "object"), objectName) {
		return false
	}
	if propertyName != "" {
		actualPropertyName, ok := eslint.GetStaticPropertyName(checkNode)
		if !ok || actualPropertyName != propertyName {
			return false
		}
	}
	return true
}

// isNaNIdentifier mirrors the rule's helper: `NaN` or `Number.NaN` (in any of
// the forms getStaticPropertyName understands, including `Number["NaN"]`).
func isNaNIdentifier(node eslint.Node) bool {
	return node != nil &&
		(isSpecificId(node, "NaN") || isSpecificMemberAccess(node, "Number", "NaN"))
}

// isComparisonOperator is the rule's /^(?:[<>]|[!=]=)=?$/u test.
func isComparisonOperator(op string) bool {
	switch op {
	case "<", ">", "<=", ">=", "==", "!=", "===", "!==":
		return true
	}
	return false
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
	// `!context.options[0] || context.options[0].enforceForSwitchCase` — note
	// that an options object without the key yields undefined (falsy), because
	// the schema default is not applied to context.options.
	opts := ctx.Option(0)
	enforceForSwitchCase := jsFalsy(opts)
	if !enforceForSwitchCase {
		if m, ok := opts.(map[string]any); ok {
			enforceForSwitchCase, _ = m["enforceForSwitchCase"].(bool)
		}
	}
	// `context.options[0] && context.options[0].enforceForIndexOf`
	enforceForIndexOf := false
	if m, ok := opts.(map[string]any); ok {
		enforceForIndexOf, _ = m["enforceForIndexOf"].(bool)
	}

	checkBinaryExpression := func(node eslint.Node) {
		if isComparisonOperator(eslint.GetString(node, "operator")) &&
			(isNaNIdentifier(eslint.GetNode(node, "left")) || isNaNIdentifier(eslint.GetNode(node, "right"))) {
			ctx.Report(eslint.Report{Node: node, MessageID: "comparisonWithNaN"})
		}
	}

	checkSwitchStatement := func(node eslint.Node) {
		if isNaNIdentifier(eslint.GetNode(node, "discriminant")) {
			ctx.Report(eslint.Report{Node: node, MessageID: "switchNaN"})
		}
		for _, switchCase := range eslint.GetNodes(node, "cases") {
			if isNaNIdentifier(eslint.GetNode(switchCase, "test")) {
				ctx.Report(eslint.Report{Node: switchCase, MessageID: "caseNaN"})
			}
		}
	}

	checkCallExpression := func(node eslint.Node) {
		callee := eslint.SkipChainExpression(eslint.GetNode(node, "callee"))

		if eslint.NodeType(callee) != "MemberExpression" {
			return
		}
		methodName, ok := eslint.GetStaticPropertyName(callee)
		args := eslint.GetNodes(node, "arguments")

		if ok && (methodName == "indexOf" || methodName == "lastIndexOf") &&
			len(args) == 1 && isNaNIdentifier(args[0]) {
			ctx.Report(eslint.Report{
				Node:      node,
				MessageID: "indexOfNaN",
				Data:      map[string]any{"methodName": methodName},
			})
		}
	}

	listeners := map[string]func(eslint.Node){
		"BinaryExpression": checkBinaryExpression,
	}
	if enforceForSwitchCase {
		listeners["SwitchStatement"] = checkSwitchStatement
	}
	if enforceForIndexOf {
		listeners["CallExpression"] = checkCallExpression
	}
	return listeners
}
