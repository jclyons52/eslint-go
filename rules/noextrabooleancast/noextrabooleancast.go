package noextrabooleancast

import (
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-extra-boolean-cast"

// Rule is a port of eslint/lib/rules/no-extra-boolean-cast.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow unnecessary boolean casts",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-extra-boolean-cast",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enforceForLogicalOperands": map[string]any{
						"type":    "boolean",
						"default": false,
					},
				},
				"additionalProperties": false,
			},
		},
		Fixable: "code",
		Messages: map[string]string{
			"unexpectedCall":     "Redundant Boolean call.",
			"unexpectedNegation": "Redundant double negation.",
		},
	},
	Create: create,
}

// booleanNodeTypes is BOOLEAN_NODE_TYPES: node types which have a test that
// coerces values to booleans.
var booleanNodeTypes = map[string]bool{
	"IfStatement":           true,
	"DoWhileStatement":      true,
	"WhileStatement":        true,
	"ConditionalExpression": true,
	"ForStatement":          true,
}

// precedence is astUtils.getPrecedence.
func precedence(node eslint.Node) int {
	switch eslint.NodeType(node) {
	case "SequenceExpression":
		return 0

	case "AssignmentExpression", "ArrowFunctionExpression", "YieldExpression":
		return 1

	case "ConditionalExpression":
		return 3

	case "LogicalExpression":
		switch eslint.GetString(node, "operator") {
		case "||", "??":
			return 4
		case "&&":
			return 5
		}
		fallthrough

	case "BinaryExpression":
		switch eslint.GetString(node, "operator") {
		case "|":
			return 6
		case "^":
			return 7
		case "&":
			return 8
		case "==", "!=", "===", "!==":
			return 9
		case "<", "<=", ">", ">=", "in", "instanceof":
			return 10
		case "<<", ">>", ">>>":
			return 11
		case "+", "-":
			return 12
		case "*", "/", "%":
			return 13
		case "**":
			return 15
		}
		fallthrough

	case "UnaryExpression", "AwaitExpression":
		return 16

	case "UpdateExpression":
		return 17

	case "CallExpression", "ChainExpression", "ImportExpression":
		return 18

	case "NewExpression":
		return 19
	}

	if _, ok := eslint.VisitorKeys[eslint.NodeType(node)]; ok {
		return 20
	}
	// Unknown node: assume the lowest precedence.
	return -1
}

// isLogicalExpression reports whether a node is `&&` or `||`.
func isLogicalExpression(node eslint.Node) bool {
	return eslint.NodeType(node) == "LogicalExpression" &&
		(eslint.GetString(node, "operator") == "&&" || eslint.GetString(node, "operator") == "||")
}

// isCoalesceExpression reports whether a node is `??`.
func isCoalesceExpression(node eslint.Node) bool {
	return eslint.NodeType(node) == "LogicalExpression" && eslint.GetString(node, "operator") == "??"
}

// isMixedLogicalAndCoalesceExpressions is astUtils.isMixedLogicalAndCoalesceExpressions.
func isMixedLogicalAndCoalesceExpressions(left, right eslint.Node) bool {
	return (isLogicalExpression(left) && isCoalesceExpression(right)) ||
		(isCoalesceExpression(left) && isLogicalExpression(right))
}

// parentSyntaxParen is eslint-utils' private getParentSyntaxParen: the
// parenthesis token a parent construct contributes around the node.
func parentSyntaxParen(node eslint.Node, sc *eslint.SourceCode) eslint.Node {
	parent := eslint.Parent(node)
	if parent == nil {
		return nil
	}

	switch eslint.NodeType(parent) {
	case "CallExpression", "NewExpression":
		args := eslint.GetNodes(parent, "arguments")
		if len(args) == 1 && eslint.SameNode(args[0], node) {
			return sc.GetTokenAfter(eslint.GetNode(parent, "callee"), eslint.TokenOpt{Filter: eslint.IsOpeningParenToken})
		}
		return nil

	case "DoWhileStatement":
		if eslint.SameNode(eslint.GetNode(parent, "test"), node) {
			return sc.GetTokenAfter(eslint.GetNode(parent, "body"), eslint.TokenOpt{Filter: eslint.IsOpeningParenToken})
		}
		return nil

	case "IfStatement", "WhileStatement":
		if eslint.SameNode(eslint.GetNode(parent, "test"), node) {
			return sc.GetFirstToken(parent, eslint.TokenOpt{Skip: 1})
		}
		return nil

	case "ImportExpression":
		if eslint.SameNode(eslint.GetNode(parent, "source"), node) {
			return sc.GetFirstToken(parent, eslint.TokenOpt{Skip: 1})
		}
		return nil

	case "SwitchStatement":
		if eslint.SameNode(eslint.GetNode(parent, "discriminant"), node) {
			return sc.GetFirstToken(parent, eslint.TokenOpt{Skip: 1})
		}
		return nil

	case "WithStatement":
		if eslint.SameNode(eslint.GetNode(parent, "object"), node) {
			return sc.GetFirstToken(parent, eslint.TokenOpt{Skip: 1})
		}
		return nil
	}
	return nil
}

