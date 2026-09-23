package nodupeargs

import (
	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "no-dupe-args"

// Rule is a port of eslint/lib/rules/no-dupe-args.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Disallow duplicate arguments in `function` definitions",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-dupe-args",
		},
		Schema: []any{},
		Messages: map[string]string{
			"unexpected": "Duplicate param '{{name}}'.",
		},
	},
	Create: create,
}

// isParameter mirrors the original's `def.type === "Parameter"` check.
func isParameter(def *eslintscope.Definition) bool {
	return def.Type == eslintscope.VarParameter
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	checkParams := func(node eslint.Node) {
		variables := ctx.DeclaredVariables(node)

		for _, variable := range variables {
			// Checks and reports duplications.
			count := 0
			for _, def := range variable.Defs {
				if isParameter(def) {
					count++
				}
			}
			if count >= 2 {
				ctx.Report(eslint.Report{
					Node:      node,
					MessageID: "unexpected",
					Data:      map[string]any{"name": variable.Name},
				})
			}
		}
	}

	return map[string]func(eslint.Node){
		"FunctionDeclaration": checkParams,
		"FunctionExpression":  checkParams,
	}
}
