package noeval

// this_binding.go — private copies of the ast-utils helpers the rule reaches
// through but which the core does not export:
//
//   - isSpecificMemberAccess / isSpecificId (the rule's `isMember`)
//   - isDefaultThisBinding (the `this.eval` path) plus its private helpers
//     (isES5Constructor, startsWithUpperCase, hasJSDocThisTag and the
//     JSDoc/comment lookups it needs, isNullOrUndefined, isReflectApply,
//     isArrayFromMethod, isMethodWhichHasThisArg)
//
// Ported from oracle/node_modules/eslint/lib/rules/utils/ast-utils.js and
// lib/source-code/{source-code.js,token-store/index.js}. The shared pieces
// (GetStaticPropertyName, SkipChainExpression, GetUpperFunction, …) come from
// the core.

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	eslint "github.com/jclyons52/eslint-go"
)

// nameMatcher is the Go stand-in for ast-utils' `string | RegExp | null`
// argument: a nil matcher is the JS `null` (check nothing), a textMatcher is a
// string (exact compare) and a regexMatcher is a RegExp (test).
type nameMatcher interface {
	match(s string) bool
}

// textMatcher mirrors checkText's string branch (a pointer so it satisfies
// nameMatcher, matching the rule's `m := textMatcher(name); &m` idiom).
type textMatcher string

func (m *textMatcher) match(s string) bool { return s == string(*m) }

// text returns a nameMatcher for an exact string, mirroring checkText's
// `typeof expected === "string"` branch.
func text(s string) *textMatcher {
	m := textMatcher(s)
	return &m
}

// regexMatcher mirrors checkText's RegExp branch.
type regexMatcher struct{ re *regexp.Regexp }

func (m *regexMatcher) match(s string) bool { return m.re.MatchString(s) }

var (
	// /^(?:bind|call|apply)$/u
	bindOrCallOrApplyPattern = &regexMatcher{regexp.MustCompile(`^(?:bind|call|apply)$`)}
	// /Array$/u
	arrayOrTypedArrayPattern = &regexMatcher{regexp.MustCompile(`Array$`)}
	// /^(?:every|filter|find(?:Last)?(?:Index)?|flatMap|forEach|map|some)$/u
	arrayMethodWithThisArgPattern = &regexMatcher{
		regexp.MustCompile(`^(?:every|filter|find(?:Last)?(?:Index)?|flatMap|forEach|map|some)$`),
	}
	// /^[\s*]*@this/mu
	thisTagPattern = regexp.MustCompile(`(?m)^[\s*]*@this`)
)

// isSpecificId mirrors ast-utils' isSpecificId: an Identifier whose name
// matches `name` (string or pattern).
func isSpecificIdMatcher(node eslint.Node, name nameMatcher) bool {
	return eslint.NodeType(node) == "Identifier" && name != nil && name.match(eslint.GetString(node, "name"))
}

// isSpecificMemberAccess mirrors ast-utils' isSpecificMemberAccess. objectName
// and propertyName are `string | RegExp | null`; nil means "don't check".
func isSpecificMemberAccess(node eslint.Node, objectName, propertyName nameMatcher) bool {
	checkNode := eslint.SkipChainExpression(node)

	if eslint.NodeType(checkNode) != "MemberExpression" {
		return false
	}

	if objectName != nil && !isSpecificIdMatcher(eslint.GetNode(checkNode, "object"), objectName) {
		return false
	}

	if propertyName != nil {
		actualPropertyName, ok := eslint.GetStaticPropertyName(checkNode)

		// typeof actualPropertyName !== "string" is the `ok == false` case.
		if !ok || !propertyName.match(actualPropertyName) {
			return false
		}
	}

	return true
}

