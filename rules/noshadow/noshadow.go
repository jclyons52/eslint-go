// Package noshadow is a port of eslint/lib/rules/no-shadow.js.
//
// Every scope below the global scope is walked and each of its variables is
// checked against the variables of its outer scopes; a hit is reported at the
// shadowing identifier. The rule leans on scope identity, definition ranges and
// the `variableScope` chain, all of which come from eslint-scope.
package noshadow

import (
	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "no-shadow"

// Rule is a port of eslint/lib/rules/no-shadow.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow variable declarations from shadowing variables declared in the outer scope",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-shadow",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"builtinGlobals":         map[string]any{"type": "boolean", "default": false},
					"hoist":                  map[string]any{"enum": []any{"all", "functions", "never"}, "default": "functions"},
					"allow":                  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"ignoreOnInitialization": map[string]any{"type": "boolean", "default": false},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			"noShadow":       "'{{name}}' is already declared in the upper scope on line {{shadowedLine}} column {{shadowedColumn}}.",
			"noShadowGlobal": "'{{name}}' is already a global variable.",
		},
	},
	Create: create,
}

// isInRange reports whether a location falls inside a node's range (a nil node
// is never in range).
func isInRange(node eslint.Node, location int) bool {
	r, ok := eslint.Range(node)
	if !ok {
		return false
	}
	return r[0] <= location && location <= r[1]
}

// findSelfOrAncestor walks from a node up through its ancestry to the first
// node matching the predicate.
func findSelfOrAncestor(node eslint.Node, match func(eslint.Node) bool) eslint.Node {
	current := node
	for current != nil && !match(current) {
		current = eslint.Parent(current)
	}
	return current
}

// getOuterScope returns a function's outer scope, skipping the
// function-expression-name scope.
func getOuterScope(scope *eslintscope.Scope) *eslintscope.Scope {
	upper := scope.Upper
	if upper != nil && upper.Type == eslintscope.ScopeFunctionExpressionName {
		return upper.Upper
	}
	return upper
}

// isVariableScopeType mirrors eslint-scope's isVariableScopeType: the scope
// types that own a `variableScope`.
func isVariableScopeType(t string) bool {
	switch t {
	case eslintscope.ScopeGlobal, eslintscope.ScopeModule, eslintscope.ScopeFunction,
		eslintscope.ScopeClassFieldInitializer, eslintscope.ScopeClassStaticBlock:
		return true
	}
	return false
}

// variableScopeOf reimplements eslint-scope's `scope.variableScope`, which the
// Go port keeps private: a scope's variableScope is the nearest enclosing scope
// that is itself a variable scope.
func variableScopeOf(scope *eslintscope.Scope) *eslintscope.Scope {
	for s := scope; s != nil; s = s.Upper {
		if isVariableScopeType(s.Type) {
			return s
		}
	}
	return nil
}

// isConfiguredGlobal reports whether a global-scope variable came from the
// configuration rather than from the source.
//
// ESLint tests `"writeable" in shadowed`, i.e. whether the linter's
// configured-globals augmentation touched the variable. eslint-scope-go's
// Variable only carries a `Writeable` bool, so this is a private stand-in: a
// marked global, or a global the augmentation created from scratch (the only
// variable anywhere with neither identifiers nor definitions).
//
// Unlike no-redeclare this stand-in is exact here: the `writeable` test is only
// reached when `shadowed.identifiers.length === 0`, and a global with no
// identifiers is either augmentation-created (caught below) or an
// eslint-scope implicit global, which lives in `scope.implicit` and so is
// invisible to getVariableByName in both implementations.
func isConfiguredGlobal(variable *eslintscope.Variable) bool {
	if variable.Scope == nil || variable.Scope.Type != eslintscope.ScopeGlobal {
		return false
	}
	if variable.Writeable {
		return true
	}
	return len(variable.Identifiers) == 0 && len(variable.Defs) == 0
}