// isParenthesized is eslint-utils' isParenthesized(1, node, sourceCode) — the
// eslint-utils variant, which is not astUtils.isParenthesised.
func isParenthesized(node eslint.Node, sc *eslint.SourceCode) bool {
	if node == nil || eslint.Parent(node) == nil {
		return false
	}
	parent := eslint.Parent(node)
	if eslint.NodeType(parent) == "CatchClause" && eslint.SameNode(eslint.GetNode(parent, "param"), node) {
		return false
	}

	times := 1
	maybeLeftParen := node
	maybeRightParen := node
	syntaxParen := parentSyntaxParen(node, sc)

	for {
		maybeLeftParen = sc.GetTokenBefore(maybeLeftParen)
		maybeRightParen = sc.GetTokenAfter(maybeRightParen)

		if maybeLeftParen == nil || maybeRightParen == nil {
			break
		}
		if !eslint.IsOpeningParenToken(maybeLeftParen) || !eslint.IsClosingParenToken(maybeRightParen) {
			break
		}
		// Avoid false positive such as `if (a) {}`.
		if syntaxParen != nil && eslint.SameNode(maybeLeftParen, syntaxParen) {
			break
		}
		times--
		if times <= 0 {
			break
		}
	}
	return times == 0
}

// canTokensBeAdjacent mirrors astUtils.canTokensBeAdjacent for two real tokens.
func canTokensBeAdjacent(left, right eslint.Node) bool {
	if eslint.NodeType(left) == "Shebang" || eslint.NodeType(left) == "Hashbang" {
		return false
	}
	if right == nil {
		return false
	}

	leftPunct := eslint.NodeType(left) == eslint.TokenPunctuator
	rightPunct := eslint.NodeType(right) == eslint.TokenPunctuator

	if leftPunct || rightPunct {
		if leftPunct && rightPunct {
			leftValue, rightValue := eslint.TokenValue(left), eslint.TokenValue(right)
			isPlus := func(v string) bool { return v == "+" || v == "++" }
			isMinus := func(v string) bool { return v == "-" || v == "--" }
			return !((isPlus(leftValue) && isPlus(rightValue)) ||
				(isMinus(leftValue) && isMinus(rightValue)))
		}
		if leftPunct && eslint.TokenValue(left) == "/" {
			switch eslint.NodeType(right) {
			case "Block", "Line", "RegularExpression":
				return false
			}
			return true
		}
		return true
	}

	if eslint.NodeType(left) == "String" || eslint.NodeType(right) == "String" ||
		eslint.NodeType(left) == "Template" || eslint.NodeType(right) == "Template" {
		return true
	}

	if eslint.NodeType(left) != "Numeric" && eslint.NodeType(right) == "Numeric" &&
		strings.HasPrefix(eslint.TokenValue(right), ".") {
		return true
	}

	if eslint.NodeType(left) == "Block" || eslint.NodeType(right) == "Block" || eslint.NodeType(right) == "Line" {
		return true
	}

	if eslint.NodeType(right) == "PrivateIdentifier" {
		return true
	}

	return false
}

