package noconsole

import (
	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "no-console"

// Rule is a port of eslint/lib/rules/no-console.js.
//
// The original meta has `hasSuggestions: true`; when the reported
// MemberExpression is the callee of an ExpressionStatement in a statement
// list, the report carries a `removeConsole` suggestion. The core LintMessage
// has no `suggestions` field (docs/porting-rules.md pitfall 6), so the
// suggestion is not emitted — every other observable property (reported node,
// message, loc, and which references are reported at all) is identical. See
// the package test for the affected corpus entries.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow the use of `console`",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-console",
		},
		Messages: map[string]string{
			"unexpected":    "Unexpected console statement.",
			"removeConsole": "Remove the console.{{ propertyName }}().",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"allow": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
						"minItems":    1,
						"uniqueItems": true,
					},
				},
				"additionalProperties": false,
			},
		},
	},
	Create: create,
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	var allowed []string
	if opts := ctx.OptionMap(0); opts != nil {
		if arr, ok := opts["allow"].([]any); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					allowed = append(allowed, s)
				}
			}
		}
	}

	sc := ctx.SourceCode

	return map[string]func(eslint.Node){
		"Program:exit": func(node eslint.Node) {
			scope := sc.Scope(node)
			consoleVar := eslint.GetVariableByName(scope, "console")
			shadowed := consoleVar != nil && len(consoleVar.Defs) > 0

			// `scope.through` holds every reference to an undefined variable,
			// so it is used when `console` itself is not a defined variable.
			var references []*eslintscope.Reference
			if consoleVar != nil {
				references = consoleVar.References
			} else if scope != nil {
				for _, ref := range scope.Through {
					if isConsole(ref) {
						references = append(references, ref)
					}
				}
			}

			if shadowed {
				return
			}
			for _, ref := range references {
				if isMemberAccessExceptAllowed(ref, allowed) {
					report(ctx, ref)
				}
			}
		},
	}
}

// isConsole reports whether the reference's identifier is named `console`.
func isConsole(ref *eslintscope.Reference) bool {
	if ref == nil {
		return false
	}
	return ref.Identifier != nil && eslint.GetString(ref.Identifier, "name") == "console"
}

// isAllowed reports whether the member expression's static property name is in
// the configured allow list. `propertyName &&` is the original's truthiness
// guard, so the empty string is not allowed even if it were listed.
func isAllowed(node eslint.Node, allowed []string) bool {
	propertyName, ok := eslint.GetStaticPropertyName(node)
	if !ok || propertyName == "" {
		return false
	}
	for _, a := range allowed {
		if a == propertyName {
			return true
		}
	}
	return false
}

// isMemberAccessExceptAllowed reports whether the reference is a member access
// whose object is `console` and whose property is not allowed.
func isMemberAccessExceptAllowed(ref *eslintscope.Reference, allowed []string) bool {
	if ref == nil {
		return false
	}
	node := ref.Identifier
	parent := eslint.Parent(node)

	return eslint.NodeType(parent) == "MemberExpression" &&
		eslint.SameNode(eslint.GetNode(parent, "object"), node) &&
		!isAllowed(parent, allowed)
}

// report reports the MemberExpression the reference is the object of.
func report(ctx *eslint.Context, ref *eslintscope.Reference) {
	node := eslint.Parent(ref.Identifier)
	ctx.Report(eslint.Report{
		Node:      node,
		MessageID: "unexpected",
	})
}
