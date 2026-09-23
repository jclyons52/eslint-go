package noextrasemi

import (
	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-extra-semi"

// Rule is a port of eslint/lib/rules/no-extra-semi.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type:       "suggestion",
		Deprecated: true,
		Docs: eslint.RuleDocs{
			Description: "Disallow unnecessary semicolons",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-extra-semi",
		},
		Fixable: "code",
		Schema:  []any{},
		Messages: map[string]string{
			"unexpected": "Unnecessary semicolon.",
		},
	},
	Create: create,
}

// isTopLevelExpressionStatement is astUtils.isTopLevelExpressionStatement: an
// expression statement at the top level of a program or of a function body
// (where removing a preceding token could turn it into a directive).
func isTopLevelExpressionStatement(node eslint.Node) bool {
	if eslint.NodeType(node) != "ExpressionStatement" {
		return false
	}
	parent := eslint.Parent(node)
	parentType := eslint.NodeType(parent)

	return parentType == "Program" ||
		(parentType == "BlockStatement" && eslint.IsFunction(eslint.Parent(parent)))
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode

	/*
	 * isFixable: a node or token is fixable if it can be removed without
	 * turning a subsequent statement into a directive after fixing other nodes.
	 */
	isFixable := func(nodeOrToken eslint.Node) bool {
		nextToken := sc.GetTokenAfter(nodeOrToken)

		if nextToken == nil || eslint.NodeType(nextToken) != eslint.TokenString {
			return true
		}
		stringNode := sc.GetNodeByRangeIndex(eslint.Start(nextToken))

		return !isTopLevelExpressionStatement(eslint.Parent(stringNode))
	}

	report := func(nodeOrToken eslint.Node) {
		var fix func(*eslint.Fixer) *eslint.Fix
		if isFixable(nodeOrToken) {
			target := nodeOrToken
			fix = func(f *eslint.Fixer) *eslint.Fix {
				/*
				 * Expand the replacement range to include the surrounding
				 * tokens to avoid conflicting with semi.
				 * https://github.com/eslint/eslint/issues/7928
				 */
				return eslint.NewFixTracker(f, sc).RetainSurroundingTokens(target).Remove(target)
			}
		}
		ctx.Report(eslint.Report{
			Node:      nodeOrToken,
			MessageID: "unexpected",
			Fix:       fix,
		})
	}

	/*
	 * checkForPartOfClassBody: checks tokens from a specified token to a next
	 * MethodDefinition or the end of the class body.
	 */
	checkForPartOfClassBody := func(firstToken eslint.Node) {
		for token := firstToken; token != nil &&
			eslint.NodeType(token) == eslint.TokenPunctuator &&
			!eslint.IsClosingBraceToken(token); token = sc.GetTokenAfter(token) {
			if eslint.IsSemicolonToken(token) {
				report(token)
			}
		}
	}

	// The source rule registers these three types as one comma selector
	// ("MethodDefinition, PropertyDefinition, StaticBlock"); a node has exactly
	// one type, so three entries are equivalent.
	checkAfterNode := func(node eslint.Node) {
		checkForPartOfClassBody(sc.GetTokenAfter(node))
	}

	return map[string]func(eslint.Node){
		// Reports an empty statement, except if the parent node is a loop.
		"EmptyStatement": func(node eslint.Node) {
			switch eslint.NodeType(eslint.Parent(node)) {
			case "ForStatement", "ForInStatement", "ForOfStatement",
				"WhileStatement", "DoWhileStatement", "IfStatement",
				"LabeledStatement", "WithStatement":
				// allowed
			default:
				report(node)
			}
		},
		"ClassBody": func(node eslint.Node) {
			checkForPartOfClassBody(sc.GetFirstToken(node, eslint.TokenOpt{Skip: 1})) // 0 is `{`.
		},
		"MethodDefinition":   checkAfterNode,
		"PropertyDefinition": checkAfterNode,
		"StaticBlock":        checkAfterNode,
	}
}