// firstTokenOf produces the first token espree.tokenize would produce for a
// short source string. astUtils.canTokensBeAdjacent tokenizes string operands
// with espree; this rule only ever passes "true" (the !Boolean() replacement),
// so the literal/identifier/punctuator cases below are all that is reachable.
func firstTokenOf(text string) eslint.Node {
	if text == "" {
		return nil
	}
	switch text[0] {
	case 't', 'f':
		if text == "true" || text == "false" {
			return eslint.Node{"type": eslint.TokenBoolean, "value": text}
		}
		return eslint.Node{"type": eslint.TokenIdentifier, "value": text}
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
		return eslint.Node{"type": eslint.TokenNumeric, "value": text}
	case '"', '\'':
		return eslint.Node{"type": eslint.TokenString, "value": text}
	case '`':
		return eslint.Node{"type": eslint.TokenTemplate, "value": text}
	}
	for _, p := range []string{">>>=", "...", "===", "!==", "**=", "<<=", ">>=", ">>>", "&&=", "||=", "??=",
		"=>", "==", "!=", "<=", ">=", "&&", "||", "??", "?.", "++", "--", "**", "<<", ">>", "+=", "-=", "*=",
		"/=", "%=", "&=", "|=", "^=", "{", "}", "(", ")", "[", "]", ".", ";", ",", "<", ">", "+", "-", "*",
		"/", "%", "&", "|", "^", "!", "~", "?", ":", "="} {
		if strings.HasPrefix(text, p) {
			return eslint.Node{"type": eslint.TokenPunctuator, "value": p}
		}
	}
	return eslint.Node{"type": eslint.TokenIdentifier, "value": text}
}

