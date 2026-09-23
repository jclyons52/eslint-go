package spaceinfixops

import (
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "space-infix-ops"

// Rule is a port of eslint/lib/rules/space-infix-ops.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "layout",
		Docs: eslint.RuleDocs{
			Description: "Require spacing around infix operators",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/space-infix-ops",
		},
		Fixable:    "whitespace",
		Deprecated: true,
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"int32Hint": map[string]any{"type": "boolean", "default": false},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			"missingSpace": "Operator '{{operator}}' must be spaced.",
		},
	},
	Create: create,
}

// isEqToken is astUtils.isEqToken (a private copy: the core exposes the generic
// punctuator predicates but not this one).
func isEqToken(token eslint.Node) bool {
	return eslint.TokenValue(token) == "=" && eslint.NodeType(token) == eslint.TokenPunctuator
}

// firstTokenBetween is sourceCode.getFirstTokenBetween(left, right, filter):
// the first token in (left.end, right.start) whose value equals op.
func firstTokenBetween(sc *eslint.SourceCode, left, right eslint.Node, op string) eslint.Node {
	tokens := sc.GetTokensBetween(left, right, eslint.TokenOpt{
		Filter: func(t eslint.Node) bool { return eslint.TokenValue(t) == op },
	})
	if len(tokens) == 0 {
		return nil
	}
	return tokens[0]
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	int32Hint := false
	if m := ctx.OptionMap(0); m != nil {
		int32Hint, _ = m["int32Hint"].(bool)
	}
	sc := ctx.SourceCode

	// getFirstNonSpacedToken returns the first token violating the rule.
	getFirstNonSpacedToken := func(left, right eslint.Node, op string) eslint.Node {
		operator := firstTokenBetween(sc, left, right, op)
		if operator == nil {
			return nil
		}
		prev := sc.GetTokenBefore(operator)
		next := sc.GetTokenAfter(operator)
		if !sc.IsSpaceBetweenTokens(prev, operator) || !sc.IsSpaceBetweenTokens(operator, next) {
			return operator
		}
		return nil
	}

	report := func(mainNode, culpritToken eslint.Node) {
		ctx.Report(eslint.Report{
			Node:      mainNode,
			Loc:       eslint.Loc(culpritToken),
			MessageID: "missingSpace",
			Data:      map[string]any{"operator": eslint.TokenValue(culpritToken)},
			Fix: func(f *eslint.Fixer) *eslint.Fix {
				previousToken := sc.GetTokenBefore(culpritToken)
				afterToken := sc.GetTokenAfter(culpritToken)
				fixString := ""
				if previousToken != nil && eslint.Start(culpritToken)-eslint.End(previousToken) == 0 {
					fixString = " "
				}
				fixString += eslint.TokenValue(culpritToken)
				if afterToken != nil && eslint.Start(afterToken)-eslint.End(culpritToken) == 0 {
					fixString += " "
				}
				return f.ReplaceText(culpritToken, fixString)
			},
		})
	}

	checkBinary := func(node eslint.Node) {
		leftNode := eslint.GetNode(node, "left")
		if ta := eslint.GetNode(leftNode, "typeAnnotation"); ta != nil {
			leftNode = ta
		}
		rightNode := eslint.GetNode(node, "right")

		// search for = in AssignmentPattern nodes
		operator := eslint.GetString(node, "operator")
		if operator == "" {
			operator = "="
		}

		nonSpacedNode := getFirstNonSpacedToken(leftNode, rightNode, operator)
		if nonSpacedNode != nil {
			if !(int32Hint && strings.HasSuffix(sc.GetText(node), "|0")) {
				report(node, nonSpacedNode)
			}
		}
	}

	checkConditional := func(node eslint.Node) {
		nonSpacedConsequentNode := getFirstNonSpacedToken(eslint.GetNode(node, "test"), eslint.GetNode(node, "consequent"), "?")
		nonSpacedAlternateNode := getFirstNonSpacedToken(eslint.GetNode(node, "consequent"), eslint.GetNode(node, "alternate"), ":")

		if nonSpacedConsequentNode != nil {
			report(node, nonSpacedConsequentNode)
		}
		if nonSpacedAlternateNode != nil {
			report(node, nonSpacedAlternateNode)
		}
	}

	checkVar := func(node eslint.Node) {
		idNode := eslint.GetNode(node, "id")
		leftNode := idNode
		if ta := eslint.GetNode(idNode, "typeAnnotation"); ta != nil {
			leftNode = ta
		}
		rightNode := eslint.GetNode(node, "init")

		if rightNode != nil {
			nonSpacedNode := getFirstNonSpacedToken(leftNode, rightNode, "=")
			if nonSpacedNode != nil {
				report(node, nonSpacedNode)
			}
		}
	}

	checkPropertyDefinition := func(node eslint.Node) {
		if eslint.Get(node, "value") == nil {
			return
		}

		/*
		 * Because of computed properties and type annotations, some tokens may
		 * exist between `node.key` and `=`. Therefore, find the `=` from the
		 * right.
		 */
		operatorToken := sc.GetTokenBefore(eslint.GetNode(node, "value"), eslint.TokenOpt{Filter: isEqToken})
		if operatorToken == nil {
			return
		}
		leftToken := sc.GetTokenBefore(operatorToken)
		rightToken := sc.GetTokenAfter(operatorToken)

		if !sc.IsSpaceBetweenTokens(leftToken, operatorToken) ||
			!sc.IsSpaceBetweenTokens(operatorToken, rightToken) {
			report(node, operatorToken)
		}
	}

	return map[string]func(eslint.Node){
		"AssignmentExpression":  checkBinary,
		"AssignmentPattern":     checkBinary,
		"BinaryExpression":      checkBinary,
		"LogicalExpression":     checkBinary,
		"ConditionalExpression": checkConditional,
		"VariableDeclarator":    checkVar,
		"PropertyDefinition":    checkPropertyDefinition,
	}
}
