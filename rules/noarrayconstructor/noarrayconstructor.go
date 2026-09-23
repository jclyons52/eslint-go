package noarrayconstructor

import (
	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-array-constructor"

// Rule is a port of eslint/lib/rules/no-array-constructor.js.
//
// The original meta has `hasSuggestions: true` and no `fixable`; every report
// carries a `suggest` array whose fix text is derived from the call's argument
// text (and, at the start of an expression statement where ASI would break, a
// leading `;`). The core LintMessage has no `suggestions` field
// (docs/porting-rules.md pitfall 6), so the suggestion is not emitted — the
// reported node, message, messageId and location are identical. The helper the
// original uses only for that fix text (`getArgumentsText` /
// `isStartOfExpressionStatement` / `needsPrecedingSemicolon`) is therefore not
// ported; see the package test for the affected corpus entries.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow `Array` constructors",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-array-constructor",
		},
		Messages: map[string]string{
			"preferLiteral":            "The array literal notation [] is preferable.",
			"useLiteral":               "Replace with an array literal.",
			"useLiteralAfterSemicolon": "Replace with an array literal, add preceding semicolon.",
		},
		Schema: []any{},
	},
	Create: create,
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	return map[string]func(eslint.Node){
		"CallExpression": check(ctx, sc),
		"NewExpression":  check(ctx, sc),
	}
}

func check(ctx *eslint.Context, sc *eslint.SourceCode) func(eslint.Node) {
	return func(node eslint.Node) {
		callee := eslint.GetNode(node, "callee")
		if eslint.NodeType(callee) != "Identifier" || eslint.GetString(callee, "name") != "Array" {
			return
		}

		// `Array(5)` builds a sparse array of the given length, not a dense
		// one, so a single non-spread argument is left alone.
		args := eslint.GetNodes(node, "arguments")
		if len(args) == 1 && eslint.NodeType(args[0]) != "SpreadElement" {
			return
		}

		// A predefined global has no declarations, so its variable carries no
		// identifiers; a shadowing declaration adds one.
		variable := eslint.GetVariableByName(sc.Scope(node), "Array")
		if variable != nil && len(variable.Identifiers) == 0 {
			argsText := getArgumentsText(sc, node)
			fixText := "[" + argsText + "]"
			messageID := "useLiteral"

			// A missing semicolon is inserted by ASI before `Array()`, but not
			// before an array literal, so the replacement may need one.
			if isStartOfExpressionStatement(node) && needsPrecedingSemicolon(sc, node) {
				fixText = ";[" + argsText + "]"
				messageID = "useLiteralAfterSemicolon"
			}

			ctx.Report(eslint.Report{
				Node:      node,
				MessageID: "preferLiteral",
				Suggest: []eslint.Suggestion{{
					MessageID: messageID,
					Fix: func(f *eslint.Fixer) *eslint.Fix {
						return f.ReplaceText(node, fixText)
					},
				}},
			})
		}
	}
}

// getArgumentsText gets the text between the calling parentheses of a
// CallExpression or NewExpression, or "" when there are none.
func getArgumentsText(sc *eslint.SourceCode, node eslint.Node) string {
	lastToken := sc.GetLastToken(node)
	if !eslint.IsClosingParenToken(lastToken) {
		return ""
	}

	firstToken := eslint.GetNode(node, "callee")
	for {
		firstToken = sc.GetTokenAfter(firstToken)
		if firstToken == nil || eslint.SameNode(firstToken, lastToken) {
			return ""
		}
		if eslint.IsOpeningParenToken(firstToken) {
			break
		}
	}
	return sc.Text()[eslint.End(firstToken):eslint.Start(lastToken)]
}

// isStartOfExpressionStatement is astUtils.isStartOfExpressionStatement: the
// node is the leftmost node of an ExpressionStatement.
func isStartOfExpressionStatement(node eslint.Node) bool {
	start := eslint.Start(node)
	ancestor := node
	for {
		ancestor = eslint.Parent(ancestor)
		if ancestor == nil || eslint.Start(ancestor) != start {
			return false
		}
		if eslint.NodeType(ancestor) == "ExpressionStatement" {
			return true
		}
	}
}

// needsPrecedingSemicolon is astUtils.needsPrecedingSemicolon: whether an
// opening `(`/`[`/backtick at this position needs a leading semicolon.
var needsPrecedingSemicolonStatements = map[string]bool{
	"DoWhileStatement": true, "ForInStatement": true, "ForOfStatement": true,
	"ForStatement": true, "IfStatement": true, "WhileStatement": true, "WithStatement": true,
}

var needsPrecedingSemicolonPunctuators = map[string]bool{
	":": true, ";": true, "{": true, "=>": true, "++": true, "--": true,
}

var needsPrecedingSemicolonDeclarations = map[string]bool{
	"ExportAllDeclaration": true, "ExportNamedDeclaration": true, "ImportDeclaration": true,
}

var needsPrecedingSemicolonKeywordNodes = map[string]string{
	"break": "BreakStatement", "continue": "ContinueStatement",
	"debugger": "DebuggerStatement", "do": "DoWhileStatement", "else": "IfStatement",
	"return": "ReturnStatement", "yield": "YieldExpression",
}

func needsPrecedingSemicolon(sc *eslint.SourceCode, node eslint.Node) bool {
	prevToken := sc.GetTokenBefore(node)
	if prevToken == nil {
		return false
	}
	if eslint.NodeType(prevToken) == "Punctuator" &&
		needsPrecedingSemicolonPunctuators[eslint.TokenValue(prevToken)] {
		return false
	}

	prevNode := sc.GetNodeByRangeIndex(eslint.Start(prevToken))

	if eslint.IsClosingParenToken(prevToken) {
		return !needsPrecedingSemicolonStatements[eslint.NodeType(prevNode)]
	}

	if eslint.IsClosingBraceToken(prevToken) {
		parent := eslint.Parent(prevNode)
		switch eslint.NodeType(prevNode) {
		case "BlockStatement":
			return eslint.NodeType(parent) == "FunctionExpression"
		case "ClassBody":
			return eslint.NodeType(parent) == "ClassExpression"
		case "ObjectExpression":
			return true
		}
		return false
	}

	if eslint.NodeType(prevToken) == "Identifier" || eslint.NodeType(prevToken) == "Keyword" {
		parent := eslint.Parent(prevNode)
		if eslint.NodeType(parent) == "BreakStatement" || eslint.NodeType(parent) == "ContinueStatement" {
			return false
		}
		return eslint.NodeType(prevNode) != needsPrecedingSemicolonKeywordNodes[eslint.TokenValue(prevToken)]
	}

	if eslint.NodeType(prevToken) == "String" {
		return !needsPrecedingSemicolonDeclarations[eslint.NodeType(eslint.Parent(prevNode))]
	}

	return true
}
