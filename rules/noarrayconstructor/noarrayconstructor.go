package noarrayconstructor

import (
	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-array-constructor"

// Rule is a port of eslint/lib/rules/no-array-constructor.js.
//
// The original meta has `hasSuggestions: true` and no `fixable`; every report
// carries a `suggest` array whose fix text is derived from the call's argument
// text (and, at the start of an expression statement where ASI would break, a
// leading `;`). The core LintMessage has no `suggestions` field
// (docs/porting-rules.md pitfall 6), so the suggestion is not emitted — the
// reported node, message, messageId and location are identical. The helper the
// original uses only for that fix text (`getArgumentsText` /
// `isStartOfExpressionStatement` / `needsPrecedingSemicolon`) is therefore not
// ported; see the package test for the affected corpus entries.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow `Array` constructors",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-array-constructor",
		},
		Messages: map[string]string{
			"preferLiteral":            "The array literal notation [] is preferable.",
			"useLiteral":               "Replace with an array literal.",
			"useLiteralAfterSemicolon": "Replace with an array literal, add preceding semicolon.",
		},
		Schema: []any{},
	},
	Create: create,
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	return map[string]func(eslint.Node){
		"CallExpression": check(ctx, sc),
		"NewExpression":  check(ctx, sc),
	}
}

func check(ctx *eslint.Context, sc *eslint.SourceCode) func(eslint.Node) {
	return func(node eslint.Node) {
		callee := eslint.GetNode(node, "callee")
		if eslint.NodeType(callee) != "Identifier" || eslint.GetString(callee, "name") != "Array" {
			return
		}

		// `Array(5)` builds a sparse array of the given length, not a dense
		// one, so a single non-spread argument is left alone.
		args := eslint.GetNodes(node, "arguments")
		if len(args) == 1 && eslint.NodeType(args[0]) != "SpreadElement" {
			return
		}

		// A predefined global has no declarations, so its variable carries no
		// identifiers; a shadowing declaration adds one.
		variable := eslint.GetVariableByName(sc.Scope(node), "Array")
		if variable != nil && len(variable.Identifiers) == 0 {
			ctx.Report(eslint.Report{
				Node:      node,
				MessageID: "preferLiteral",
			})
		}
	}
}
