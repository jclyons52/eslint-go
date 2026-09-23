package noeval

import (
	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "no-eval"

// candidatesOfGlobalObject mirrors the original's frozen list of names the
// global object is reachable under.
var candidatesOfGlobalObject = []string{"global", "window", "globalThis"}

// thisScopeInfo mirrors the original's funcInfo stack entry: a `this` scope is
// a non-arrow function, a class static block, or a class field initializer.
type thisScopeInfo struct {
	upper              *thisScopeInfo
	node               eslint.Node
	strict             bool
	isTopLevelOfScript bool
	defaultThis        bool
	initialized        bool
}

// Rule is a port of eslint/lib/rules/no-eval.js.
//
// PLATFORM GAP: the Go parser (acorn-go) rejects the identifier `eval` in
// expression position outright ("The keyword 'eval' is reserved"), where real
// espree accepts it — so the direct-call, bare-reference and `allowIndirect`
// paths of this rule cannot be exercised against the oracle here. The
// global-object path (`window.eval`, `window.window.eval`, `global.eval`,
// `globalThis.eval`) and the `this.eval` path can, and are covered by the
// package test. See that file for the exact blocked cases.
//
// SELECTOR GAP: the original also registers the selector
// "PropertyDefinition > *.value" (class field initializers) to push a `this`
// scope. The core's listener map is keyed by plain node type, so that selector
// cannot be expressed; the ThisExpression handler applies the same rule
// directly instead (see inClassFieldInitializerValue). The scope is always
// strict (class bodies are), so the effect is that `this.eval` inside a field
// initializer is never reported — but omitting it entirely is NOT harmless: the
// enclosing scope (at top level of a script, the Program frame) would report
// instead. rules/noeval's corpus covers both directions.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow the use of `eval()`",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-eval",
		},
		Messages: map[string]string{
			"unexpected": "eval can be harmful.",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"allowIndirect": map[string]any{
						"type":    "boolean",
						"default": false,
					},
				},
				"additionalProperties": false,
			},
		},
	},
	Create: create,
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	// Boolean(context.options[0] && context.options[0].allowIndirect)
	allowIndirect := false
	if opts := ctx.OptionMap(0); opts != nil {
		if b, ok := opts["allowIndirect"].(bool); ok {
			allowIndirect = b
		}
	}

	sc := ctx.SourceCode
	var funcInfo *thisScopeInfo

	if allowIndirect {
		// Checks only direct calls to eval. It's simple!
		return map[string]func(eslint.Node){
			"CallExpression:exit": func(node eslint.Node) {
				callee := eslint.GetNode(node, "callee")

				// Optional call (`eval?.("code")`) is not direct eval.
				if !eslint.GetBool(node, "optional") && isSpecificId(callee, "eval") {
					report(ctx, callee)
				}
			},
		}
	}

	// enterThisScope pushes a `this` scope: strictness decides whether `this`
	// can be the global object, and `initialized` defers the (expensive)
	// default-binding analysis until a `this.eval` is actually seen.
	enterThisScope := func(node eslint.Node) {
		strict := false
		if scope := sc.Scope(node); scope != nil {
			strict = scope.IsStrict
		}
		funcInfo = &thisScopeInfo{
			upper:       funcInfo,
			node:        node,
			strict:      strict,
			initialized: strict,
		}
	}
	exitThisScope := func() {
		if funcInfo != nil {
			funcInfo = funcInfo.upper
		}
	}

	return map[string]func(eslint.Node){
		"CallExpression:exit": func(node eslint.Node) {
			callee := eslint.GetNode(node, "callee")
			if isSpecificId(callee, "eval") {
				report(ctx, callee)
			}
		},

		"Program": func(node eslint.Node) {
			scope := sc.Scope(node)
			features := map[string]any{}
			if m, ok := ctx.ParserOptions["ecmaFeatures"].(map[string]any); ok {
				features = m
			}
			globalReturn, _ := features["globalReturn"].(bool)

			strict := false
			if scope != nil && scope.IsStrict {
				strict = true
			}
			sourceType := eslint.GetString(node, "sourceType")
			if sourceType == "module" {
				strict = true
			}
			if globalReturn && scope != nil && len(scope.ChildScopes) > 0 && scope.ChildScopes[0].IsStrict {
				strict = true
			}

			funcInfo = &thisScopeInfo{
				upper:              nil,
				node:               node,
				strict:             strict,
				isTopLevelOfScript: sourceType != "module" && !globalReturn,
				defaultThis:        true,
				initialized:        true,
			}
		},

		"Program:exit": func(node eslint.Node) {
			globalScope := sc.Scope(node)

			exitThisScope()
			reportAccessingEval(ctx, globalScope)
			reportAccessingEvalViaGlobalObject(ctx, globalScope)
		},

		"FunctionDeclaration":      enterThisScope,
		"FunctionDeclaration:exit": func(eslint.Node) { exitThisScope() },
		"FunctionExpression":       enterThisScope,
		"FunctionExpression:exit":  func(eslint.Node) { exitThisScope() },
		"StaticBlock":              enterThisScope,
		"StaticBlock:exit":         func(eslint.Node) { exitThisScope() },

		"ThisExpression": func(node eslint.Node) {
			if !isMember(eslint.Parent(node), "eval") {
				return
			}

			// The original pushes a `this` scope for
			// "PropertyDefinition > *.value" (class field initializers); that
			// scope is always strict (class bodies are), so `this.eval` in a
			// field initializer is never reported. The core's listener map is
			// keyed by plain node type and cannot express that selector, so the
			// same rule is applied here instead.
			if inClassFieldInitializerValue(node) {
				return
			}

			if funcInfo == nil {
				return
			}

			// `this.eval` is found: check whether the value of `this` is the
			// global object.
			if !funcInfo.initialized {
				funcInfo.initialized = true
				funcInfo.defaultThis = isDefaultThisBinding(funcInfo.node, sc)
			}

			// `this` at the top level of scripts always refers to the global
			// object.
			if funcInfo.isTopLevelOfScript || (!funcInfo.strict && funcInfo.defaultThis) {
				report(ctx, eslint.Parent(node))
			}
		},
	}
}