// isInitPatternNode reports whether a variable and the variable it shadows have
// the same initializer pattern ancestor (`ignoreOnInitialization`).
func isInitPatternNode(variable, shadowedVariable *eslintscope.Variable) bool {
	if len(shadowedVariable.Defs) == 0 {
		return false
	}
	outerDef := shadowedVariable.Defs[0]
	if outerDef == nil {
		return false
	}

	variableScope := variableScopeOf(variable.Scope)
	if variableScope == nil || variableScope.Block == nil {
		return false
	}
	blockType := eslint.NodeType(variableScope.Block)
	if (blockType != "ArrowFunctionExpression" && blockType != "FunctionExpression") ||
		getOuterScope(variableScope) != shadowedVariable.Scope {
		return false
	}

	fun := variableScope.Block
	callExpression := findSelfOrAncestor(eslint.Parent(fun), func(n eslint.Node) bool {
		return eslint.NodeType(n) == "CallExpression"
	})
	if callExpression == nil {
		return false
	}

	node := outerDef.Name
	location := eslint.End(callExpression)

	for node != nil {
		switch eslint.NodeType(node) {
		case "VariableDeclarator":
			if isInRange(eslint.GetNode(node, "init"), location) {
				return true
			}
			grandparent := eslint.Parent(eslint.Parent(node))
			switch eslint.NodeType(grandparent) {
			case "ForInStatement", "ForOfStatement":
				if isInRange(eslint.GetNode(grandparent, "right"), location) {
					return true
				}
			}
			return false
		case "AssignmentPattern":
			if isInRange(eslint.GetNode(node, "right"), location) {
				return true
			}
		default:
			switch eslint.NodeType(node) {
			case "FunctionDeclaration", "FunctionExpression", "ClassDeclaration", "ClassExpression",
				"ArrowFunctionExpression", "CatchClause", "ImportDeclaration", "ExportNamedDeclaration":
				return false
			}
		}
		node = eslint.Parent(node)
	}
	return false
}

// isDuplicatedClassNameVariable reports whether a variable is the class name
// inside a ClassDeclaration's own class scope (which the outer declaration
// already accounts for).
func isDuplicatedClassNameVariable(variable *eslintscope.Variable) bool {
	if variable.Scope == nil {
		return false
	}
	block := variable.Scope.Block
	if eslint.NodeType(block) != "ClassDeclaration" || len(variable.Identifiers) == 0 {
		return false
	}
	return eslint.SameNode(eslint.GetNode(block, "id"), variable.Identifiers[0])
}

// isOnInitializer reports whether a variable is declared inside the initializer
// of the variable it shadows (`var a = function a() {}`).
func isOnInitializer(variable, scopeVar *eslintscope.Variable) bool {
	outerScope := scopeVar.Scope

	var outer [2]int
	hasOuter := false
	if len(scopeVar.Defs) > 0 && scopeVar.Defs[0] != nil && scopeVar.Defs[0].Parent != nil {
		outer, hasOuter = eslint.Range(scopeVar.Defs[0].Parent)
	}

	var inner [2]int
	hasInner := false
	var innerDef *eslintscope.Definition
	if len(variable.Defs) > 0 {
		innerDef = variable.Defs[0]
	}
	if innerDef != nil && innerDef.Name != nil {
		inner, hasInner = eslint.Range(innerDef.Name)
	}

	return hasOuter && hasInner &&
		outer[0] < inner[0] && inner[1] < outer[1] &&
		((innerDef.Type == eslintscope.VarFunctionName &&
			eslint.NodeType(innerDef.Node) == "FunctionExpression") ||
			eslint.NodeType(innerDef.Node) == "ClassExpression") &&
		outerScope == variable.Scope.Upper
}

// getNameRange returns the range of a variable's definition name, if any.
func getNameRange(variable *eslintscope.Variable) ([2]int, bool) {
	if len(variable.Defs) == 0 || variable.Defs[0] == nil || variable.Defs[0].Name == nil {
		return [2]int{}, false
	}
	return eslint.Range(variable.Defs[0].Name)
}

