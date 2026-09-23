package yoda

import (
	"math/big"
	"strconv"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "yoda"

// Rule is a port of eslint/lib/rules/yoda.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: `Require or disallow "Yoda" conditions`,
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/yoda",
		},
		Fixable: "code",
		Schema: []any{
			map[string]any{"enum": []any{"always", "never"}},
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"exceptRange":  map[string]any{"type": "boolean", "default": false},
					"onlyEquality": map[string]any{"type": "boolean", "default": false},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			"expected": "Expected literal to be on the {{expectedSide}} side of {{operator}}.",
		},
	},
	Create: create,
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// isComparisonOperator mirrors the rule's local regexp /^(==|===|!=|!==|<|>|<=|>=)$/.
func isComparisonOperator(op string) bool {
	switch op {
	case "==", "===", "!=", "!==", "<", ">", "<=", ">=":
		return true
	}
	return false
}

// isEqualityOperator mirrors /^(==|===)$/.
func isEqualityOperator(op string) bool {
	return op == "==" || op == "==="
}

// isRangeTestOperator mirrors ["<", "<="].includes(operator).
func isRangeTestOperator(op string) bool {
	return op == "<" || op == "<="
}

// isNumericLiteral is astUtils.isNumericLiteral: a Literal whose value is a
// number, or a BigInt literal (whose Go value is its digit string).
func isNumericLiteral(node eslint.Node) bool {
	if eslint.NodeType(node) != "Literal" {
		return false
	}
	if _, ok := eslint.Get(node, "value").(float64); ok {
		return true
	}
	return eslint.GetString(node, "bigint") != ""
}

// isStaticTemplateLiteral is astUtils.isStaticTemplateLiteral.
func isStaticTemplateLiteral(node eslint.Node) bool {
	return eslint.NodeType(node) == "TemplateLiteral" && len(eslint.GetNodes(node, "expressions")) == 0
}

// isNegativeNumericLiteral mirrors the rule's local helper.
func isNegativeNumericLiteral(node eslint.Node) bool {
	return eslint.NodeType(node) == "UnaryExpression" &&
		eslint.GetString(node, "operator") == "-" &&
		eslint.GetBool(node, "prefix") &&
		isNumericLiteral(eslint.GetNode(node, "argument"))
}

// looksLikeLiteral mirrors the rule's local helper.
func looksLikeLiteral(node eslint.Node) bool {
	return isNegativeNumericLiteral(node) || isStaticTemplateLiteral(node)
}

// jsValue is a normalised JavaScript value, used to reproduce the `<=`
// comparisons the rule performs on the two literals of a range test.
type jsValue interface{ isJSValue() }

type jsNumber struct{ v float64 }
type jsString struct{ v string }
type jsBigint struct{ v *big.Int }
type jsBool struct{ v bool }
type jsNull struct{}
type jsUndefined struct{}

func (jsNumber) isJSValue()    {}
func (jsString) isJSValue()    {}
func (jsBigint) isJSValue()    {}
func (jsBool) isJSValue()      {}
func (jsNull) isJSValue()      {}
func (jsUndefined) isJSValue() {}

// normalizedLiteral is getNormalizedLiteral's result: either "no literal here"
// (ok == false, the JS `null`) or a JavaScript value.
type normalizedLiteral struct {
	value jsValue
	ok    bool
}

