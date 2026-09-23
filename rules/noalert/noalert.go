package noalert

import (
	"regexp"

	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "no-alert"

// prohibitedIdentifier mirrors the original's /^(alert|confirm|prompt)$/u.
var prohibitedIdentifier = regexp.MustCompile(`^(alert|confirm|prompt)$`)

// Rule is a port of eslint/lib/rules/no-alert.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow the use of `alert`, `confirm`, and `prompt`",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-alert",
		},
		Messages: map[string]string{
			"unexpected": "Unexpected {{name}}.",
		},
		Schema: []any{},
	},
	Create: create,
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	return map[string]func(eslint.Node){
		"CallExpression": func(node eslint.Node) {
			callee := eslint.SkipChainExpression(eslint.GetNode(node, "callee"))
			currentScope := sc.Scope(node)

			switch eslint.NodeType(callee) {
			case "Identifier":
				// Without `window`.
				name := eslint.GetString(callee, "name")
				if !isShadowed(currentScope, callee) && prohibitedIdentifier.MatchString(name) {
					ctx.Report(eslint.Report{
						Node:      node,
						MessageID: "unexpected",
						Data:      map[string]any{"name": name},
					})
				}

			case "MemberExpression":
				if isGlobalThisReferenceOrGlobalWindow(currentScope, eslint.GetNode(callee, "object")) {
					name, _ := eslint.GetStaticPropertyName(callee)
					if prohibitedIdentifier.MatchString(name) {
						ctx.Report(eslint.Report{
							Node:      node,
							MessageID: "unexpected",
							Data:      map[string]any{"name": name},
						})
					}
				}
			}
		},
	}
}

// findReference mirrors ast-utils' local findReference: the single reference in
// `scope` whose identifier covers exactly the same range as `node`.
func findReference(scope *eslintscope.Scope, node eslint.Node) *eslintscope.Reference {
	if scope == nil || node == nil {
		return nil
	}
	target, _ := eslint.Range(node)

	var found *eslintscope.Reference
	count := 0
	for _, ref := range scope.References {
		if ref.Identifier == nil {
			continue
		}
		r, ok := eslint.Range(ref.Identifier)
		if ok && r[0] == target[0] && r[1] == target[1] {
			count++
			found = ref
		}
	}
	if count == 1 {
		return found
	}
	return nil
}

// isShadowed reports whether the identifier resolves to a declared variable.
func isShadowed(scope *eslintscope.Scope, node eslint.Node) bool {
	ref := findReference(scope, node)
	return ref != nil && ref.Resolved != nil && len(ref.Resolved.Defs) > 0
}

// isGlobalThisReferenceOrGlobalWindow reports whether a node is a reference to
// the global object: `this` in the global scope, `window`, or `globalThis`
// when that name is itself defined.
func isGlobalThisReferenceOrGlobalWindow(scope *eslintscope.Scope, node eslint.Node) bool {
	if scope != nil && scope.Type == eslintscope.ScopeGlobal && eslint.NodeType(node) == "ThisExpression" {
		return true
	}
	if eslint.NodeType(node) == "Identifier" {
		name := eslint.GetString(node, "name")
		if name == "window" ||
			(name == "globalThis" && eslint.GetVariableByName(scope, "globalThis") != nil) {
			return !isShadowed(scope, node)
		}
	}
	return false
}