// isDefaultThisBinding mirrors ast-utils' isDefaultThisBinding(node,
// sourceCode) with the default capIsConstructor = true.
func isDefaultThisBinding(node eslint.Node, sc *eslint.SourceCode) bool {

	/*
	 * Class field initializers are implicit functions, but ESTree doesn't have
	 * the AST node of field initializers. Therefore, an expression node at
	 * `PropertyDefinition#value` is a function. In this case, `this` is always
	 * not default binding.
	 */
	parentOfNode := eslint.Parent(node)
	if eslint.NodeType(parentOfNode) == "PropertyDefinition" &&
		eslint.SameNode(eslint.GetNode(parentOfNode, "value"), node) {
		return false
	}

	// Class static blocks are implicit functions. In this case, `this` is
	// always not default binding.
	if eslint.NodeType(node) == "StaticBlock" {
		return false
	}

	if isES5Constructor(node) || hasJSDocThisTag(node, sc) {
		return false
	}

	isAnonymous := eslint.GetNode(node, "id") == nil
	currentNode := node

	for currentNode != nil {
		parent := eslint.Parent(currentNode)

		switch eslint.NodeType(parent) {

		/*
		 * Looks up the destination.
		 * e.g., obj.foo = nativeFoo || function foo() { ... };
		 */
		case "LogicalExpression", "ConditionalExpression", "ChainExpression":
			currentNode = parent

		/*
		 * If the upper function is IIFE, checks the destination of the return
		 * value.
		 */
		case "ReturnStatement":
			fn := eslint.GetUpperFunction(parent)

			if fn == nil || !isCallee(fn) {
				return true
			}
			currentNode = eslint.Parent(fn)

		case "ArrowFunctionExpression":
			if !eslint.SameNode(currentNode, eslint.GetNode(parent, "body")) || !isCallee(parent) {
				return true
			}
			currentNode = eslint.Parent(parent)

		/*
		 * e.g.
		 *   var obj = { foo() { ... } };
		 *   class A { foo() { ... } }
		 */
		case "Property", "PropertyDefinition", "MethodDefinition":
			return !eslint.SameNode(eslint.GetNode(parent, "value"), currentNode)

		/*
		 * e.g.
		 *   obj.foo = function foo() { ... };
		 *   Foo = function() { ... };
		 */
		case "AssignmentExpression", "AssignmentPattern":
			if eslint.NodeType(eslint.GetNode(parent, "left")) == "MemberExpression" {
				return false
			}
			if isAnonymous &&
				eslint.NodeType(eslint.GetNode(parent, "left")) == "Identifier" &&
				startsWithUpperCase(eslint.GetString(eslint.GetNode(parent, "left"), "name")) {
				return false
			}
			return true

		/*
		 * e.g.
		 *   var Foo = function() { ... };
		 */
		case "VariableDeclarator":
			return !(isAnonymous &&
				eslint.SameNode(eslint.GetNode(parent, "init"), currentNode) &&
				eslint.NodeType(eslint.GetNode(parent, "id")) == "Identifier" &&
				startsWithUpperCase(eslint.GetString(eslint.GetNode(parent, "id"), "name")))

		/*
		 * e.g.
		 *   var foo = function foo() { ... }.bind(obj);
		 *   (function foo() { ... }).call(obj);
		 *   (function foo() { ... }).apply(obj, []);
		 */
		case "MemberExpression":
			if eslint.SameNode(eslint.GetNode(parent, "object"), currentNode) &&
				isSpecificMemberAccess(parent, nil, bindOrCallOrApplyPattern) {
				maybeCalleeNode := parent
				if eslint.NodeType(eslint.Parent(parent)) == "ChainExpression" {
					maybeCalleeNode = eslint.Parent(parent)
				}

				args := eslint.GetNodes(eslint.Parent(maybeCalleeNode), "arguments")
				return !(isCallee(maybeCalleeNode) && len(args) >= 1 && !isNullOrUndefined(args[0]))
			}
			return true

		/*
		 * e.g.
		 *   Reflect.apply(function() {}, obj, []);
		 *   Array.from([], function() {}, obj);
		 *   list.forEach(function() {}, obj);
		 */
		case "CallExpression":
			callee := eslint.GetNode(parent, "callee")
			args := eslint.GetNodes(parent, "arguments")

			if isReflectApply(callee) {
				return len(args) != 3 || !eslint.SameNode(args[0], currentNode) || isNullOrUndefined(args[1])
			}
			if isArrayFromMethod(callee) {
				return len(args) != 3 || !eslint.SameNode(args[1], currentNode) || isNullOrUndefined(args[2])
			}
			if isMethodWhichHasThisArg(callee) {
				return len(args) != 2 || !eslint.SameNode(args[0], currentNode) || isNullOrUndefined(args[1])
			}
			return true

		// Otherwise `this` is default.
		default:
			return true
		}
	}

	return true
}

// isES5Constructor mirrors ast-utils' isES5Constructor.
func isES5Constructor(node eslint.Node) bool {
	id := eslint.GetNode(node, "id")
	return id != nil && startsWithUpperCase(eslint.GetString(id, "name"))
}

// startsWithUpperCase mirrors ast-utils' startsWithUpperCase
// (s[0] !== s[0].toLocaleLowerCase()).
func startsWithUpperCase(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r != unicode.ToLower(r)
}

// isNullOrUndefined mirrors ast-utils' isNullOrUndefined.
func isNullOrUndefined(node eslint.Node) bool {
	return eslint.IsNullLiteral(node) ||
		(eslint.NodeType(node) == "Identifier" && eslint.GetString(node, "name") == "undefined") ||
		(eslint.NodeType(node) == "UnaryExpression" && eslint.GetString(node, "operator") == "void")
}

// isReflectApply mirrors ast-utils' isReflectApply.
func isReflectApply(node eslint.Node) bool {
	return isSpecificMemberAccess(node, text("Reflect"), text("apply"))
}

// isArrayFromMethod mirrors ast-utils' isArrayFromMethod.
func isArrayFromMethod(node eslint.Node) bool {
	return isSpecificMemberAccess(node, arrayOrTypedArrayPattern, text("from"))
}

