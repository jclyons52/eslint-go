package eslint

import (
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// scope.go — scope analysis integration. ESLint runs eslint-scope over the
// parsed AST before rules execute and hands rules a ScopeManager through
// SourceCode; eslint-scope-go provides the analysis itself.
//
// The Linter always analyzes with the same option set ESLint's Linter uses:
// ignoreEval: true, optimistic: false, fallback "iteration" (key iteration on
// node types missing from the visitor-keys table), and the program's real
// sourceType/ecmaVersion — otherwise scope-sensitive rules (no-undef,
// no-unused-vars, no-shadow, no-redeclare) would disagree with the oracle.

// analyzeScope runs eslint-scope over a parsed program.
func analyzeScope(ast Node, sourceType string, ecmaVersion int) *eslintscope.ScopeManager {
	if sourceType == "" {
		sourceType = "script"
	}
	return eslintscope.Analyze(ast, &eslintscope.Options{
		IgnoreEval:       true,
		SourceType:       sourceType,
		ECMAVersion:      ecmaVersion,
		ChildVisitorKeys: VisitorKeys,
		Fallback:         "iteration",
	})
}

// SetScopeManager attaches the analyzed scope manager.
func (s *SourceCode) SetScopeManager(sm *eslintscope.ScopeManager) { s.scopeManager = sm }

// ScopeManager returns the analyzed scope manager.
func (s *SourceCode) ScopeManager() *eslintscope.ScopeManager { return s.scopeManager }

// Scope returns the scope for a node, mirroring SourceCode.getScope: walk up
// from the node until a scope is acquired, preferring the innermost scope
// unless the node is the Program (then the outermost). A
// "function-expression-name" scope resolves to its first child, and a node
// with no scope at all falls back to the global scope.
func (s *SourceCode) Scope(currentNode Node) *eslintscope.Scope {
	if currentNode == nil {
		return nil
	}
	if s.scopeManager == nil {
		return nil
	}
	key := nodeKey(currentNode)
	if sc, ok := s.scopeCache[key]; ok {
		return sc
	}
	inner := NodeType(currentNode) != "Program"
	for node := currentNode; node != nil; node = Parent(node) {
		scope := s.scopeManager.Acquire(node, inner)
		if scope != nil {
			if scope.Type == eslintscope.ScopeFunctionExpressionName {
				if len(scope.ChildScopes) > 0 {
					s.scopeCache[key] = scope.ChildScopes[0]
					return scope.ChildScopes[0]
				}
			}
			s.scopeCache[key] = scope
			return scope
		}
	}
	var global *eslintscope.Scope
	if len(s.scopeManager.Scopes) > 0 {
		global = s.scopeManager.Scopes[0]
	}
	s.scopeCache[key] = global
	return global
}

// DeclaredVariables returns the variables a node declares
// (SourceCode.getDeclaredVariables).
func (s *SourceCode) DeclaredVariables(node Node) []*eslintscope.Variable {
	if s.scopeManager == nil || node == nil {
		return nil
	}
	return s.scopeManager.GetDeclaredVariables(node)
}

// MarkVariableAsUsed marks a variable used so no-unused-vars ignores it,
// mirroring SourceCode.markVariableAsUsed (ESM/CommonJS top-level scopes are
// searched when the global scope delegates to the Program).
func (s *SourceCode) MarkVariableAsUsed(name string, refNode Node) bool {
	if s.scopeManager == nil {
		return false
	}
	current := s.Scope(refNode)
	if current == nil {
		return false
	}
	initial := current
	if current.Type == eslintscope.ScopeGlobal && len(current.ChildScopes) > 0 &&
		SameNode(current.ChildScopes[0].Block, s.ast) {
		initial = current.ChildScopes[0]
	}
	for scope := initial; scope != nil; scope = scope.Upper {
		for _, v := range scope.Variables {
			if v.Name == name {
				v.ESLintUsed = true
				return true
			}
		}
	}
	return false
}
