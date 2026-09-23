package nonewobject

import (
	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-new-object"

// Rule is a port of eslint/lib/rules/no-new-object.js (deprecated in ESLint
// v8.50.0 in favour of no-object-constructor).
//
// The original meta block has no `fixable` key — the rule reports only, it
// does not fix — so Fixable is left empty.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow `Object` constructors",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-new-object",
		},
		Deprecated: true,
		Messages: map[string]string{
			"preferLiteral": "The object literal notation {} is preferable.",
		},
		Schema: []any{},
	},
	Create: func(ctx *eslint.Context) map[string]func(eslint.Node) {
		sc := ctx.SourceCode
		return map[string]func(eslint.Node){
			"NewExpression": func(node eslint.Node) {
				// `node.callee.name` is undefined for a non-Identifier callee;
				// getVariableByName(scope, undefined) never matches a name, and
				// the comparison below is then false — GetString's "" keeps both
				// halves of the original behaviour.
				name := eslint.GetString(eslint.GetNode(node, "callee"), "name")

				variable := eslint.GetVariableByName(sc.Scope(node), name)
				if variable != nil && len(variable.Identifiers) > 0 {
					return
				}

				if name == "Object" {
					ctx.Report(eslint.Report{
						Node:      node,
						MessageID: "preferLiteral",
					})
				}
			},
		}
	},
}