// getNormalizedLiteral mirrors the rule's local helper.
func getNormalizedLiteral(node eslint.Node) normalizedLiteral {
	if eslint.NodeType(node) == "Literal" {
		return normalizedLiteral{value: literalValue(node), ok: true}
	}

	if isNegativeNumericLiteral(node) {
		arg := eslint.GetNode(node, "argument")
		if digits := eslint.GetString(arg, "bigint"); digits != "" {
			if n, ok := new(big.Int).SetString(digits, 10); ok {
				return normalizedLiteral{value: jsBigint{v: n.Neg(n)}, ok: true}
			}
			return normalizedLiteral{value: jsUndefined{}, ok: true}
		}
		return normalizedLiteral{value: jsNumber{v: -eslint.GetFloat(arg, "value")}, ok: true}
	}

	if isStaticTemplateLiteral(node) {
		quasis := eslint.GetNodes(node, "quasis")
		if len(quasis) > 0 {
			cooked, has := eslint.GetMap(quasis[0], "value")["cooked"].(string)
			if !has {
				return normalizedLiteral{value: jsUndefined{}, ok: true}
			}
			return normalizedLiteral{value: jsString{v: cooked}, ok: true}
		}
		return normalizedLiteral{value: jsUndefined{}, ok: true}
	}

	return normalizedLiteral{}
}

// literalValue maps a Literal node's value into a jsValue. Regex literals (and
// any other object-valued literal) become the undefined value, which is what
// JavaScript's relational comparison sees (NaN).
func literalValue(node eslint.Node) jsValue {
	if digits := eslint.GetString(node, "bigint"); digits != "" {
		if n, ok := new(big.Int).SetString(digits, 10); ok {
			return jsBigint{v: n}
		}
		return jsUndefined{}
	}
	switch v := eslint.Get(node, "value").(type) {
	case float64:
		return jsNumber{v: v}
	case string:
		return jsString{v: v}
	case bool:
		return jsBool{v: v}
	case nil:
		return jsNull{}
	}
	return jsUndefined{}
}

