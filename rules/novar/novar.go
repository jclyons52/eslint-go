package novar

import (
	"regexp"

	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "no-var"

// Rule is a port of eslint/lib/rules/no-var.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Require `let` or `const` instead of `var`",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-var",
		},
		Schema:  []any{},
		Fixable: "code",
		Messages: map[string]string{
			"unexpectedVar": "Unexpected var, use let or const instead.",
		},
	},
	Create: create,
}

// scopeNodeRe is the SCOPE_NODE_TYPE pattern: the statement node types that
// own a block scope.
var scopeNodeRe = regexp.MustCompile(`^(?:Program|BlockStatement|SwitchStatement|ForStatement|ForInStatement|ForOfStatement)$`)

// statementListParents is astUtils.STATEMENT_LIST_PARENTS.
var statementListParents = map[string]bool{
	"Program":        true,
	"BlockStatement": true,
	"StaticBlock":    true,
	"SwitchCase":     true,
}

// isGlobal reports whether a variable lives in the global scope.
func isGlobal(variable *eslintscope.Variable) bool {
	return variable.Scope != nil && variable.Scope.Type == eslintscope.ScopeGlobal
}

// getEnclosingFunctionScope walks up to the nearest function or global scope.
func getEnclosingFunctionScope(scope *eslintscope.Scope) *eslintscope.Scope {
	current := scope
	for current != nil && current.Type != eslintscope.ScopeFunction && current.Type != eslintscope.ScopeGlobal {
		current = current.Upper
	}
	return current
}

// isReferencedInClosure reports whether the variable is referenced from a more
// specific function expression than the one it is declared in.
func isReferencedInClosure(variable *eslintscope.Variable) bool {
	enclosing := getEnclosingFunctionScope(variable.Scope)
	for _, reference := range variable.References {
		if getEnclosingFunctionScope(reference.From) != enclosing {
			return true
		}
	}
	return false
}

// isLoopAssignee reports whether the declaration is the left side of a
// for-in/for-of header.
func isLoopAssignee(node eslint.Node) bool {
	parent := eslint.Parent(node)
	switch eslint.NodeType(parent) {
	case "ForOfStatement", "ForInStatement":
		return eslint.SameNode(node, eslint.GetNode(parent, "left"))
	}
	return false
}

// isDeclarationInitialized reports whether every declarator has an initializer.
func isDeclarationInitialized(node eslint.Node) bool {
	for _, declarator := range eslint.GetNodes(node, "declarations") {
		if eslint.GetNode(declarator, "init") == nil {
			return false
		}
	}
	return true
}

// getScopeNode returns the nearest enclosing scope-owning statement.
func getScopeNode(node eslint.Node) eslint.Node {
	for current := node; current != nil; current = eslint.Parent(current) {
		if scopeNodeRe.MatchString(eslint.NodeType(current)) {
			return current
		}
	}
	return nil
}

// isRedeclared reports whether the variable has more than one definition.
func isRedeclared(variable *eslintscope.Variable) bool {
	return len(variable.Defs) >= 2
}

// isUsedFromOutsideOf builds the predicate that detects references outside the
// given scope node's range.
func isUsedFromOutsideOf(scopeNode eslint.Node) func(*eslintscope.Variable) bool {
	scope := eslint.MustRange(scopeNode)
	return func(variable *eslintscope.Variable) bool {
		for _, reference := range variable.References {
			id := eslint.MustRange(reference.Identifier)
			if id[0] < scope[0] || id[1] > scope[1] {
				return true
			}
		}
		return false
	}
}