// isMethodWhichHasThisArg mirrors ast-utils' isMethodWhichHasThisArg.
func isMethodWhichHasThisArg(node eslint.Node) bool {
	return isSpecificMemberAccess(node, nil, arrayMethodWithThisArgPattern)
}

// hasJSDocThisTag mirrors ast-utils' hasJSDocThisTag.
func hasJSDocThisTag(node eslint.Node, sc *eslint.SourceCode) bool {
	jsdocComment := getJSDocComment(node, sc)

	if jsdocComment != nil && thisTagPattern.MatchString(eslint.GetString(jsdocComment, "value")) {
		return true
	}

	// Checks `@this` in its leading comments for callbacks, because callbacks
	// don't have their own JSDoc comment.
	for _, comment := range getCommentsBefore(node, sc) {
		if thisTagPattern.MatchString(eslint.GetString(comment, "value")) {
			return true
		}
	}
	return false
}

// inClassFieldInitializerValue reports whether a node sits inside a class
// field initializer's `value` with no intervening non-arrow function — i.e.
// whether the original's `PropertyDefinition > *.value` this-scope is the one
// in effect for it.
//
// Arrow functions are transparent (they do not push a `this` scope), so the
// walk steps through them; a FunctionDeclaration/FunctionExpression/StaticBlock
// or the Program pushes its own scope and ends the walk. A computed key is not
// the value, so `class A { [this.eval("x")] = 1 }` keeps the enclosing scope.
func inClassFieldInitializerValue(node eslint.Node) bool {
	child := node
	for parent := eslint.Parent(child); parent != nil; parent = eslint.Parent(parent) {
		switch eslint.NodeType(parent) {
		case "ArrowFunctionExpression":
			// Transparent: `this` is inherited from the enclosing scope.
		case "FunctionDeclaration", "FunctionExpression", "StaticBlock", "Program":
			return false
		case "PropertyDefinition":
			return eslint.SameNode(eslint.GetNode(parent, "value"), child)
		}
		child = parent
	}
	return false
}

// getCommentsBefore mirrors SourceCode#getCommentsBefore: the run of comment
// tokens immediately before the node (stopping at the first non-comment token).
func getCommentsBefore(node eslint.Node, sc *eslint.SourceCode) []eslint.Node {
	var out []eslint.Node

	current := sc.GetTokenBefore(node, eslint.TokenOpt{IncludeComments: true})
	for current != nil && eslint.IsCommentToken(current) {
		out = append(out, current)
		current = sc.GetTokenBefore(current, eslint.TokenOpt{IncludeComments: true})
	}

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// looksLikeExport mirrors source-code.js's looksLikeExport.
func looksLikeExport(node eslint.Node) bool {
	switch eslint.NodeType(node) {
	case "ExportDefaultDeclaration", "ExportNamedDeclaration", "ExportAllDeclaration", "ExportSpecifier":
		return true
	}
	return false
}

// getJSDocComment mirrors SourceCode#getJSDocComment for the node types
// isDefaultThisBinding can hand it (functions and static blocks).
func getJSDocComment(node eslint.Node, sc *eslint.SourceCode) eslint.Node {
	findJSDocComment := func(astNode eslint.Node) eslint.Node {
		tokenBefore := sc.GetTokenBefore(astNode, eslint.TokenOpt{IncludeComments: true})

		if tokenBefore != nil &&
			eslint.IsCommentToken(tokenBefore) &&
			eslint.NodeType(tokenBefore) == eslint.CommentBlock &&
			strings.HasPrefix(eslint.GetString(tokenBefore, "value"), "*") {
			startLine, _ := eslint.LocStart(astNode)
			endLine, _ := eslint.LocEnd(tokenBefore)
			if startLine-endLine <= 1 {
				return tokenBefore
			}
		}
		return nil
	}

	parent := eslint.Parent(node)

	switch eslint.NodeType(node) {
	case "ClassDeclaration", "FunctionDeclaration":
		if looksLikeExport(parent) {
			return findJSDocComment(parent)
		}
		return findJSDocComment(node)

	case "ClassExpression":
		return findJSDocComment(eslint.Parent(parent))

	case "ArrowFunctionExpression", "FunctionExpression":
		if eslint.NodeType(parent) != "CallExpression" && eslint.NodeType(parent) != "NewExpression" {
			for parent != nil &&
				len(getCommentsBefore(parent, sc)) == 0 &&
				!strings.Contains(eslint.NodeType(parent), "Function") &&
				eslint.NodeType(parent) != "MethodDefinition" &&
				eslint.NodeType(parent) != "Property" {
				parent = eslint.Parent(parent)
			}

			if parent != nil &&
				eslint.NodeType(parent) != "FunctionDeclaration" &&
				eslint.NodeType(parent) != "Program" {
				return findJSDocComment(parent)
			}
		}

		return findJSDocComment(node)
	}

	return nil
}
