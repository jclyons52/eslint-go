// Package preferconst is a port of eslint/lib/rules/prefer-const.js.
//
// A `let` variable that is written exactly once (at its declaration) is
// reported, and the fix rewrites the `let` keyword to `const`. The rule works
// entirely off eslint-scope's Variable/Reference model, so it depends on the
// scope integration in SourceCode.
package preferconst

import (
	"reflect"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "prefer-const"

// Rule is a port of eslint/lib/rules/prefer-const.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Require `const` declarations for variables that are never reassigned after declared",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/prefer-const",
		},
		Fixable: "code",
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"destructuring":          map[string]any{"enum": []any{"any", "all"}, "default": "any"},
					"ignoreReadBeforeAssign": map[string]any{"type": "boolean", "default": false},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			"useConst": "'{{name}}' is never reassigned. Use 'const' instead.",
		},
	},
	Create: create,
}

// isPatternType mirrors the JS PATTERN_TYPE regexp
// /^(?:.+?Pattern|RestElement|SpreadProperty|ExperimentalRestProperty|Property)$/u.
func isPatternType(t string) bool {
	if len(t) > len("Pattern") && strings.HasSuffix(t, "Pattern") {
		return true
	}
	switch t {
	case "RestElement", "SpreadProperty", "ExperimentalRestProperty", "Property":
		return true
	}
	return false
}

// isDeclarationHostType mirrors DECLARATION_HOST_TYPE
// /^(?:Program|BlockStatement|StaticBlock|SwitchCase)$/u.
func isDeclarationHostType(t string) bool {
	switch t {
	case "Program", "BlockStatement", "StaticBlock", "SwitchCase":
		return true
	}
	return false
}

// isDestructuringHostType mirrors DESTRUCTURING_HOST_TYPE
// /^(?:VariableDeclarator|AssignmentExpression)$/u.
func isDestructuringHostType(t string) bool {
	return t == "VariableDeclarator" || t == "AssignmentExpression"
}

// isInitOfForStatement reports whether a declaration is `for (…;;)`'s init.
func isInitOfForStatement(node eslint.Node) bool {
	parent := eslint.Parent(node)
	return eslint.NodeType(parent) == "ForStatement" && eslint.SameNode(eslint.GetNode(parent, "init"), node)
}

// canBecomeVariableDeclaration reports whether an identifier's assignment can
// be turned into a declaration.
func canBecomeVariableDeclaration(identifier eslint.Node) bool {
	node := eslint.Parent(identifier)
	for node != nil && isPatternType(eslint.NodeType(node)) {
		node = eslint.Parent(node)
	}
	if node == nil {
		return false
	}
	switch eslint.NodeType(node) {
	case "VariableDeclarator":
		return true
	case "AssignmentExpression":
		parent := eslint.Parent(node)
		if eslint.NodeType(parent) != "ExpressionStatement" {
			return false
		}
		return isDeclarationHostType(eslint.NodeType(eslint.Parent(parent)))
	}
	return false
}

// isOuterVariableInDestructing reports whether a destructured name refers to an
// outer-scope variable or a function parameter.
func isOuterVariableInDestructing(name string, initScope *eslintscope.Scope) bool {
	for _, ref := range initScope.Through {
		if ref.Resolved != nil && ref.Resolved.Name == name {
			return true
		}
	}
	if variable := eslint.GetVariableByName(initScope, name); variable != nil {
		for _, def := range variable.Defs {
			if def.Type == eslintscope.VarParameter {
				return true
			}
		}
	}
	return false
}

// getDestructuringHost returns the VariableDeclarator/AssignmentExpression a
// write reference belongs to, or nil.
func getDestructuringHost(reference *eslintscope.Reference) eslint.Node {
	if !reference.IsWrite() {
		return nil
	}
	node := eslint.Parent(reference.Identifier)
	for node != nil && isPatternType(eslint.NodeType(node)) {
		node = eslint.Parent(node)
	}
	if node == nil || !isDestructuringHostType(eslint.NodeType(node)) {
		return nil
	}
	return node
}

// hasMemberExpressionAssignment reports whether a destructuring pattern assigns
// into a member expression (which cannot become a const declaration).
func hasMemberExpressionAssignment(node eslint.Node) bool {
	switch eslint.NodeType(node) {
	case "ObjectPattern":
		for _, prop := range eslint.GetNodes(node, "properties") {
			if prop == nil {
				continue
			}
			// Spread elements have `argument`; properties have `value`.
			target := eslint.GetNode(prop, "argument")
			if target == nil {
				target = eslint.GetNode(prop, "value")
			}
			if hasMemberExpressionAssignment(target) {
				return true
			}
		}
		return false
	case "ArrayPattern":
		for _, element := range eslint.GetNodes(node, "elements") {
			if element == nil {
				continue
			}
			if hasMemberExpressionAssignment(element) {
				return true
			}
		}
		return false
	case "AssignmentPattern":
		return hasMemberExpressionAssignment(eslint.GetNode(node, "left"))
	case "MemberExpression":
		return true
	}
	return false
}

