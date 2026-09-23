package nocondassign

import eslint "github.com/jclyons52/eslint-go"

// Name is the ESLint rule id.
const Name = "no-cond-assign"

// Rule is a port of eslint/lib/rules/no-cond-assign.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Disallow assignment operators in conditional expressions",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-cond-assign",
		},
		Schema: []any{
			map[string]any{"enum": []any{"except-parens", "always"}},
		},
		Messages: map[string]string{
			"unexpected": "Unexpected assignment within {{type}}.",

			// must match JSHint's error message
			"missing": "Expected a conditional expression and instead saw an assignment.",
		},
	},
	Create: create,
}

// testConditionParentTypes is TEST_CONDITION_PARENT_TYPES.
var testConditionParentTypes = map[string]bool{
	"IfStatement":           true,
	"WhileStatement":        true,
	"DoWhileStatement":      true,
	"ForStatement":          true,
	"ConditionalExpression": true,
}

// nodeDescriptions is NODE_DESCRIPTIONS. A type missing here (the
// ConditionalExpression case) falls back to the raw node type.
var nodeDescriptions = map[string]string{
	"DoWhileStatement": "a 'do...while' statement",
	"ForStatement":     "a 'for' statement",
	"IfStatement":      "an 'if' statement",
	"WhileStatement":   "a 'while' statement",
}

// isParenthesised is astUtils' private isParenthesised: the tokens immediately
// surrounding the node are a `(`/`)` pair that does not overlap it. It compares
// the token *value* (not the token type), exactly as the original does.
func isParenthesised(sc *eslint.SourceCode, node eslint.Node) bool {
	previousToken := sc.GetTokenBefore(node)
	nextToken := sc.GetTokenAfter(node)

	return previousToken != nil && nextToken != nil &&
		eslint.TokenValue(previousToken) == "(" && eslint.End(previousToken) <= eslint.Start(node) &&
		eslint.TokenValue(nextToken) == ")" && eslint.Start(nextToken) >= eslint.End(node)
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	// const prohibitAssign = (context.options[0] || "except-parens");
	prohibitAssign := ctx.OptionString(0, "except-parens")
	if prohibitAssign == "" {
		prohibitAssign = "except-parens"
	}

	sc := ctx.SourceCode

	// isConditionalTestExpression: is the node the test expression of a
	// conditional statement?
	isConditionalTestExpression := func(node eslint.Node) bool {
		parent := eslint.Parent(node)
		return parent != nil &&
			testConditionParentTypes[eslint.NodeType(parent)] &&
			eslint.SameNode(node, eslint.GetNode(parent, "test"))
	}

	// findConditionalAncestor: bottom-up search for the first ancestor that
	// represents a conditional statement.
	//
	// JS: do { … } while ((currentAncestor = currentAncestor.parent) &&
	// !astUtils.isFunction(currentAncestor));
	findConditionalAncestor := func(node eslint.Node) eslint.Node {
		currentAncestor := node
		for {
			if isConditionalTestExpression(currentAncestor) {
				return eslint.Parent(currentAncestor)
			}
			currentAncestor = eslint.Parent(currentAncestor)
			if currentAncestor == nil || eslint.IsFunction(currentAncestor) {
				return nil
			}
		}
	}

	// isParenthesisedTwice: is the code enclosed in two sets of parentheses?
	isParenthesisedTwice := func(node eslint.Node) bool {
		previousToken := sc.GetTokenBefore(node, eslint.TokenOpt{Skip: 1})
		nextToken := sc.GetTokenAfter(node, eslint.TokenOpt{Skip: 1})

		return isParenthesised(sc, node) &&
			previousToken != nil && eslint.IsOpeningParenToken(previousToken) && eslint.End(previousToken) <= eslint.Start(node) &&
			eslint.IsClosingParenToken(nextToken) && eslint.Start(nextToken) >= eslint.End(node)
	}

	// testForAssign: check a conditional statement's test for a top-level
	// assignment that is not enclosed in parentheses.
	testForAssign := func(node eslint.Node) {
		test := eslint.GetNode(node, "test")
		if test == nil || eslint.NodeType(test) != "AssignmentExpression" {
			return
		}
		var parenthesised bool
		if eslint.NodeType(node) == "ForStatement" {
			parenthesised = isParenthesised(sc, test)
		} else {
			parenthesised = isParenthesisedTwice(test)
		}
		if parenthesised {
			return
		}
		ctx.Report(eslint.Report{Node: test, MessageID: "missing"})
	}

	// testForConditionalAncestor: is the assignment descended from a
	// conditional statement's test expression?
	testForConditionalAncestor := func(node eslint.Node) {
		ancestor := findConditionalAncestor(node)
		if ancestor == nil {
			return
		}
		ancestorType := eslint.NodeType(ancestor)
		description, ok := nodeDescriptions[ancestorType]
		if !ok {
			description = ancestorType
		}
		ctx.Report(eslint.Report{
			Node:      node,
			MessageID: "unexpected",
			Data:      map[string]any{"type": description},
		})
	}

	if prohibitAssign == "always" {
		return map[string]func(eslint.Node){
			"AssignmentExpression": testForConditionalAncestor,
		}
	}

	return map[string]func(eslint.Node){
		"DoWhileStatement":      testForAssign,
		"ForStatement":          testForAssign,
		"IfStatement":           testForAssign,
		"WhileStatement":        testForAssign,
		"ConditionalExpression": testForAssign,
	}
}
