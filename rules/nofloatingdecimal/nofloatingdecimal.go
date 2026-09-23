package nofloatingdecimal

import (
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-floating-decimal"

// Rule is a port of eslint/lib/rules/no-floating-decimal.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow leading or trailing decimal points in numeric literals",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-floating-decimal",
		},
		Fixable:    "code",
		Deprecated: true,
		Schema:     []any{},
		Messages: map[string]string{
			"leading":  "A leading decimal point can be confused with a dot.",
			"trailing": "A trailing decimal point can be confused with a dot.",
		},
	},
	Create: create,
}

// canTokensBeAdjacent is a private port of astUtils.canTokensBeAdjacent for the
// one shape this rule produces: leftValue is a real source token and rightValue
// is the synthesized numeric literal `0${node.raw}`. Because the right side is
// always a Numeric token (tokenizing "0" + a raw that starts with "." can only
// yield a number), the string-tokenization branch of the original collapses to
// a fixed type/value pair.
func canTokensBeAdjacent(leftToken eslint.Node, rightType, rightValue string) bool {
	if leftToken == nil {
		return false
	}

	leftType := eslint.NodeType(leftToken)
	leftValue := eslint.TokenValue(leftToken)

	// A hashbang token can never be adjacent to anything.
	if leftType == "Shebang" || leftType == "Hashbang" {
		return false
	}

	if leftType == eslint.TokenPunctuator || rightType == eslint.TokenPunctuator {
		if leftType == eslint.TokenPunctuator && rightType == eslint.TokenPunctuator {
			isPlus := func(v string) bool { return v == "+" || v == "++" }
			isMinus := func(v string) bool { return v == "-" || v == "--" }
			return !((isPlus(leftValue) && isPlus(rightValue)) || (isMinus(leftValue) && isMinus(rightValue)))
		}
		if leftType == eslint.TokenPunctuator && leftValue == "/" {
			return rightType != "Block" && rightType != "Line" && rightType != "RegularExpression"
		}
		return true
	}

	if leftType == eslint.TokenString || rightType == eslint.TokenString ||
		leftType == eslint.TokenTemplate || rightType == eslint.TokenTemplate {
		return true
	}

	if leftType != eslint.TokenNumeric && rightType == eslint.TokenNumeric && strings.HasPrefix(rightValue, ".") {
		return true
	}

	if leftType == "Block" || rightType == "Block" || rightType == "Line" {
		return true
	}

	if rightType == "PrivateIdentifier" {
		return true
	}

	return false
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode

	return map[string]func(eslint.Node){
		"Literal": func(node eslint.Node) {
			if _, isNumber := eslint.Get(node, "value").(float64); !isNumber {
				return
			}
			raw := eslint.GetString(node, "raw")

			if strings.HasPrefix(raw, ".") {
				ctx.Report(eslint.Report{
					Node:      node,
					MessageID: "leading",
					Fix: func(f *eslint.Fixer) *eslint.Fix {
						tokenBefore := sc.GetTokenBefore(node)
						needsSpaceBefore := tokenBefore != nil &&
							eslint.End(tokenBefore) == eslint.Start(node) &&
							!canTokensBeAdjacent(tokenBefore, eslint.TokenNumeric, "0"+raw)

						if needsSpaceBefore {
							return f.InsertTextBefore(node, " 0")
						}
						return f.InsertTextBefore(node, "0")
					},
				})
			}

			if strings.Index(raw, ".") == len(raw)-1 {
				ctx.Report(eslint.Report{
					Node:      node,
					MessageID: "trailing",
					Fix: func(f *eslint.Fixer) *eslint.Fix {
						return f.InsertTextAfter(node, "0")
					},
				})
			}
		},
	}
}
