package noduplicatecase

import eslint "github.com/jclyons52/eslint-go"

// Name is the ESLint rule id.
const Name = "no-duplicate-case"

// Rule is a port of eslint/lib/rules/no-duplicate-case.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Disallow duplicate case labels",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-duplicate-case",
		},
		Schema: []any{},
		Messages: map[string]string{
			"unexpected": "Duplicate case label.",
		},
	},
	Create: create,
}

// equalTokens mirrors astUtils.equalTokens: same token count and every pair
// matching on type and value. Comments are not tokens, so they do not make two
// case labels differ; parentheses around the test are outside its range, so
// `case b:` and `case (b):` compare equal.
func equalTokens(sc *eslint.SourceCode, left, right eslint.Node) bool {
	tokensL := sc.GetTokens(left)
	tokensR := sc.GetTokens(right)

	if len(tokensL) != len(tokensR) {
		return false
	}
	for i := range tokensL {
		if eslint.NodeType(tokensL[i]) != eslint.NodeType(tokensR[i]) ||
			eslint.TokenValue(tokensL[i]) != eslint.TokenValue(tokensR[i]) {
			return false
		}
	}
	return true
}

// equal mirrors the rule's equal helper: the node types must match first, then
// the tokens.
func equal(sc *eslint.SourceCode, a, b eslint.Node) bool {
	if eslint.NodeType(a) != eslint.NodeType(b) {
		return false
	}
	return equalTokens(sc, a, b)
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	return map[string]func(eslint.Node){
		"SwitchStatement": func(node eslint.Node) {
			var previousTests []eslint.Node

			for _, switchCase := range eslint.GetNodes(node, "cases") {
				test := eslint.GetNode(switchCase, "test")
				if test == nil {
					continue
				}
				duplicate := false
				for _, previousTest := range previousTests {
					if equal(sc, previousTest, test) {
						duplicate = true
						break
					}
				}
				if duplicate {
					ctx.Report(eslint.Report{Node: switchCase, MessageID: "unexpected"})
				} else {
					previousTests = append(previousTests, test)
				}
			}
		},
	}
}
