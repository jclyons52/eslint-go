package nodebugger

import eslint "github.com/jclyons52/eslint-go"

// Name is the ESLint rule id.
const Name = "no-debugger"

// Rule is a port of eslint/lib/rules/no-debugger.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Disallow the use of `debugger`",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-debugger",
		},
		Messages: map[string]string{
			"unexpected": "Unexpected 'debugger' statement.",
		},
	},
	Create: func(ctx *eslint.Context) map[string]func(eslint.Node) {
		return map[string]func(eslint.Node){
			"DebuggerStatement": func(node eslint.Node) {
				ctx.Report(eslint.Report{Node: node, MessageID: "unexpected"})
			},
		}
	},
}