// report reports a given Identifier or MemberExpression. The location of the
// report is always the `eval` identifier (or property), while the reported
// node is the enclosing CallExpression when the node is its callee.
func report(ctx *eslint.Context, node eslint.Node) {
	parent := eslint.Parent(node)

	locationNode := node
	if eslint.NodeType(node) == "MemberExpression" {
		locationNode = eslint.GetNode(node, "property")
	}

	reportNode := node
	if eslint.NodeType(parent) == "CallExpression" && eslint.SameNode(eslint.GetNode(parent, "callee"), node) {
		reportNode = parent
	}

	ctx.Report(eslint.Report{
		Node:      reportNode,
		Loc:       eslint.Loc(locationNode),
		MessageID: "unexpected",
	})
}

// reportAccessingEval reports all accesses of `eval` (excluding direct calls).
func reportAccessingEval(ctx *eslint.Context, globalScope *eslintscope.Scope) {
	variable := eslint.GetVariableByName(globalScope, "eval")
	if variable == nil {
		return
	}

	for _, reference := range variable.References {
		id := reference.Identifier
		if eslint.GetString(id, "name") == "eval" && !isCallee(id) {
			report(ctx, id)
		}
	}
}

// reportAccessingEvalViaGlobalObject reports accesses of `eval` via the global
// object (`window.eval`, `global.eval`, `globalThis.eval`, `window.window.eval`).
func reportAccessingEvalViaGlobalObject(ctx *eslint.Context, globalScope *eslintscope.Scope) {
	for _, name := range candidatesOfGlobalObject {
		variable := eslint.GetVariableByName(globalScope, name)
		if variable == nil {
			continue
		}

		for _, reference := range variable.References {
			identifier := reference.Identifier
			node := eslint.Parent(identifier)

			// To detect code like `window.window.eval`.
			for isMember(node, name) {
				node = eslint.Parent(node)
			}

			// Reports.
			if isMember(node, "eval") {
				report(ctx, node)
			}
		}
	}
}

// isSpecificId mirrors ast-utils' isSpecificId (checkText with a string).
func isSpecificId(node eslint.Node, name string) bool {
	return eslint.NodeType(node) == "Identifier" && eslint.GetString(node, "name") == name
}

// isCallee mirrors ast-utils' isCallee.
func isCallee(node eslint.Node) bool {
	parent := eslint.Parent(node)
	return eslint.NodeType(parent) == "CallExpression" &&
		eslint.SameNode(eslint.GetNode(parent, "callee"), node)
}

// isMember mirrors the rule's local helper: a MemberExpression whose static
// property name is `name`, whatever the object is.
func isMember(node eslint.Node, name string) bool {
	m := textMatcher(name)
	return isSpecificMemberAccess(node, nil, &m)
}
