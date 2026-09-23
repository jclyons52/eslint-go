package nonewwrappers

import (
	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-new-wrappers"

// wrapperObjects mirrors the original's ["String", "Number", "Boolean"].
var wrapperObjects = map[string]bool{
	"String":  true,
	"Number":  true,
	"Boolean": true,
}

// Rule is a port of eslint/lib/rules/no-new-wrappers.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow `new` operators with the `String`, `Number`, and `Boolean` objects",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-new-wrappers",
		},
		Messages: map[string]string{
			"noConstructor": "Do not use {{fn}} as a constructor.",
		},
		Schema: []any{},
	},
	Create: func(ctx *eslint.Context) map[string]func(eslint.Node) {
		sc := ctx.SourceCode
		return map[string]func(eslint.Node){
			"NewExpression": func(node eslint.Node) {
				// `node.callee.name` is undefined for a non-Identifier callee.
				name := eslint.GetString(eslint.GetNode(node, "callee"), "name")
				if !wrapperObjects[name] {
					return
				}

				// The JS rule reports when the resolved variable has no
				// identifiers, i.e. the name is the built-in wrapper object
				// (a language global). The Go core does not seed the global
				// scope with the ecmaVersion's language globals
				// (eslint/lib/source-code's getGlobalsForEcmaVersion), so the
				// built-in resolves to no variable at all: treat a missing
				// variable as that built-in global. A shadowing declaration —
				// the only other way to get a variable — still carries
				// identifiers and is left alone.
				variable := eslint.GetVariableByName(sc.Scope(node), name)
				if variable == nil || len(variable.Identifiers) == 0 {
					ctx.Report(eslint.Report{
						Node:      node,
						MessageID: "noConstructor",
						Data:      map[string]any{"fn": name},
					})
				}
			},
		}
	},
}
