// Package noelsereturn is a port of eslint/lib/rules/no-else-return.js.
//
// Besides the usual control-flow analysis, this rule's fix depends on scope
// analysis: removing the `else` block merges its block-scoped names into the
// enclosing scope, so the fix is only offered when that merge cannot collide
// with (or silently capture) an existing name. The fix also goes through
// FixTracker so it cannot fight another rule's fix in the same pass.
package noelsereturn

import (
	"regexp"

	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "no-else-return"

// Rule is a port of eslint/lib/rules/no-else-return.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow `else` blocks after `return` statements in `if` statements",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-else-return",
		},
		Fixable: "code",
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"allowElseIf": map[string]any{"type": "boolean", "default": true},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{"unexpected": "Unnecessary 'else' after 'return'."},
	},
	Create: create,
}

// statementListParents is astUtils.STATEMENT_LIST_PARENTS.
var statementListParents = map[string]bool{
	"Program": true, "BlockStatement": true, "StaticBlock": true, "SwitchCase": true,
}

// unsafeTokenStart is the JS /^[([/+`-]/u ASI-hazard test.
var unsafeTokenStart = regexp.MustCompile("^[([/+`-]")

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode

	// variableScope is Scope#variableScope (unexported in eslint-scope-go):
	// the nearest enclosing scope that holds variables.
	variableScope := func(scope *eslintscope.Scope) *eslintscope.Scope {
		for scope != nil && !isVariableScopeType(scope.Type) {
			scope = scope.Upper
		}
		return scope
	}

	// isSafeToDeclare: can the names be declared block-scoped in scope without
	// a redeclaration error or a silent rebinding?
	isSafeToDeclare := func(names []string, scope *eslintscope.Scope) bool {
		if len(names) == 0 {
			return true
		}
		if scope == nil {
			return true
		}
		contains := func(name string) bool {
			for _, n := range names {
				if n == name {
					return true
				}
			}
			return false
		}

		functionScope := variableScope(scope)

		/*
		 * Redeclaring any of the scope's declared variables (parameters, var,
		 * let/const/class/function in the scope) would be a syntax error,
		 * except for implicit variables such as `arguments`.
		 */
		for _, v := range scope.Variables {
			if len(v.Defs) > 0 && contains(v.Name) {
				return false
			}
		}

		// Redeclaring a catch variable would also be a syntax error.
		if scope != functionScope && scope.Upper != nil && scope.Upper.Type == eslintscope.ScopeCatch {
			for _, v := range scope.Upper.Variables {
				if contains(v.Name) {
					return false
				}
			}
		}

		/*
		 * Redeclaring an implicit variable would not be a syntax error, but
		 * declaring a new one with the same name would change references.
		 */
		for _, v := range scope.Variables {
			if len(v.Defs) == 0 && len(v.References) > 0 && contains(v.Name) {
				return false
			}
		}

		// Declaring a name already referenced from an upper scope would change
		// what that reference resolves to.
		for _, t := range scope.Through {
			if t.Identifier != nil && contains(eslint.GetString(t.Identifier, "name")) {
				return false
			}
		}

		/*
		 * In an inner scope, an uninitialized `var` declared inside the scope
		 * node (directly or in a descendant) is neither declared nor "through",
		 * so it has to be looked up on the function scope.
		 */
		if scope != functionScope && functionScope != nil {
			scopeNodeRange := eslint.MustRange(scope.Block)
			for _, v := range functionScope.Variables {
				if !contains(v.Name) {
					continue
				}
				for _, d := range v.Defs {
					if d.Node == nil {
						continue
					}
					r := eslint.MustRange(d.Node)
					if scopeNodeRange[0] <= r[0] && r[1] <= scopeNodeRange[1] {
						return false
					}
				}
			}
		}

		return true
	}

	// isSafeFromNameCollisions: is removing the `else` (and its braces) safe
	// from variable name collisions?
	isSafeFromNameCollisions := func(node eslint.Node, scope *eslintscope.Scope) bool {
		if eslint.NodeType(node) == "FunctionDeclaration" {
			// A conditional function declaration: hoisting is unpredictable.
			return false
		}
		if eslint.NodeType(node) != "BlockStatement" {
			return true
		}

		var elseBlockScope *eslintscope.Scope
		if scope != nil {
			for _, child := range scope.ChildScopes {
				if child.Block != nil && eslint.SameNode(child.Block, node) {
					elseBlockScope = child
					break
				}
			}
		}
		if elseBlockScope == nil {
			// No scope of its own: no possible collisions.
			return true
		}

		namesToCheck := make([]string, 0, len(elseBlockScope.Variables))
		for _, v := range elseBlockScope.Variables {
			namesToCheck = append(namesToCheck, v.Name)
		}
		return isSafeToDeclare(namesToCheck, scope)
	}

	checkForReturn := func(node eslint.Node) bool { return eslint.NodeType(node) == "ReturnStatement" }

	naiveHasReturn := func(node eslint.Node) bool {
		if eslint.NodeType(node) == "BlockStatement" {
			body := eslint.GetNodes(node, "body")
			if len(body) == 0 {
				return false
			}
			return checkForReturn(body[len(body)-1])
		}
		return checkForReturn(node)
	}

	hasElse := func(node eslint.Node) bool {
		return eslint.GetNode(node, "alternate") != nil && eslint.GetNode(node, "consequent") != nil
	}

	checkForIf := func(node eslint.Node) bool {
		return eslint.NodeType(node) == "IfStatement" && hasElse(node) &&
			naiveHasReturn(eslint.GetNode(node, "alternate")) &&
			naiveHasReturn(eslint.GetNode(node, "consequent"))
	}

	checkForReturnOrIf := func(node eslint.Node) bool {
		return checkForReturn(node) || checkForIf(node)
	}

	alwaysReturns := func(node eslint.Node) bool {
		if eslint.NodeType(node) == "BlockStatement" {
			for _, b := range eslint.GetNodes(node, "body") {
				if checkForReturnOrIf(b) {
					return true
				}
			}
			return false
		}
		return checkForReturnOrIf(node)
	}

	displayReport := func(elseNode eslint.Node) {
		currentScope := sc.Scope(eslint.Parent(elseNode))

		ctx.Report(eslint.Report{
			Node:      elseNode,
			MessageID: "unexpected",
			Fix: func(f *eslint.Fixer) *eslint.Fix {
				if !isSafeFromNameCollisions(elseNode, currentScope) {
					return nil
				}

				startToken := sc.GetFirstToken(elseNode)
				elseToken := sc.GetTokenBefore(startToken)
				source := sc.GetText(elseNode)
				lastIfToken := sc.GetTokenBefore(elseToken)

				var firstTokenOfElseBlock eslint.Node
				if eslint.IsPunctuatorToken(startToken, "{") {
					firstTokenOfElseBlock = sc.GetTokenAfter(startToken)
				} else {
					firstTokenOfElseBlock = startToken
				}

				/*
				 * If the if block has no braces and does not end in a
				 * semicolon, and the else block starts with (, [, /, +, ` or -,
				 * removing the `else` is not ASI-safe.
				 */
				ifBlockMaybeUnsafe := eslint.NodeType(eslint.GetNode(eslint.Parent(elseNode), "consequent")) != "BlockStatement" &&
					eslint.TokenValue(lastIfToken) != ";"
				elseBlockUnsafe := firstTokenOfElseBlock != nil &&
					unsafeTokenStart.MatchString(eslint.TokenValue(firstTokenOfElseBlock))

				if ifBlockMaybeUnsafe && elseBlockUnsafe {
					return nil
				}

				endToken := sc.GetLastToken(elseNode)
				lastTokenOfElseBlock := sc.GetTokenBefore(endToken)

				if eslint.TokenValue(lastTokenOfElseBlock) != ";" {
					nextToken := sc.GetTokenAfter(endToken)

					nextTokenUnsafe := nextToken != nil &&
						unsafeTokenStart.MatchString(eslint.TokenValue(nextToken))
					nextTokenOnSameLine := false
					if nextToken != nil {
						lastLine, _ := eslint.LocStart(lastTokenOfElseBlock)
						nextLine, _ := eslint.LocStart(nextToken)
						nextTokenOnSameLine = nextLine == lastLine
					}

					/*
					 * If the else block's contents do not end in a semicolon
					 * and the next token is unsafe or on the same line, ASI
					 * would not add one, so don't remove the else.
					 */
					if nextTokenUnsafe || (nextTokenOnSameLine && eslint.TokenValue(nextToken) != "}") {
						return nil
					}
				}

				fixedSource := source
				if eslint.IsPunctuatorToken(startToken, "{") {
					fixedSource = source[1 : len(source)-1]
				}

				/*
				 * Retaining the enclosing function keeps this fix from
				 * conflicting with other rules (and from colliding with
				 * another else block's fix in the same pass).
				 */
				return eslint.NewFixTracker(f, sc).
					RetainEnclosingFunction(elseNode).
					ReplaceTextRange([2]int{eslint.Start(elseToken), eslint.End(elseNode)}, fixedSource)
			},
		})
	}

	// checkIfWithoutElse: the if chain's final else after every branch returns
	// (used when allowElseIf is on, so `else if` itself is fine).
	checkIfWithoutElse := func(node eslint.Node) {
		parent := eslint.Parent(node)

		/*
		 * Fixing this would require splitting one statement into two, so no
		 * report when only one statement is allowed here.
		 */
		if parent == nil || !statementListParents[eslint.NodeType(parent)] {
			return
		}

		var consequents []eslint.Node
		var alternate eslint.Node

		for currentNode := node; eslint.NodeType(currentNode) == "IfStatement"; currentNode = eslint.GetNode(currentNode, "alternate") {
			if eslint.GetNode(currentNode, "alternate") == nil {
				return
			}
			consequents = append(consequents, eslint.GetNode(currentNode, "consequent"))
			alternate = eslint.GetNode(currentNode, "alternate")
		}

		for _, c := range consequents {
			if !alwaysReturns(c) {
				return
			}
		}
		displayReport(alternate)
	}

	// checkIfWithElse: any else after a returning consequent (allowElseIf off).
	checkIfWithElse := func(node eslint.Node) {
		parent := eslint.Parent(node)

		if parent == nil || !statementListParents[eslint.NodeType(parent)] {
			return
		}

		alternate := eslint.GetNode(node, "alternate")
		if alternate != nil && alwaysReturns(eslint.GetNode(node, "consequent")) {
			displayReport(alternate)
		}
	}

	allowElseIf := true
	if options := ctx.OptionMap(0); options != nil {
		if v, ok := options["allowElseIf"].(bool); ok && !v {
			allowElseIf = false
		}
	}

	handler := checkIfWithoutElse
	if !allowElseIf {
		handler = checkIfWithElse
	}

	return map[string]func(eslint.Node){"IfStatement:exit": handler}
}

// isVariableScopeType mirrors eslint-scope's variableScope types.
func isVariableScopeType(t string) bool {
	switch t {
	case eslintscope.ScopeGlobal, eslintscope.ScopeModule, eslintscope.ScopeFunction,
		eslintscope.ScopeClassFieldInitializer, eslintscope.ScopeClassStaticBlock:
		return true
	}
	return false
}
