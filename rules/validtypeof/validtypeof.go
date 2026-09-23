package validtypeof

import (
	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "valid-typeof"

// Rule is a port of eslint/lib/rules/valid-typeof.js.
//
// NOTE (core gap): the original sets `meta.hasSuggestions: true` and reports a
// `suggest` array on the `typeof x === undefined` branch (and on the
// requireStringLiterals form of it). RuleMeta has no HasSuggestions field and
// eslint.Message has no `suggestions` field, so this port reports the same
// message/messageId/loc but cannot emit the suggestions array — see the
// package test.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Enforce comparing `typeof` expressions against valid strings",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/valid-typeof",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"requireStringLiterals": map[string]any{"type": "boolean", "default": false},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			"invalidValue":  "Invalid typeof comparison value.",
			"notString":     "Typeof comparisons should be to string literals.",
			"suggestString": "Use `\"{{type}}\"` instead of `{{type}}`.",
		},
	},
	Create: create,
}

// validTypes is the rule's VALID_TYPES Set. Only string values can be members,
// which is what makes `typeof x === 5` (or null, or a regex) invalid.
var validTypes = map[string]bool{
	"symbol":    true,
	"undefined": true,
	"object":    true,
	"boolean":   true,
	"number":    true,
	"string":    true,
	"function":  true,
	"bigint":    true,
}

// comparisonOperators is the rule's OPERATORS Set.
var comparisonOperators = map[string]bool{
	"==": true, "===": true, "!=": true, "!==": true,
}

// isTypeofExpression mirrors the rule's helper.
func isTypeofExpression(node eslint.Node) bool {
	return eslint.NodeType(node) == "UnaryExpression" && eslint.GetString(node, "operator") == "typeof"
}

// isStaticTemplateLiteral mirrors astUtils.isStaticTemplateLiteral.
func isStaticTemplateLiteral(node eslint.Node) bool {
	return eslint.NodeType(node) == "TemplateLiteral" && len(eslint.GetNodes(node, "expressions")) == 0
}

// isReferenceToGlobalVariable mirrors the rule's helper: the name is in the
// global scope, has no definition, and one of that variable's references is
// this very identifier node.
func isReferenceToGlobalVariable(globalScope *eslintscope.Scope, node eslint.Node) bool {
	if globalScope == nil {
		return false
	}
	variable := globalScope.Set[eslint.GetString(node, "name")]
	if variable == nil || len(variable.Defs) != 0 {
		return false
	}
	for _, ref := range variable.References {
		if eslint.SameNode(ref.Identifier, node) {
			return true
		}
	}
	return false
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	requireStringLiterals := false
	if m := ctx.OptionMap(0); m != nil {
		requireStringLiterals, _ = m["requireStringLiterals"].(bool)
	}

	var globalScope *eslintscope.Scope

	return map[string]func(eslint.Node){
		"Program": func(node eslint.Node) {
			globalScope = ctx.ScopeOf(node)
		},

		"UnaryExpression": func(node eslint.Node) {
			if !isTypeofExpression(node) {
				return
			}
			parent := eslint.Parent(node)

			if eslint.NodeType(parent) != "BinaryExpression" || !comparisonOperators[eslint.GetString(parent, "operator")] {
				return
			}
			sibling := eslint.GetNode(parent, "left")
			if eslint.SameNode(sibling, node) {
				sibling = eslint.GetNode(parent, "right")
			}

			switch {
			case eslint.NodeType(sibling) == "Literal" || isStaticTemplateLiteral(sibling):
				var value any
				if eslint.NodeType(sibling) == "Literal" {
					value = eslint.Get(sibling, "value")
				} else {
					quasis := eslint.GetNodes(sibling, "quasis")
					if len(quasis) > 0 {
						value = eslint.GetMap(quasis[0], "value")["cooked"]
					}
				}
				if s, ok := value.(string); !ok || !validTypes[s] {
					ctx.Report(eslint.Report{Node: sibling, MessageID: "invalidValue"})
				}

			case eslint.NodeType(sibling) == "Identifier" && eslint.GetString(sibling, "name") == "undefined" &&
				isReferenceToGlobalVariable(globalScope, sibling):
				messageID := "invalidValue"
				if requireStringLiterals {
					messageID = "notString"
				}
				ctx.Report(eslint.Report{
					Node:      sibling,
					MessageID: messageID,
					Suggest: []eslint.Suggestion{{
						MessageID: "suggestString",
						Data:      map[string]any{"type": "undefined"},
						Fix: func(f *eslint.Fixer) *eslint.Fix {
							return f.ReplaceText(sibling, `"undefined"`)
						},
					}},
				})

			case requireStringLiterals && !isTypeofExpression(sibling):
				ctx.Report(eslint.Report{Node: sibling, MessageID: "notString"})
			}
		},
	}
}
