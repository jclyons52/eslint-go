package nocaller

import (
	"regexp"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-caller"

// calleeOrCaller mirrors the original's /^calle[er]$/u.
var calleeOrCaller = regexp.MustCompile(`^calle[er]$`)

// Rule is a port of eslint/lib/rules/no-caller.js.
//
// KNOWN PLATFORM GAP: the rule's only trigger is `arguments.callee` /
// `arguments.caller`, and the Go parser (acorn-go) rejects the identifier
// `arguments` outright — see the package test for the exact case and the
// report that goes with this port.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow the use of `arguments.caller` or `arguments.callee`",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-caller",
		},
		Messages: map[string]string{
			"unexpected": "Avoid arguments.{{prop}}.",
		},
		Schema: []any{},
	},
	Create: func(ctx *eslint.Context) map[string]func(eslint.Node) {
		return map[string]func(eslint.Node){
			"MemberExpression": func(node eslint.Node) {
				// `node.object.name` / `node.property.name` are undefined for
				// non-Identifier operands; "" is falsy in the original check, so
				// GetString's zero value is the faithful equivalent.
				objectName := eslint.GetString(eslint.GetNode(node, "object"), "name")
				propertyName := eslint.GetString(eslint.GetNode(node, "property"), "name")

				if objectName == "arguments" && !eslint.GetBool(node, "computed") &&
					propertyName != "" && calleeOrCaller.MatchString(propertyName) {
					ctx.Report(eslint.Report{
						Node:      node,
						MessageID: "unexpected",
						Data:      map[string]any{"prop": propertyName},
					})
				}
			},
		}
	},
}