// toFloat converts a jsValue to a float64 the way JavaScript's abstract
// ToNumber does for the primitives the rule can produce.
func toFloat(v jsValue) (float64, bool) {
	switch t := v.(type) {
	case jsNumber:
		return t.v, true
	case jsBool:
		if t.v {
			return 1, true
		}
		return 0, true
	case jsNull:
		return 0, true
	case jsString:
		s := strings.TrimSpace(t.v)
		if s == "" {
			return 0, true
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	case jsBigint:
		f, _ := new(big.Float).SetInt(t.v).Float64()
		return f, true
	}
	return 0, false
}

// jsLessEqual reproduces JavaScript's `left <= right` for the value shapes a
// range test can produce (both strings compare lexicographically, otherwise
// both sides are converted to numbers and a NaN comparison is false).
func jsLessEqual(left, right jsValue) bool {
	ls, lIsStr := left.(jsString)
	rs, rIsStr := right.(jsString)
	if lIsStr && rIsStr {
		return ls.v <= rs.v
	}

	lb, lIsBig := left.(jsBigint)
	rb, rIsBig := right.(jsBigint)
	if lIsBig && rIsBig {
		return lb.v.Cmp(rb.v) <= 0
	}
	if lIsBig || rIsBig {
		lf, lok := toFloat(left)
		rf, rok := toFloat(right)
		if !lok || !rok {
			return false
		}
		return big.NewFloat(lf).Cmp(big.NewFloat(rf)) <= 0
	}

	lf, lok := toFloat(left)
	rf, rok := toFloat(right)
	if !lok || !rok {
		return false
	}
	return lf <= rf
}

// staticPropertyName is a private, faithful port of astUtils.getStaticPropertyName
// for the MemberExpression case isSameReference uses. The core's
// GetStaticPropertyName is equivalent for identifiers, strings, null, regex and
// bigint literals but formats a non-integer numeric key through int truncation
// (String(1.5) is "1.5", not "1"), so the numeric branch is redone here.
func staticPropertyName(node eslint.Node) (string, bool) {
	if eslint.NodeType(node) == "ChainExpression" {
		return staticPropertyName(eslint.GetNode(node, "expression"))
	}
	var prop eslint.Node
	switch eslint.NodeType(node) {
	case "Property", "PropertyDefinition", "MethodDefinition":
		prop = eslint.GetNode(node, "key")
	case "MemberExpression":
		prop = eslint.GetNode(node, "property")
	default:
		return "", false
	}
	if prop == nil {
		return "", false
	}
	if eslint.NodeType(prop) == "Identifier" && !eslint.GetBool(node, "computed") {
		return eslint.GetString(prop, "name"), true
	}
	return staticStringValue(prop)
}

// staticStringValue is astUtils.getStaticStringValue.
func staticStringValue(node eslint.Node) (string, bool) {
	switch eslint.NodeType(node) {
	case "Literal":
		v, has := node["value"]
		if !has || v == nil {
			if eslint.GetString(node, "raw") == "null" {
				return "null", true
			}
			if re := eslint.GetMap(node, "regex"); re != nil {
				return "/" + eslint.GetString(eslint.Node(re), "pattern") + "/" + eslint.GetString(eslint.Node(re), "flags"), true
			}
			if digits := eslint.GetString(node, "bigint"); digits != "" {
				return digits, true
			}
			return "", false
		}
		switch t := v.(type) {
		case string:
			return t, true
		case float64:
			return jsNumberToString(t), true
		case bool:
			if t {
				return "true", true
			}
			return "false", true
		}
		return "", false
	case "TemplateLiteral":
		if len(eslint.GetNodes(node, "expressions")) == 0 && len(eslint.GetNodes(node, "quasis")) == 1 {
			quasi := eslint.GetNodes(node, "quasis")[0]
			cooked, has := eslint.GetMap(quasi, "value")["cooked"].(string)
			if !has {
				return "", false
			}
			return cooked, true
		}
	}
	return "", false
}

// jsNumberToString is JS String(number) for the finite numbers a literal key
// can hold.
func jsNumberToString(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// equalLiteralValue is astUtils.equalLiteralValue.
func equalLiteralValue(left, right eslint.Node) bool {
	lre, rre := eslint.GetMap(left, "regex"), eslint.GetMap(right, "regex")
	if lre != nil || rre != nil {
		return lre != nil && rre != nil &&
			eslint.GetString(eslint.Node(lre), "pattern") == eslint.GetString(eslint.Node(rre), "pattern") &&
			eslint.GetString(eslint.Node(lre), "flags") == eslint.GetString(eslint.Node(rre), "flags")
	}
	lb, rb := eslint.GetString(left, "bigint"), eslint.GetString(right, "bigint")
	if lb != "" || rb != "" {
		return lb == rb
	}
	return jsStrictEqual(eslint.Get(left, "value"), eslint.Get(right, "value"))
}

// jsStrictEqual is `===` over the literal values the parser produces (numbers,
// strings, booleans, null, undefined).
func jsStrictEqual(a, b any) bool {
	switch x := a.(type) {
	case float64:
		y, ok := b.(float64)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case nil:
		return b == nil
	}
	// Object-valued literals (regex) are never identical unless they are the
	// same object, which two distinct literals never are.
	return false
}

// isSameReference is astUtils.isSameReference (with the default
// disableStaticComputedKey = false).
func isSameReference(left, right eslint.Node) bool {
	if eslint.NodeType(left) != eslint.NodeType(right) {
		// Handle `a.b` and `a?.b` are samely.
		if eslint.NodeType(left) == "ChainExpression" {
			return isSameReference(eslint.GetNode(left, "expression"), right)
		}
		if eslint.NodeType(right) == "ChainExpression" {
			return isSameReference(left, eslint.GetNode(right, "expression"))
		}
		return false
	}

	switch eslint.NodeType(left) {
	case "Super", "ThisExpression":
		return true
	case "Identifier", "PrivateIdentifier":
		return eslint.GetString(left, "name") == eslint.GetString(right, "name")
	case "Literal":
		return equalLiteralValue(left, right)
	case "ChainExpression":
		return isSameReference(eslint.GetNode(left, "expression"), eslint.GetNode(right, "expression"))
	case "MemberExpression":
		if nameA, ok := staticPropertyName(left); ok {
			// x.y = x["y"]
			nameB, okB := staticPropertyName(right)
			return isSameReference(eslint.GetNode(left, "object"), eslint.GetNode(right, "object")) &&
				okB && nameA == nameB
		}

		/*
		 * x[0] = x[0]
		 * x[y] = x[y]
		 * x.y = x.y
		 */
		return eslint.GetBool(left, "computed") == eslint.GetBool(right, "computed") &&
			isSameReference(eslint.GetNode(left, "object"), eslint.GetNode(right, "object")) &&
			isSameReference(eslint.GetNode(left, "property"), eslint.GetNode(right, "property"))
	}
	return false
}

// isParenthesised is astUtils.isParenthesised (note: not the same helper as the
// core's IsParenthesized(times, node, sourceCode)).
func isParenthesised(sc *eslint.SourceCode, node eslint.Node) bool {
	previousToken := sc.GetTokenBefore(node)
	nextToken := sc.GetTokenAfter(node)

	return previousToken != nil && nextToken != nil &&
		eslint.TokenValue(previousToken) == "(" && eslint.End(previousToken) <= eslint.Start(node) &&
		eslint.TokenValue(nextToken) == ")" && eslint.Start(nextToken) >= eslint.End(node)
}

// canTokensBeAdjacent is astUtils.canTokensBeAdjacent for two real tokens
// (yoda always passes tokens, never strings, so the espree-tokenize branch of
// the original is unreachable here).
func canTokensBeAdjacent(leftToken, rightToken eslint.Node) bool {
	if leftToken == nil || rightToken == nil {
		return false
	}

	leftType, rightType := eslint.NodeType(leftToken), eslint.NodeType(rightToken)
	leftValue, rightValue := eslint.TokenValue(leftToken), eslint.TokenValue(rightToken)

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

// operatorFlipMap is the rule's OPERATOR_FLIP_MAP.
var operatorFlipMap = map[string]string{
	"===": "===",
	"!==": "!==",
	"==":  "==",
	"!=":  "!=",
	"<":   ">",
	">":   "<",
	"<=":  ">=",
	">=":  "<=",
}

// isRangeTest mirrors the rule's local helper.
func isRangeTest(sc *eslint.SourceCode, node eslint.Node) bool {
	if eslint.NodeType(node) != "LogicalExpression" {
		return false
	}
	left := eslint.GetNode(node, "left")
	right := eslint.GetNode(node, "right")

	if eslint.NodeType(left) != "BinaryExpression" || eslint.NodeType(right) != "BinaryExpression" {
		return false
	}
	if !isRangeTestOperator(eslint.GetString(left, "operator")) || !isRangeTestOperator(eslint.GetString(right, "operator")) {
		return false
	}
	if !isBetweenTest(node, left, right) && !isOutsideTest(node, left, right) {
		return false
	}
	return isParenthesised(sc, node)
}

// isBetweenTest mirrors the rule's inner helper for `0 <= x && x < 1`.
func isBetweenTest(node, left, right eslint.Node) bool {
	if eslint.GetString(node, "operator") != "&&" ||
		!isSameReference(eslint.GetNode(left, "right"), eslint.GetNode(right, "left")) {
		return false
	}
	leftLiteral := getNormalizedLiteral(eslint.GetNode(left, "left"))
	rightLiteral := getNormalizedLiteral(eslint.GetNode(right, "right"))

	if !leftLiteral.ok && !rightLiteral.ok {
		return false
	}
	if !rightLiteral.ok || !leftLiteral.ok {
		return true
	}
	return jsLessEqual(leftLiteral.value, rightLiteral.value)
}

// isOutsideTest mirrors the rule's inner helper for `x < 0 || 1 <= x`.
func isOutsideTest(node, left, right eslint.Node) bool {
	if eslint.GetString(node, "operator") != "||" ||
		!isSameReference(eslint.GetNode(left, "left"), eslint.GetNode(right, "right")) {
		return false
	}
	leftLiteral := getNormalizedLiteral(eslint.GetNode(left, "right"))
	rightLiteral := getNormalizedLiteral(eslint.GetNode(right, "left"))

	if !leftLiteral.ok && !rightLiteral.ok {
		return false
	}
	if !rightLiteral.ok || !leftLiteral.ok {
		return true
	}
	return jsLessEqual(leftLiteral.value, rightLiteral.value)
}

// getFlippedString mirrors the rule's local helper.
func getFlippedString(sc *eslint.SourceCode, node eslint.Node) string {
	left := eslint.GetNode(node, "left")
	right := eslint.GetNode(node, "right")
	operator := eslint.GetString(node, "operator")

	tokens := sc.GetTokensBetween(left, right, eslint.TokenOpt{
		Filter: func(t eslint.Node) bool { return eslint.TokenValue(t) == operator },
	})
	if len(tokens) == 0 {
		return sc.GetText(node)
	}
	operatorToken := tokens[0]
	lastLeftToken := sc.GetTokenBefore(operatorToken)
	firstRightToken := sc.GetTokenAfter(operatorToken)
	if lastLeftToken == nil || firstRightToken == nil {
		return sc.GetText(node)
	}

	source := sc.Text()
	leftText := source[eslint.Start(node):eslint.End(lastLeftToken)]
	textBeforeOperator := source[eslint.End(lastLeftToken):eslint.Start(operatorToken)]
	textAfterOperator := source[eslint.End(operatorToken):eslint.Start(firstRightToken)]
	rightText := source[eslint.Start(firstRightToken):eslint.End(node)]

	tokenBefore := sc.GetTokenBefore(node)
	tokenAfter := sc.GetTokenAfter(node)
	prefix, suffix := "", ""

	if tokenBefore != nil &&
		eslint.End(tokenBefore) == eslint.Start(node) &&
		!canTokensBeAdjacent(tokenBefore, firstRightToken) {
		prefix = " "
	}

	if tokenAfter != nil &&
		eslint.End(node) == eslint.Start(tokenAfter) &&
		!canTokensBeAdjacent(lastLeftToken, tokenAfter) {
		suffix = " "
	}

	return prefix +
		rightText +
		textBeforeOperator +
		operatorFlipMap[eslint.TokenValue(operatorToken)] +
		textAfterOperator +
		leftText +
		suffix
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	// Default to "never" (!always) if no option
	always := ctx.OptionString(0, "") == "always"
	exceptRange := false
	onlyEquality := false
	if m := ctx.OptionMap(1); m != nil {
		exceptRange, _ = m["exceptRange"].(bool)
		onlyEquality, _ = m["onlyEquality"].(bool)
	}

	sc := ctx.SourceCode

	return map[string]func(eslint.Node){
		"BinaryExpression": func(node eslint.Node) {
			var expectedLiteral, expectedNonLiteral eslint.Node
			if always {
				expectedLiteral = eslint.GetNode(node, "left")
				expectedNonLiteral = eslint.GetNode(node, "right")
			} else {
				expectedLiteral = eslint.GetNode(node, "right")
				expectedNonLiteral = eslint.GetNode(node, "left")
			}

			// If `expectedLiteral` is not a literal, and `expectedNonLiteral`
			// is a literal, raise an error.
			if (eslint.NodeType(expectedNonLiteral) == "Literal" || looksLikeLiteral(expectedNonLiteral)) &&
				!(eslint.NodeType(expectedLiteral) == "Literal" || looksLikeLiteral(expectedLiteral)) &&
				!(onlyEquality && !isEqualityOperator(eslint.GetString(node, "operator"))) &&
				isComparisonOperator(eslint.GetString(node, "operator")) &&
				!(exceptRange && isRangeTest(sc, eslint.Parent(node))) {
				ctx.Report(eslint.Report{
					Node:      node,
					MessageID: "expected",
					Data: map[string]any{
						"operator":     eslint.GetString(node, "operator"),
						"expectedSide": map[bool]string{true: "left", false: "right"}[always],
					},
					Fix: func(f *eslint.Fixer) *eslint.Fix {
						return f.ReplaceText(node, getFlippedString(sc, node))
					},
				})
			}
		},
	}
}
