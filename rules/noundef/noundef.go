package noundef

import eslint "github.com/jclyons52/eslint-go"

// Name is the ESLint rule id.
const Name = "no-undef"

// Rule is a port of eslint/lib/rules/no-undef.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Disallow the use of undeclared variables unless mentioned in `/*global */` comments",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-undef",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"typeof": map[string]any{"type": "boolean", "default": false},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			"undef": "'{{name}}' is not defined.",
		},
	},
	Create: create,
}

// hasTypeOfOperator reports whether an identifier is the argument of typeof.
func hasTypeOfOperator(node eslint.Node) bool {
	parent := eslint.Parent(node)
	return eslint.NodeType(parent) == "UnaryExpression" && eslint.GetString(parent, "operator") == "typeof"
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	considerTypeOf := ctx.OptionBool(0, false)
	if opts := ctx.OptionMap(0); opts != nil {
		if t, ok := opts["typeof"].(bool); ok {
			considerTypeOf = t
		}
	} else if ctx.OptionBool(0, false) {
		considerTypeOf = true
	}

	return map[string]func(eslint.Node){
		"Program:exit": func(node eslint.Node) {
			globalScope := ctx.ScopeOf(node)
			for _, ref := range globalScope.Through {
				identifier := ref.Identifier
				if !considerTypeOf && hasTypeOfOperator(identifier) {
					continue
				}
				ctx.Report(eslint.Report{
					Node:      identifier,
					MessageID: "undef",
					Data:      identifier,
				})
			}
		},
	}
}