// getIdentifierIfShouldBeConst returns the identifier to report for a variable,
// or nil when the variable must stay `let`.
func getIdentifierIfShouldBeConst(variable *eslintscope.Variable, ignoreReadBeforeAssign bool) eslint.Node {
	if variable.ESLintUsed && variable.Scope != nil && variable.Scope.Type == eslintscope.ScopeGlobal {
		return nil
	}

	var writer *eslintscope.Reference
	isReadBeforeInit := false

	for _, reference := range variable.References {
		if reference.IsWrite() {
			if writer != nil && !eslint.SameNode(writer.Identifier, reference.Identifier) {
				return nil // reassigned
			}

			if host := getDestructuringHost(reference); host != nil && eslint.Get(host, "left") != nil {
				leftNode := eslint.GetNode(host, "left")
				hasOuterVariables := false
				hasNonIdentifiers := false

				switch eslint.NodeType(leftNode) {
				case "ObjectPattern":
					for _, prop := range eslint.GetNodes(leftNode, "properties") {
						value := eslint.GetNode(prop, "value")
						if value == nil {
							continue
						}
						if isOuterVariableInDestructing(eslint.GetString(value, "name"), variable.Scope) {
							hasOuterVariables = true
						}
					}
					hasNonIdentifiers = hasMemberExpressionAssignment(leftNode)
				case "ArrayPattern":
					for _, element := range eslint.GetNodes(leftNode, "elements") {
						if element == nil {
							continue
						}
						if isOuterVariableInDestructing(eslint.GetString(element, "name"), variable.Scope) {
							hasOuterVariables = true
						}
					}
					hasNonIdentifiers = hasMemberExpressionAssignment(leftNode)
				}

				if hasOuterVariables || hasNonIdentifiers {
					return nil
				}
			}

			writer = reference
		} else if reference.IsRead() && writer == nil {
			if ignoreReadBeforeAssign {
				return nil
			}
			isReadBeforeInit = true
		}
	}

	shouldBeConst := writer != nil &&
		writer.From == variable.Scope &&
		canBecomeVariableDeclaration(writer.Identifier)

	if !shouldBeConst {
		return nil
	}
	if isReadBeforeInit {
		if len(variable.Defs) == 0 {
			return nil
		}
		return variable.Defs[0].Name
	}
	return writer.Identifier
}

// destructuringGroup is one VariableDeclarator/AssignmentExpression's group of
// reported identifiers, in insertion order.
type destructuringGroup struct {
	host  eslint.Node
	nodes []eslint.Node
}

// groupByDestructuring groups identifiers by the destructuring host they were
// written through, preserving the JS Map's insertion order.
func groupByDestructuring(variables []*eslintscope.Variable, ignoreReadBeforeAssign bool) []*destructuringGroup {
	var order []*destructuringGroup
	byHost := map[uintptr]*destructuringGroup{}

	for _, variable := range variables {
		identifier := getIdentifierIfShouldBeConst(variable, ignoreReadBeforeAssign)
		var prevID eslint.Node

		for _, reference := range variable.References {
			id := reference.Identifier
			if eslint.SameNode(id, prevID) {
				continue
			}
			prevID = id

			group := getDestructuringHost(reference)
			if group == nil {
				continue
			}
			key := reflect.ValueOf(group).Pointer()
			g, ok := byHost[key]
			if !ok {
				g = &destructuringGroup{host: group}
				byHost[key] = g
				order = append(order, g)
			}
			g.nodes = append(g.nodes, identifier)
		}
	}
	return order
}

// findUp finds the nearest ancestor (or self) of a given type, stopping when
// shouldStop matches.
func findUp(node eslint.Node, nodeType string, shouldStop func(eslint.Node) bool) eslint.Node {
	if node == nil || shouldStop(node) {
		return nil
	}
	if eslint.NodeType(node) == nodeType {
		return node
	}
	return findUp(eslint.Parent(node), nodeType, shouldStop)
}

// rawArray reads an array-valued property without dropping null entries, so
// `.length` arithmetic matches the JS.
func rawArray(n eslint.Node, key string) []any {
	arr, _ := eslint.Get(n, key).([]any)
	return arr
}

