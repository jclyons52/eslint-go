package noconsole

import (
	"strings"

	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "no-console"

// Rule is a port of eslint/lib/rules/no-console.js.
//
// The original meta has `hasSuggestions: true`; when the reported
// MemberExpression is the callee of an ExpressionStatement in a statement
// list, the report carries a `removeConsole` suggestion (emitted by the JSON
// formatter, never applied by --fix).
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
	r := eslint.Report{
		Node:      node,
		MessageID: "unexpected",
	}
	if canProvideSuggestions(ctx.SourceCode, node) {
		// `getStaticPropertyName` returns null for a computed non-static
		// access, which JS interpolates as "null".
		var propertyName any
		if name, ok := eslint.GetStaticPropertyName(node); ok {
			propertyName = name
		}
		statement := eslint.Parent(eslint.Parent(node))
		r.Suggest = []eslint.Suggestion{{
			MessageID: "removeConsole",
			Data:      map[string]any{"propertyName": propertyName},
			Fix: func(f *eslint.Fixer) *eslint.Fix {
				return f.Remove(statement)
			},
		}}
	}
	ctx.Report(r)
}

// statementListParents is astUtils.STATEMENT_LIST_PARENTS.
var statementListParents = map[string]bool{
	"Program": true, "BlockStatement": true, "StaticBlock": true, "SwitchCase": true,
}

// maybeAsiHazard reports whether removing the statement would make the next
// statement continue onto it (ASI hazard).
func maybeAsiHazard(sc *eslint.SourceCode, node eslint.Node) bool {
	tokenBefore := sc.GetTokenBefore(node)
	tokenAfter := sc.GetTokenAfter(node)

	if tokenAfter == nil {
		return false
	}
	after := eslint.TokenValue(tokenAfter)
	if after == "" || !strings.ContainsRune("-[(/+`", rune(after[0])) {
		return false
	}
	if after == "++" || after == "--" {
		return false
	}
	if tokenBefore == nil {
		return false
	}
	before := eslint.TokenValue(tokenBefore)
	return before != ":" && before != ";" && before != "{"
}

// canProvideSuggestions is the original's canProvideSuggestions: only a bare
// `console.x()` statement in a statement list can be removed without changing
// the parse.
func canProvideSuggestions(sc *eslint.SourceCode, node eslint.Node) bool {
	parent := eslint.Parent(node)
	grandparent := eslint.Parent(parent)
	statement := eslint.Parent(grandparent)
	return eslint.NodeType(parent) == "CallExpression" &&
		eslint.SameNode(eslint.GetNode(parent, "callee"), node) &&
		eslint.NodeType(grandparent) == "ExpressionStatement" &&
		statementListParents[eslint.NodeType(statement)] &&
		!maybeAsiHazard(sc, grandparent)
}
