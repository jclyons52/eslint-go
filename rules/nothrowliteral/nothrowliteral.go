package nothrowliteral

import eslint "github.com/jclyons52/eslint-go"

// Name is the ESLint rule id.
const Name = "no-throw-literal"

// Rule is a port of eslint/lib/rules/no-throw-literal.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow throwing literals as exceptions",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-throw-literal",
		},
		Schema: []any{},
		Messages: map[string]string{
			"object": "Expected an error object to be thrown.",
			"undef":  "Do not throw undefined.",
		},
	},
	Create: create,
}

// couldBeError is a port of astUtils.couldBeError: does the node have a
// possibility of being an Error object?
func couldBeError(node eslint.Node) bool {
	switch eslint.NodeType(node) {
	case "Identifier", "CallExpression", "NewExpression", "MemberExpression",
		"TaggedTemplateExpression", "YieldExpression", "AwaitExpression", "ChainExpression":
		return true // possibly an error object.

	case "AssignmentExpression":
		switch eslint.GetString(node, "operator") {
		case "=", "&&=":
			return couldBeError(eslint.GetNode(node, "right"))
		case "||=", "??=":
			return couldBeError(eslint.GetNode(node, "left")) || couldBeError(eslint.GetNode(node, "right"))
		}
		// All other assignment operators are mathematical assignment
		// operators (arithmetic or bitwise); they cannot evaluate to an
		// Error object.
		return false

	case "SequenceExpression":
		exprs := eslint.GetNodes(node, "expressions")
		return len(exprs) != 0 && couldBeError(exprs[len(exprs)-1])

	case "LogicalExpression":
		if eslint.GetString(node, "operator") == "&&" {
			return couldBeError(eslint.GetNode(node, "right"))
		}
		return couldBeError(eslint.GetNode(node, "left")) || couldBeError(eslint.GetNode(node, "right"))

	case "ConditionalExpression":
		return couldBeError(eslint.GetNode(node, "consequent")) || couldBeError(eslint.GetNode(node, "alternate"))
	}
	return false
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	return map[string]func(eslint.Node){
		"ThrowStatement": func(node eslint.Node) {
			argument := eslint.GetNode(node, "argument")
			if !couldBeError(argument) {
				ctx.Report(eslint.Report{Node: node, MessageID: "object"})
			} else if eslint.NodeType(argument) == "Identifier" {
				if eslint.GetString(argument, "name") == "undefined" {
					ctx.Report(eslint.Report{Node: node, MessageID: "undef"})
				}
			}
		},
	}
}
