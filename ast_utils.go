package eslint

import (
	"strings"

	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// ast_utils.go — the subset of eslint/lib/rules/utils/ast-utils.js that the
// ported rule set needs, exported so rule packages can use it. Anything a rule
// shares with another rule belongs here rather than in a rule package.

// IsNullLiteral reports whether a Literal is the `null` literal (rather than a
// regex or a bigint whose raw text also has no value).
func IsNullLiteral(node Node) bool {
	return NodeType(node) == "Literal" && GetString(node, "raw") == "null"
}

// IsStringLiteral reports whether a node is a string Literal.
func IsStringLiteral(node Node) bool {
	if NodeType(node) != "Literal" {
		return false
	}
	_, ok := Get(node, "value").(string)
	return ok
}

// IsEmptyStringLiteral reports whether a node is the empty string literal.
func IsEmptyStringLiteral(node Node) bool {
	if !IsStringLiteral(node) {
		return false
	}
	s, _ := Get(node, "value").(string)
	return s == ""
}

// GetStaticStringValue returns the static string value of a node, and whether
// it has one — Literal strings, `null`, regex literals (as /p/f) and bigints
// (as their digit string), plus a single-quasi TemplateLiteral.
func GetStaticStringValue(node Node) (string, bool) {
	switch NodeType(node) {
	case "Literal":
		v, hasValue := node["value"]
		if !hasValue || v == nil {
			raw := GetString(node, "raw")
			if raw == "null" {
				return "null", true
			}
			if re := GetMap(node, "regex"); re != nil {
				return "/" + GetString(Node(re), "pattern") + "/" + GetString(Node(re), "flags"), true
			}
			if b, ok := node["bigint"].(string); ok {
				return b, true
			}
			return "", false
		}
		switch t := v.(type) {
		case string:
			return t, true
		case float64:
			return itoa(int(t)), true
		case bool:
			if t {
				return "true", true
			}
			return "false", true
		}
		return "", false
	case "TemplateLiteral":
		if len(GetNodes(node, "expressions")) == 0 && len(GetNodes(node, "quasis")) == 1 {
			quasi := GetNodes(node, "quasis")[0]
			if cooked, ok := GetMap(quasi, "value")["cooked"].(string); ok {
				return cooked, true
			}
		}
	}
	return "", false
}

// GetStaticPropertyName returns the statically known property name of a
// Property/MethodDefinition/PropertyDefinition/MemberExpression, and whether
// the name is static ("" can be a legitimate static name, so the bool matters).
func GetStaticPropertyName(node Node) (string, bool) {
	var prop Node
	switch NodeType(node) {
	case "ChainExpression":
		return GetStaticPropertyName(GetNode(node, "expression"))
	case "Property", "PropertyDefinition", "MethodDefinition":
		prop = GetNode(node, "key")
	case "MemberExpression":
		prop = GetNode(node, "property")
	}
	if prop == nil {
		return "", false
	}
	if NodeType(prop) == "Identifier" && !GetBool(node, "computed") {
		return GetString(prop, "name"), true
	}
	return GetStaticStringValue(prop)
}

// ModuleScope returns the module scope of a Program, mirroring
// astUtils.getModuleScope / the Linter's module scope lookup.
func ModuleScope(initialScope *eslintscope.Scope) *eslintscope.Scope {
	scope := initialScope
	for scope != nil && scope.Type != eslintscope.ScopeModule {
		scope = scope.Upper
	}
	return scope
}

// GetInnermostScope returns the innermost scope containing a node
// (astUtils.getInnermostScope).
func GetInnermostScope(initialScope *eslintscope.Scope, node Node) *eslintscope.Scope {
	scope := initialScope
	location := Start(node)
	for {
		found := false
		for _, child := range scope.ChildScopes {
			if child.Block == nil {
				continue
			}
			if Start(child.Block) <= location && location <= End(child.Block) {
				scope = child
				found = true
				break
			}
		}
		if !found {
			return scope
		}
	}
}

// GetVariableByName finds a variable in a scope by name.
func GetVariableByName(initialScope *eslintscope.Scope, name string) *eslintscope.Variable {
	scope := initialScope
	for scope != nil {
		for _, v := range scope.Variables {
			if v.Name == name {
				return v
			}
		}
		scope = scope.Upper
	}
	return nil
}

// IsParenthesized reports whether a node is wrapped in `times` extra parens,
// mirroring astUtils.isParenthesized (which walks outward from the node,
// skipping the enclosing parens of the expression's own grammar).
func IsParenthesized(times int, node Node, sc *SourceCode) bool {
	count := 0
	previous := node
	interestingNode := node
	for {
		if !SameNode(previous, interestingNode) {
			openingParen := sc.GetTokenBefore(previous)
			if !IsOpeningParenToken(openingParen) {
				break
			}
			closingParen := sc.GetTokenAfter(previous)
			if !IsClosingParenToken(closingParen) {
				break
			}
			if End(openingParen) == End(previous) || Start(closingParen) == Start(previous) {
				break
			}
			count++
			previous = openingParen
			continue
		}
		parent := Parent(previous)
		if parent == nil || NodeType(parent) == "ExpressionStatement" {
			break
		}
		interestingNode = parent
	}
	return count >= times
}

// IsFunction reports whether a node is any function form.
func IsFunction(node Node) bool {
	switch NodeType(node) {
	case "FunctionDeclaration", "FunctionExpression", "ArrowFunctionExpression":
		return true
	}
	return false
}

// IsArrowFunction reports whether a node is an arrow function.
func IsArrowFunction(node Node) bool { return NodeType(node) == "ArrowFunctionExpression" }

// IsLoop reports whether a node is a loop statement.
func IsLoop(node Node) bool {
	switch NodeType(node) {
	case "ForStatement", "ForInStatement", "ForOfStatement", "WhileStatement", "DoWhileStatement":
		return true
	}
	return false
}

// IsLexicalDeclaration reports whether a node is let/const/class/import.
func IsLexicalDeclaration(node Node) bool {
	switch NodeType(node) {
	case "VariableDeclaration":
		return GetString(node, "kind") != "var"
	case "ClassDeclaration", "ImportDeclaration":
		return true
	}
	return false
}

// SkipChainExpression unwraps an optional-chain wrapper so callers can inspect
// the underlying call/member expression.
func SkipChainExpression(node Node) Node {
	if NodeType(node) == "ChainExpression" {
		return GetNode(node, "expression")
	}
	return node
}

// GetUpperFunction returns the closest enclosing function-ish node, or nil when
// the node is at the top level (astUtils.getUpperFunction).
func GetUpperFunction(node Node) Node {
	if IsFunction(node) {
		return node
	}
	for p := Parent(node); p != nil; p = Parent(p) {
		if IsFunction(p) {
			return p
		}
		switch NodeType(p) {
		case "StaticBlock", "BlockStatement", "SwitchStatement", "ForStatement",
			"ForInStatement", "ForOfStatement", "WhileStatement", "DoWhileStatement",
			"WithStatement", "TryStatement", "CatchClause", "IfStatement", "LabeledStatement":
			continue
		case "Program":
			return nil
		}
	}
	return nil
}

// IsDecimalInteger reports whether a node is an integer literal token or node
// written in base 10 (astUtils.isDecimalInteger).
func IsDecimalInteger(node Node) bool {
	if GetBool(node, "bigint") || len(GetString(node, "bigint")) > 0 {
		return false
	}
	switch NodeType(node) {
	case "Literal":
		raw := GetString(node, "raw")
		if raw == "" {
			return false
		}
		if _, isNum := node["value"].(float64); !isNum {
			return false
		}
		return raw == itoa(int(GetFloat(node, "value")))
	case TokenNumeric:
		return IsDecimalIntegerNumericToken(node)
	}
	return false
}

// IsDecimalIntegerNumericToken reports whether a token is a base-10 integer.
func IsDecimalIntegerNumericToken(node Node) bool {
	if NodeType(node) != TokenNumeric {
		return false
	}
	raw := TokenValue(node)
	if len(raw) > 1 && raw[0] == '0' && raw[1] != '.' && raw[1] != 'e' && raw[1] != 'E' &&
		raw[1] != 'x' && raw[1] != 'X' && raw[1] != 'o' && raw[1] != 'O' && raw[1] != 'b' && raw[1] != 'B' {
		return false
	}
	return !strings.ContainsAny(raw, "_.")
}

// IsDirectiveComment reports whether a comment is an ESLint directive
// (eslint-disable / eslint-enable / eslint / globals / exported).
func IsDirectiveComment(node Node) bool {
	if !IsCommentToken(node) {
		return false
	}
	comment := strings.TrimSpace(GetString(node, "value"))
	switch {
	case strings.HasPrefix(comment, "eslint-disable"), strings.HasPrefix(comment, "eslint-enable"),
		strings.HasPrefix(comment, "eslint "), strings.HasPrefix(comment, "global "),
		strings.HasPrefix(comment, "globals "), strings.HasPrefix(comment, "exported "),
		strings.HasPrefix(comment, "eslint-env "), strings.HasPrefix(comment, "eslint-disable-line"),
		strings.HasPrefix(comment, "eslint-disable-next-line"):
		return true
	}
	return false
}

// GetUpperFunctionOrProgram returns the closest enclosing function, or the
// Program when the node is not inside one.
func GetUpperFunctionOrProgram(node Node) Node {
	for p := node; p != nil; p = Parent(p) {
		switch NodeType(p) {
		case "FunctionDeclaration", "FunctionExpression", "ArrowFunctionExpression", "Program":
			return p
		}
	}
	return nil
}
