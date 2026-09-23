package noselfcompare

import eslint "github.com/jclyons52/eslint-go"

// Name is the ESLint rule id.
const Name = "no-self-compare"

// Rule is a port of eslint/lib/rules/no-self-compare.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Disallow comparisons where both sides are exactly the same",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-self-compare",
		},
		Schema: []any{},
		Messages: map[string]string{
			"comparingToSelf": "Comparing to itself is potentially pointless.",
		},
	},
	Create: create,
}

// selfCompareOperators is the rule's `operators` Set.
var selfCompareOperators = map[string]bool{
	"===": true, "==": true, "!==": true, "!=": true,
	">": true, "<": true, ">=": true, "<=": true,
}

// hasSameTokens mirrors the rule's hasSameTokens helper: the two nodes have the
// same number of tokens and each pair matches on type and value. Tokens come
// from sourceCode.getTokens, so comments are invisible and the parentheses
// around a parenthesised operand (which live outside the operand's range) are
// not part of the comparison.
func hasSameTokens(sc *eslint.SourceCode, nodeA, nodeB eslint.Node) bool {
	tokensA := sc.GetTokens(nodeA)
	tokensB := sc.GetTokens(nodeB)

	if len(tokensA) != len(tokensB) {
		return false
	}
	for i := range tokensA {
		if eslint.NodeType(tokensA[i]) != eslint.NodeType(tokensB[i]) ||
			eslint.TokenValue(tokensA[i]) != eslint.TokenValue(tokensB[i]) {
			return false
		}
	}
	return true
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	return map[string]func(eslint.Node){
		"BinaryExpression": func(node eslint.Node) {
			if !selfCompareOperators[eslint.GetString(node, "operator")] {
				return
			}
			if hasSameTokens(sc, eslint.GetNode(node, "left"), eslint.GetNode(node, "right")) {
				ctx.Report(eslint.Report{Node: node, MessageID: "comparingToSelf"})
			}
		},
	}
}
