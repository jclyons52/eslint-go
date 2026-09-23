package nowhitespacebeforeproperty

import (
	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-whitespace-before-property"

// Rule is a port of eslint/lib/rules/no-whitespace-before-property.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "layout",
		Docs: eslint.RuleDocs{
			Description: "Disallow whitespace before properties",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-whitespace-before-property",
		},
		Fixable:    "whitespace",
		Deprecated: true,
		Schema:     []any{},
		Messages: map[string]string{
			"unexpectedWhitespace": "Unexpected whitespace before property {{propName}}.",
		},
	},
	Create: create,
}

// isTokenOnSameLine is astUtils.isTokenOnSameLine.
func isTokenOnSameLine(left, right eslint.Node) bool {
	leftEndLine, _ := eslint.LocEnd(left)
	rightStartLine, _ := eslint.LocStart(right)
	return leftEndLine == rightStartLine
}

// commentsExistBetween is SourceCode#commentsExistBetween: the first comment
// starting at or after left.end must end at or before right.start.
func commentsExistBetween(sc *eslint.SourceCode, left, right eslint.Node) bool {
	for _, comment := range sc.GetAllComments() {
		if eslint.Start(comment) >= eslint.End(left) {
			return eslint.End(comment) <= eslint.Start(right)
		}
	}
	return false
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode

	reportError := func(node, leftToken, rightToken eslint.Node) {
		computed := eslint.GetBool(node, "computed")
		optional := eslint.GetBool(node, "optional")
		object := eslint.GetNode(node, "object")

		ctx.Report(eslint.Report{
			Node:      node,
			MessageID: "unexpectedWhitespace",
			Data:      map[string]any{"propName": sc.GetText(eslint.GetNode(node, "property"))},
			Fix: func(f *eslint.Fixer) *eslint.Fix {
				if !computed && !optional && eslint.IsDecimalInteger(object) {
					/*
					 * If the object is a number literal, fixing it to something
					 * like 5.toString() would cause a SyntaxError. Don't fix this
					 * case.
					 */
					return nil
				}

				// Don't fix if comments exist.
				if commentsExistBetween(sc, leftToken, rightToken) {
					return nil
				}

				replacementText := ""
				if optional {
					replacementText = "?."
				} else if !computed {
					replacementText = "."
				}

				return f.ReplaceTextRange([2]int{eslint.End(leftToken), eslint.Start(rightToken)}, replacementText)
			},
		})
	}

	return map[string]func(eslint.Node){
		"MemberExpression": func(node eslint.Node) {
			object := eslint.GetNode(node, "object")
			property := eslint.GetNode(node, "property")

			if !isTokenOnSameLine(object, property) {
				return
			}

			var rightToken, leftToken eslint.Node
			if eslint.GetBool(node, "computed") {
				rightToken = sc.GetTokenBefore(property, eslint.TokenOpt{Filter: eslint.IsOpeningBracketToken})
				skip := 0
				if eslint.GetBool(node, "optional") {
					skip = 1
				}
				leftToken = sc.GetTokenBefore(rightToken, eslint.TokenOpt{Skip: skip})
			} else {
				rightToken = sc.GetFirstToken(property)
				leftToken = sc.GetTokenBefore(rightToken, eslint.TokenOpt{Skip: 1})
			}

			if leftToken == nil || rightToken == nil {
				return
			}

			if sc.IsSpaceBetweenTokens(leftToken, rightToken) {
				reportError(node, leftToken, rightToken)
			}
		},
	}
}
