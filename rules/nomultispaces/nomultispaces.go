package nomultispaces

import (
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "no-multi-spaces"

// Rule is a port of eslint/lib/rules/no-multi-spaces.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "layout",
		Docs: eslint.RuleDocs{
			Description: "Disallow multiple spaces",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-multi-spaces",
		},
		Fixable:    "whitespace",
		Deprecated: true,
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"exceptions": map[string]any{
						"type": "object",
						"patternProperties": map[string]any{
							"^([A-Z][a-z]*)+$": map[string]any{"type": "boolean"},
						},
						"additionalProperties": false,
					},
					"ignoreEOLComments": map[string]any{"type": "boolean", "default": false},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			"multipleSpaces": "Multiple spaces found before '{{displayValue}}'.",
		},
	},
	Create: create,
}

// formatReportedCommentValue mirrors the rule's local helper: the comment's
// first line, truncated to 12 characters (with an ellipsis) when the comment is
// multi-line or longer than 12 characters.
func formatReportedCommentValue(token eslint.Node) string {
	valueLines := strings.Split(eslint.TokenValue(token), "\n")
	value := valueLines[0]
	if len(valueLines) == 1 && len(value) <= 12 {
		return value
	}
	if len(value) > 12 {
		value = value[:12]
	}
	return value + "..."
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	options := ctx.OptionMap(0)

	ignoreEOLComments := false
	exceptions := map[string]bool{"Property": true}
	if options != nil {
		if b, ok := options["ignoreEOLComments"].(bool); ok {
			ignoreEOLComments = b
		}
		// Object.assign({ Property: true }, options.exceptions)
		if ex, ok := options["exceptions"].(map[string]any); ok {
			for key, v := range ex {
				if b, ok := v.(bool); ok {
					exceptions[key] = b
				}
			}
		}
	}
	hasExceptions := false
	for _, v := range exceptions {
		if v {
			hasExceptions = true
			break
		}
	}

	return map[string]func(eslint.Node){
		"Program": func(node eslint.Node) {
			tokensAndComments := sc.AllTokens()
			for leftIndex := 0; leftIndex < len(tokensAndComments)-1; leftIndex++ {
				leftToken := tokensAndComments[leftIndex]
				rightToken := tokensAndComments[leftIndex+1]

				// Ignore tokens that don't have 2 spaces between them or are on
				// different lines.
				between := sc.GetTextRange([2]int{eslint.End(leftToken), eslint.Start(rightToken)})
				if !strings.Contains(between, "  ") {
					continue
				}
				leftEndLine, _ := eslint.LocEnd(leftToken)
				rightStartLine, _ := eslint.LocStart(rightToken)
				if leftEndLine < rightStartLine {
					continue
				}

				// Ignore comments that are the last token on their line if
				// `ignoreEOLComments` is active.
				if ignoreEOLComments && eslint.IsCommentToken(rightToken) {
					lastOnLine := leftIndex == len(tokensAndComments)-2
					if !lastOnLine {
						rightEndLine, _ := eslint.LocEnd(rightToken)
						afterStartLine, _ := eslint.LocStart(tokensAndComments[leftIndex+2])
						lastOnLine = rightEndLine < afterStartLine
					}
					if lastOnLine {
						continue
					}
				}

				// Ignore tokens that are in a node in the "exceptions" object.
				if hasExceptions {
					parentNode := sc.GetNodeByRangeIndex(eslint.Start(rightToken) - 1)
					if parentNode != nil && exceptions[eslint.NodeType(parentNode)] {
						continue
					}
				}

				var displayValue string
				switch eslint.NodeType(rightToken) {
				case eslint.CommentBlock:
					displayValue = "/*" + formatReportedCommentValue(rightToken) + "*/"
				case eslint.CommentLine:
					displayValue = "//" + formatReportedCommentValue(rightToken)
				default:
					displayValue = eslint.TokenValue(rightToken)
				}

				lEndLine, lEndCol := eslint.LocEnd(leftToken)
				rStartLine, rStartCol := eslint.LocStart(rightToken)
				fixRange := [2]int{eslint.End(leftToken), eslint.Start(rightToken)}

				ctx.Report(eslint.Report{
					Node:      rightToken,
					Loc:       eslint.LocOf(lEndLine, lEndCol, rStartLine, rStartCol),
					MessageID: "multipleSpaces",
					Data:      map[string]any{"displayValue": displayValue},
					Fix: func(f *eslint.Fixer) *eslint.Fix {
						return f.ReplaceTextRange(fixRange, " ")
					},
				})
			}
		},
	}
}
