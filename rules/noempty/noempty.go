package noempty

import (
	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-empty"

// Rule is a port of eslint/lib/rules/no-empty.js.
//
// The original meta has `hasSuggestions: true` and attaches a `suggest` array
// to every BlockStatement report. The core LintMessage has no `suggestions`
// field (docs/porting-rules.md pitfall 6), so the suggestion is not emitted;
// everything else — the guards, the reported node, the message and its
// location — is reproduced exactly. See the package test for the affected
// corpus entries.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow empty block statements",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-empty",
		},
		Messages: map[string]string{
			"unexpected":     "Empty {{type}} statement.",
			"suggestComment": "Add comment inside empty {{type}} statement.",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"allowEmptyCatch": map[string]any{
						"type":    "boolean",
						"default": false,
					},
				},
				"additionalProperties": false,
			},
		},
	},
	Create: create,
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	allowEmptyCatch := false
	if opts := ctx.OptionMap(0); opts != nil {
		if b, ok := opts["allowEmptyCatch"].(bool); ok {
			allowEmptyCatch = b
		}
	}

	sc := ctx.SourceCode

	return map[string]func(eslint.Node){
		"BlockStatement": func(node eslint.Node) {
			// If the body is not empty, we can just return immediately.
			if len(eslint.GetNodes(node, "body")) != 0 {
				return
			}

			// A function is generally allowed to be empty.
			if eslint.IsFunction(eslint.Parent(node)) {
				return
			}

			if allowEmptyCatch && eslint.NodeType(eslint.Parent(node)) == "CatchClause" {
				return
			}

			// Any other block is only allowed to be empty if it contains a comment.
			if len(sc.GetCommentsInside(node)) > 0 {
				return
			}

			ctx.Report(eslint.Report{
				Node:      node,
				MessageID: "unexpected",
				Data:      map[string]any{"type": "block"},
			})
		},

		"SwitchStatement": func(node eslint.Node) {
			// `typeof node.cases === "undefined"` never holds for a parsed
			// SwitchStatement, so only the length check can fire.
			if len(eslint.GetNodes(node, "cases")) == 0 {
				ctx.Report(eslint.Report{
					Node:      node,
					MessageID: "unexpected",
					Data:      map[string]any{"type": "switch"},
				})
			}
		},
	}
}