// hasReferenceInTDZ builds the predicate that reports a variable having a
// reference inside the TDZ of the given initializer node.
func hasReferenceInTDZ(node eslint.Node) func(*eslintscope.Variable) bool {
	initStart, initEnd := eslint.Start(node), eslint.End(node)
	return func(variable *eslintscope.Variable) bool {
		if len(variable.Defs) == 0 {
			return false
		}
		id := variable.Defs[0].Name
		idStart := eslint.Start(id)

		var defaultValue eslint.Node
		if eslint.NodeType(eslint.Parent(id)) == "AssignmentPattern" {
			defaultValue = eslint.GetNode(eslint.Parent(id), "right")
		}
		var defaultStart, defaultEnd int
		if defaultValue != nil {
			defaultStart, defaultEnd = eslint.Start(defaultValue), eslint.End(defaultValue)
		}

		for _, reference := range variable.References {
			start, end := eslint.Start(reference.Identifier), eslint.End(reference.Identifier)
			if reference.Init {
				continue
			}
			if start < idStart {
				return true
			}
			if defaultValue != nil && start >= defaultStart && end <= defaultEnd {
				return true
			}
			if !eslint.IsFunction(node) && start >= initStart && end <= initEnd {
				return true
			}
		}
		return false
	}
}

// hasSelfReferenceInTDZ reports whether the declarator's initializer holds a
// reference to one of the variables the declarator declares.
func hasSelfReferenceInTDZ(ctx *eslint.Context, declarator eslint.Node) bool {
	init := eslint.GetNode(declarator, "init")
	if init == nil {
		return false
	}
	for _, variable := range ctx.DeclaredVariables(declarator) {
		if hasReferenceInTDZ(init)(variable) {
			return true
		}
	}
	return false
}

// hasNameDisallowedForLetDeclarations reports whether the variable is named
// `let`.
func hasNameDisallowedForLetDeclarations(variable *eslintscope.Variable) bool {
	return variable.Name == "let"
}

// isInLoop reports whether the node is inside a loop without crossing a
// function boundary (astUtils.isInLoop).
func isInLoop(node eslint.Node) bool {
	for current := node; current != nil && !eslint.IsFunction(current); current = eslint.Parent(current) {
		if eslint.IsLoop(current) {
			return true
		}
	}
	return false
}

// canFix mirrors the rule's canFix(): `var` may become `let` only when the
// conversion cannot change behaviour.
func canFix(ctx *eslint.Context, node eslint.Node) bool {
	variables := ctx.DeclaredVariables(node)
	scopeNode := getScopeNode(node)
	parent := eslint.Parent(node)

	if eslint.NodeType(parent) == "SwitchCase" {
		return false
	}
	for _, declarator := range eslint.GetNodes(node, "declarations") {
		if hasSelfReferenceInTDZ(ctx, declarator) {
			return false
		}
	}
	for _, variable := range variables {
		if isGlobal(variable) {
			return false
		}
	}
	for _, variable := range variables {
		if isRedeclared(variable) {
			return false
		}
	}
	usedFromOutside := isUsedFromOutsideOf(scopeNode)
	for _, variable := range variables {
		if usedFromOutside(variable) {
			return false
		}
	}
	for _, variable := range variables {
		if hasNameDisallowedForLetDeclarations(variable) {
			return false
		}
	}

	if isInLoop(node) {
		for _, variable := range variables {
			if isReferencedInClosure(variable) {
				return false
			}
		}
		if !isLoopAssignee(node) && !isDeclarationInitialized(node) {
			return false
		}
	}

	if !isLoopAssignee(node) &&
		!(eslint.NodeType(parent) == "ForStatement" && eslint.SameNode(eslint.GetNode(parent, "init"), node)) &&
		!statementListParents[eslint.NodeType(parent)] {
		return false
	}

	return true
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	return map[string]func(eslint.Node){
		"VariableDeclaration:exit": func(node eslint.Node) {
			if eslint.GetString(node, "kind") != "var" {
				return
			}
			ctx.Report(eslint.Report{
				Node:      node,
				MessageID: "unexpectedVar",
				Fix: func(f *eslint.Fixer) *eslint.Fix {
					varToken := ctx.SourceCode.GetFirstToken(node, eslint.TokenOpt{
						Filter: func(t eslint.Node) bool { return eslint.TokenValue(t) == "var" },
					})
					if varToken == nil || !canFix(ctx, node) {
						return nil
					}
					return f.ReplaceText(varToken, "let")
				},
			})
		},
	}
}