// isInTdz reports whether a variable is declared before the variable it shadows
// (a temporal-dead-zone reference).
func isInTdz(variable, scopeVar *eslintscope.Variable, hoist string) bool {
	var outerDef *eslintscope.Definition
	if len(scopeVar.Defs) > 0 {
		outerDef = scopeVar.Defs[0]
	}
	inner, hasInner := getNameRange(variable)
	outer, hasOuter := getNameRange(scopeVar)

	return hasInner && hasOuter && inner[1] < outer[0] &&
		(hoist != "functions" || outerDef == nil ||
			eslint.NodeType(outerDef.Node) != "FunctionDeclaration")
}

// declaredLocation is getDeclaredLocation()'s return value.
type declaredLocation struct {
	global bool
	line   int
	column int
}

func getDeclaredLocation(variable *eslintscope.Variable) declaredLocation {
	if len(variable.Identifiers) > 0 {
		line, column := eslint.LocStart(variable.Identifiers[0])
		return declaredLocation{line: line, column: column + 1}
	}
	return declaredLocation{global: true}
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	var opts map[string]any
	if len(ctx.Options) > 0 {
		opts, _ = ctx.Options[0].(map[string]any)
	}

	builtinGlobals := false
	if opts != nil {
		if b, ok := opts["builtinGlobals"].(bool); ok {
			builtinGlobals = b
		}
	}
	hoist := "functions"
	if opts != nil {
		if s, ok := opts["hoist"].(string); ok && s != "" {
			hoist = s
		}
	}
	var allow []string
	if opts != nil {
		if arr, ok := opts["allow"].([]any); ok {
			for _, entry := range arr {
				if s, ok := entry.(string); ok {
					allow = append(allow, s)
				}
			}
		}
	}
	ignoreOnInitialization := false
	if opts != nil {
		if b, ok := opts["ignoreOnInitialization"].(bool); ok {
			ignoreOnInitialization = b
		}
	}

	isAllowed := func(variable *eslintscope.Variable) bool {
		for _, name := range allow {
			if name == variable.Name {
				return true
			}
		}
		return false
	}

	checkForShadows := func(scope *eslintscope.Scope) {
		for _, variable := range scope.Variables {
			// Skip "arguments" and the class-name variable of a
			// ClassDeclaration's own class scope.
			if len(variable.Identifiers) == 0 ||
				isDuplicatedClassNameVariable(variable) ||
				isAllowed(variable) {
				continue
			}

			shadowed := eslint.GetVariableByName(scope.Upper, variable.Name)
			if shadowed == nil {
				continue
			}
			if !(len(shadowed.Identifiers) > 0 || (builtinGlobals && isConfiguredGlobal(shadowed))) {
				continue
			}
			if isOnInitializer(variable, shadowed) {
				continue
			}
			if ignoreOnInitialization && isInitPatternNode(variable, shadowed) {
				continue
			}
			if hoist != "all" && isInTdz(variable, shadowed, hoist) {
				continue
			}

			location := getDeclaredLocation(shadowed)
			messageID := "noShadow"
			data := map[string]any{"name": variable.Name}
			if location.global {
				messageID = "noShadowGlobal"
			} else {
				data["shadowedLine"] = location.line
				data["shadowedColumn"] = location.column
			}

			ctx.Report(eslint.Report{
				Node:      variable.Identifiers[0],
				MessageID: messageID,
				Data:      data,
			})
		}
	}

	return map[string]func(eslint.Node){
		"Program:exit": func(node eslint.Node) {
			globalScope := ctx.ScopeOf(node)
			if globalScope == nil {
				return
			}
			// A stack walk over every scope below the global scope.
			stack := make([]*eslintscope.Scope, len(globalScope.ChildScopes))
			copy(stack, globalScope.ChildScopes)

			for len(stack) > 0 {
				scope := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				stack = append(stack, scope.ChildScopes...)
				checkForShadows(scope)
			}
		},
	}
}