// nameOf models `node.name` in JS: the string for an Identifier, and JS
// `undefined` (Go nil) for any other node.
func nameOf(node eslint.Node) any {
	if s, ok := eslint.GetStringOK(node, "name"); ok {
		return s
	}
	return nil
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	options := ctx.OptionMap(0)
	shouldMatchAnyDestructuredVariable := true
	ignoreReadBeforeAssign := false
	if options != nil {
		if d, ok := options["destructuring"].(string); ok && d == "all" {
			shouldMatchAnyDestructuredVariable = false
		}
		if b, ok := options["ignoreReadBeforeAssign"].(bool); ok {
			ignoreReadBeforeAssign = b
		}
	}

	var variables []*eslintscope.Variable
	reportCount := 0
	var checkedID eslint.Node
	var checkedName any = ""

	endsWithStatement := func(node eslint.Node) bool {
		return strings.HasSuffix(eslint.NodeType(node), "Statement")
	}

	checkGroup := func(nodes []eslint.Node) {
		var nodesToReport []eslint.Node
		for _, n := range nodes {
			if n != nil {
				nodesToReport = append(nodesToReport, n)
			}
		}
		if len(nodes) == 0 {
			return
		}
		if !(shouldMatchAnyDestructuredVariable || len(nodesToReport) == len(nodes)) {
			return
		}

		varDeclParent := findUp(nodes[0], "VariableDeclaration", endsWithStatement)
		isVarDecParentNull := varDeclParent == nil

		if !isVarDecParentNull && len(rawArray(varDeclParent, "declarations")) > 0 {
			firstDeclaration := rawArray(varDeclParent, "declarations")[0]
			firstDeclarationNode, _ := firstDeclaration.(eslint.Node)
			if init := eslint.GetNode(firstDeclarationNode, "init"); init != nil {
				firstDecParent := eslint.Parent(init)
				if eslint.NodeType(firstDecParent) == "VariableDeclarator" {
					id := eslint.GetNode(firstDecParent, "id")
					if nameOf(id) != checkedName {
						checkedName = nameOf(id)
						reportCount = 0
					}
					if eslint.NodeType(id) == "ObjectPattern" {
						initName := nameOf(eslint.GetNode(firstDecParent, "init"))
						if initName != checkedName {
							checkedName = initName
							reportCount = 0
						}
					}
					if !eslint.SameNode(id, checkedID) {
						checkedID = id
						reportCount = 0
					}
				}
			}
		}

		declarations := rawArray(varDeclParent, "declarations")
		parentType := eslint.NodeType(eslint.Parent(varDeclParent))

		shouldFix := varDeclParent != nil &&
			(parentType == "ForInStatement" || parentType == "ForOfStatement" ||
				allInitialized(declarations)) &&
			len(nodesToReport) == len(nodes)

		if !isVarDecParentNull && len(declarations) != 1 {
			if len(declarations) >= 1 {
				reportCount += len(nodesToReport)

				totalDeclarationsCount := 0
				for _, declaration := range declarations {
					decl, _ := declaration.(eslint.Node)
					id := eslint.GetNode(decl, "id")
					switch eslint.NodeType(id) {
					case "ObjectPattern":
						totalDeclarationsCount += len(rawArray(id, "properties"))
					case "ArrayPattern":
						totalDeclarationsCount += len(rawArray(id, "elements"))
					default:
						totalDeclarationsCount++
					}
				}

				shouldFix = shouldFix && reportCount == totalDeclarationsCount
			}
		}

		for _, node := range nodesToReport {
			var fix func(*eslint.Fixer) *eslint.Fix
			if shouldFix {
				declParent := varDeclParent
				kind := eslint.GetString(declParent, "kind")
				fix = func(f *eslint.Fixer) *eslint.Fix {
					letKeywordToken := sc.GetFirstToken(declParent, eslint.TokenOpt{
						Filter: func(t eslint.Node) bool { return eslint.TokenValue(t) == kind },
					})
					if letKeywordToken == nil {
						return nil
					}
					return eslint.NewFixTracker(f, sc).
						RetainRange(eslint.MustRange(declParent)).
						ReplaceTextRange(eslint.MustRange(letKeywordToken), "const")
				}
			}
			ctx.Report(eslint.Report{
				Node:      node,
				MessageID: "useConst",
				Data:      map[string]any{"name": eslint.GetString(node, "name")},
				Fix:       fix,
			})
		}
	}

	return map[string]func(eslint.Node){
		"Program:exit": func(node eslint.Node) {
			for _, group := range groupByDestructuring(variables, ignoreReadBeforeAssign) {
				checkGroup(group.nodes)
			}
		},
		"VariableDeclaration": func(node eslint.Node) {
			if eslint.GetString(node, "kind") == "let" && !isInitOfForStatement(node) {
				variables = append(variables, ctx.DeclaredVariables(node)...)
			}
		},
	}
}

// allInitialized mirrors declarations.every(declaration => declaration.init).
func allInitialized(declarations []any) bool {
	for _, declaration := range declarations {
		decl, _ := declaration.(eslint.Node)
		if eslint.GetNode(decl, "init") == nil {
			return false
		}
	}
	return true
}