// canTokensBeAdjacentString mirrors astUtils.canTokensBeAdjacent when the right
// operand is a source string (which espree tokenizes first).
func canTokensBeAdjacentString(left eslint.Node, rightValue string) bool {
	return canTokensBeAdjacent(left, firstTokenOf(rightValue))
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode

	// isBooleanFunctionOrConstructorCall: Boolean(<bool>) and new Boolean(<bool>).
	isBooleanFunctionOrConstructorCall := func(node eslint.Node) bool {
		return (eslint.NodeType(node) == "CallExpression" || eslint.NodeType(node) == "NewExpression") &&
			eslint.NodeType(eslint.GetNode(node, "callee")) == "Identifier" &&
			eslint.GetString(eslint.GetNode(node, "callee"), "name") == "Boolean"
	}

	// isLogicalContext: a logical expression with the option enabled.
	isLogicalContext := func(node eslint.Node) bool {
		if eslint.NodeType(node) != "LogicalExpression" {
			return false
		}
		op := eslint.GetString(node, "operator")
		if op != "||" && op != "&&" {
			return false
		}
		enforce, _ := ctx.OptionMapValue(0, "enforceForLogicalOperands", false).(bool)
		return enforce
	}

	// isInBooleanContext: is the node's value coerced to a boolean at runtime?
	isInBooleanContext := func(node eslint.Node) bool {
		parent := eslint.Parent(node)
		if parent == nil {
			return false
		}
		if isBooleanFunctionOrConstructorCall(parent) {
			args := eslint.GetNodes(parent, "arguments")
			if len(args) > 0 && eslint.SameNode(node, args[0]) {
				return true
			}
		}
		if booleanNodeTypes[eslint.NodeType(parent)] && eslint.SameNode(node, eslint.GetNode(parent, "test")) {
			return true
		}
		// !<bool>
		if eslint.NodeType(parent) == "UnaryExpression" && eslint.GetString(parent, "operator") == "!" {
			return true
		}
		return false
	}

	// isInFlaggedContext: a context that should report, recursing through
	// logical operands when the option is on.
	var isInFlaggedContext func(node eslint.Node) bool
	isInFlaggedContext = func(node eslint.Node) bool {
		if eslint.NodeType(eslint.Parent(node)) == "ChainExpression" {
			return isInFlaggedContext(eslint.Parent(node))
		}
		if isInBooleanContext(node) {
			return true
		}
		parent := eslint.Parent(node)
		return parent != nil && isLogicalContext(parent) && isInFlaggedContext(parent)
	}

	// hasCommentsInside: does the node contain any comment?
	hasCommentsInside := func(node eslint.Node) bool {
		return len(sc.GetCommentsInside(node)) > 0
	}

	// needsParens: does `node` need parentheses when it replaces `previousNode`?
	var needsParens func(previousNode, node eslint.Node) bool
	needsParens = func(previousNode, node eslint.Node) bool {
		if eslint.NodeType(eslint.Parent(previousNode)) == "ChainExpression" {
			return needsParens(eslint.Parent(previousNode), node)
		}
		if isParenthesized(previousNode, sc) {
			// The parentheses around the previous node stay, so there is no
			// need for an additional pair.
			return false
		}

		parent := eslint.Parent(previousNode)
		switch eslint.NodeType(parent) {
		case "CallExpression", "NewExpression":
			return eslint.NodeType(node) == "SequenceExpression"
		case "IfStatement", "DoWhileStatement", "WhileStatement", "ForStatement":
			return false
		case "ConditionalExpression":
			return precedence(node) <= precedence(parent)
		case "UnaryExpression":
			return precedence(node) < precedence(parent)
		case "LogicalExpression":
			if isMixedLogicalAndCoalesceExpressions(node, parent) {
				return true
			}
			if eslint.SameNode(previousNode, eslint.GetNode(parent, "left")) {
				return precedence(node) < precedence(parent)
			}
			return precedence(node) <= precedence(parent)
		}
		return false
	}

	return map[string]func(eslint.Node){
		"UnaryExpression": func(node eslint.Node) {
			parent := eslint.Parent(node)

			// Exit early if it's guaranteed not to match.
			if eslint.GetString(node, "operator") != "!" ||
				eslint.NodeType(parent) != "UnaryExpression" ||
				eslint.GetString(parent, "operator") != "!" {
				return
			}

			if !isInFlaggedContext(parent) {
				return
			}

			ctx.Report(eslint.Report{
				Node:      parent,
				MessageID: "unexpectedNegation",
				Fix: func(f *eslint.Fixer) *eslint.Fix {
					if hasCommentsInside(parent) {
						return nil
					}

					argument := eslint.GetNode(node, "argument")
					if needsParens(parent, argument) {
						return f.ReplaceText(parent, "("+sc.GetText(argument)+")")
					}

					prefix := ""
					tokenBefore := sc.GetTokenBefore(parent)
					firstReplacementToken := sc.GetFirstToken(argument)

					if tokenBefore != nil &&
						eslint.End(tokenBefore) == eslint.Start(parent) &&
						!canTokensBeAdjacent(tokenBefore, firstReplacementToken) {
						prefix = " "
					}

					return f.ReplaceText(parent, prefix+sc.GetText(argument))
				},
			})
		},

		"CallExpression": func(node eslint.Node) {
			callee := eslint.GetNode(node, "callee")
			if eslint.NodeType(callee) != "Identifier" || eslint.GetString(callee, "name") != "Boolean" {
				return
			}

			if !isInFlaggedContext(node) {
				return
			}

			ctx.Report(eslint.Report{
				Node:      node,
				MessageID: "unexpectedCall",
				Fix: func(f *eslint.Fixer) *eslint.Fix {
					parent := eslint.Parent(node)
					args := eslint.GetNodes(node, "arguments")

					if len(args) == 0 {
						if eslint.NodeType(parent) == "UnaryExpression" && eslint.GetString(parent, "operator") == "!" {

							/*
							 * !Boolean() -> true
							 */

							if hasCommentsInside(parent) {
								return nil
							}

							replacement := "true"
							prefix := ""
							tokenBefore := sc.GetTokenBefore(parent)

							if tokenBefore != nil &&
								eslint.End(tokenBefore) == eslint.Start(parent) &&
								!canTokensBeAdjacentString(tokenBefore, replacement) {
								prefix = " "
							}

							return f.ReplaceText(parent, prefix+replacement)
						}

						/*
						 * Boolean() -> false
						 */

						if hasCommentsInside(node) {
							return nil
						}

						return f.ReplaceText(node, "false")
					}

					if len(args) == 1 {
						argument := args[0]

						if eslint.NodeType(argument) == "SpreadElement" || hasCommentsInside(node) {
							return nil
						}

						/*
						 * Boolean(expression) -> expression
						 */

						if needsParens(node, argument) {
							return f.ReplaceText(node, "("+sc.GetText(argument)+")")
						}

						return f.ReplaceText(node, sc.GetText(argument))
					}

					// two or more arguments
					return nil
				},
			})
		},
	}
}
