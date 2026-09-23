// Package noredeclare is a port of eslint/lib/rules/no-redeclare.js.
//
// A scope variable with more than one declaration is reported once per
// declaration after the first. Declarations come from the variable's
// identifiers; with `builtinGlobals` (the default), a global that also exists
// in the configuration counts as an implicit first declaration, which turns a
// single declaration into a redeclaration.
package noredeclare

import (
	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "no-redeclare"

// Rule is a port of eslint/lib/rules/no-redeclare.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow variable redeclaration",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-redeclare",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"builtinGlobals": map[string]any{"type": "boolean", "default": true},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			"redeclared":          "'{{id}}' is already defined.",
			"redeclaredAsBuiltin": "'{{id}}' is already defined as a built-in global variable.",
			"redeclaredBySyntax":  "'{{id}}' is already defined by a variable declaration.",
		},
	},
	Create: create,
}

// declaration is one entry of iterateDeclarations().
type declaration struct {
	// typ is "builtin", "syntax" or "comment".
	typ string
	// node and loc are the reported node/location (builtin has neither).
	node eslint.Node
	loc  map[string]any
}

// isConfiguredGlobal reports whether a global-scope variable was defined by the
// configuration rather than by the source. ESLint expresses this directly as
// `variable.eslintImplicitGlobalSetting`, which the linter sets on every global
// it applies (the ecmaVersion builtins, `env` entries and the config's
// `globals`); the port mirrors it in eslint-scope-go's Variable.
func isConfiguredGlobal(variable *eslintscope.Variable) bool {
	switch variable.ESLintImplicitGlobalSetting {
	case "readonly", "writable":
		return true
	}
	return false
}

// iterateDeclarations yields a variable's declarations, in the order ESLint
// does: the implicit configured global first, then the syntax identifiers.
//
// The JS rule also yields one "comment" declaration per `/* global */`
// directive comment attached to the variable
// (`variable.eslintExplicitGlobalComments`). The core does not parse inline
// configuration comments yet, so that source of declarations cannot exist here.
func iterateDeclarations(variable *eslintscope.Variable, builtinGlobals bool) []declaration {
	var out []declaration
	if builtinGlobals && isConfiguredGlobal(variable) {
		out = append(out, declaration{typ: "builtin"})
	}
	for _, id := range variable.Identifiers {
		out = append(out, declaration{typ: "syntax", node: id, loc: eslint.Loc(id)})
	}
	return out
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	builtinGlobals := true
	if len(ctx.Options) > 0 {
		builtinGlobals = false
		if opts, ok := ctx.Options[0].(map[string]any); ok {
			if b, ok := opts["builtinGlobals"].(bool); ok {
				builtinGlobals = b
			}
		}
	}

	findVariablesInScope := func(scope *eslintscope.Scope) {
		for _, variable := range scope.Variables {
			declarations := iterateDeclarations(variable, builtinGlobals)
			if len(declarations) <= 1 {
				continue
			}
			declaration := declarations[0]
			extraDeclarations := declarations[1:]

			// A first declaration of a different type makes the extra ones
			// report the "detail" message instead of the plain one.
			detailMessageID := "redeclaredBySyntax"
			if declaration.typ == "builtin" {
				detailMessageID = "redeclaredAsBuiltin"
			}
			data := map[string]any{"id": variable.Name}

			for _, extra := range extraDeclarations {
				messageID := detailMessageID
				if extra.typ == declaration.typ {
					messageID = "redeclared"
				}
				ctx.Report(eslint.Report{
					Node:      extra.node,
					Loc:       extra.loc,
					MessageID: messageID,
					Data:      data,
				})
			}
		}
	}

	// checkForBlock only inspects a node whose scope is that node itself (in
	// ES5 some node types have no scope of their own).
	checkForBlock := func(node eslint.Node) {
		scope := ctx.ScopeOf(node)
		if scope == nil {
			return
		}
		if eslint.SameNode(scope.Block, node) {
			findVariablesInScope(scope)
		}
	}

	return map[string]func(eslint.Node){
		"Program": func(node eslint.Node) {
			scope := ctx.ScopeOf(node)
			if scope == nil {
				return
			}

			findVariablesInScope(scope)

			// Node.js / ES modules have a special scope whose block is also the
			// Program node; check it too.
			if scope.Type == eslintscope.ScopeGlobal &&
				len(scope.ChildScopes) > 0 &&
				eslint.SameNode(scope.Block, scope.ChildScopes[0].Block) {
				findVariablesInScope(scope.ChildScopes[0])
			}
		},

		"FunctionDeclaration":     checkForBlock,
		"FunctionExpression":      checkForBlock,
		"ArrowFunctionExpression": checkForBlock,

		"StaticBlock": checkForBlock,

		"BlockStatement":  checkForBlock,
		"ForStatement":    checkForBlock,
		"ForInStatement":  checkForBlock,
		"ForOfStatement":  checkForBlock,
		"SwitchStatement": checkForBlock,
	}
}
