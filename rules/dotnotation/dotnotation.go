// Package dotnotation is a port of eslint/lib/rules/dot-notation.js.
//
// The rule rewrites `obj["foo"]` to `obj.foo` (and, with allowKeywords: false,
// `obj.catch` to `obj["catch"]`). Its fixes are *merged* before being reported:
// ESLint's report-translator combines the iterator of fixes a generator fix
// yields into one fix, so the merge (and its "must not overlap" rule) is part
// of the observable output and is ported here.
package dotnotation

import (
	"fmt"
	"regexp"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "dot-notation"

// Rule is a port of eslint/lib/rules/dot-notation.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Enforce dot notation whenever possible",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/dot-notation",
		},
		Fixable: "code",
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"allowKeywords": map[string]any{"type": "boolean", "default": true},
					"allowPattern":  map[string]any{"type": "string", "default": ""},
				},
				"additionalProperties": false,
			},
		},
		Messages: map[string]string{
			"useDot":      "[{{key}}] is better written in dot notation.",
			"useBrackets": ".{{key}} is a syntax error.",
		},
	},
	Create: create,
}

// validIdentifier mirrors the rule's /^[a-zA-Z_$][a-zA-Z0-9_$]*$/u.
var validIdentifier = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*$`)

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	options := ctx.OptionMap(0)
	allowKeywords := true
	if options != nil {
		if v, ok := options["allowKeywords"].(bool); ok {
			allowKeywords = v
		}
	}
	sc := ctx.SourceCode

	var allowPattern *regexp.Regexp
	if p, ok := options["allowPattern"].(string); ok && p != "" {
		allowPattern = regexp.MustCompile(jsRegexpToGo(p))
	}

	// checkComputedProperty reports `obj["foo"]` / obj[`foo`] as a dot-notation
	// candidate. value is the JS property value (string, boolean or null) and
	// isLiteral distinguishes a Literal property from a template literal.
	checkComputedProperty := func(node eslint.Node, value any, isLiteral bool) {
		strValue := jsString(value)

		if !validIdentifier.MatchString(strValue) ||
			(!allowKeywords && jsKeywords[strValue]) ||
			(allowPattern != nil && allowPattern.MatchString(strValue)) {
			return
		}

		formattedValue := "`" + strValue + "`"
		if isLiteral {
			formattedValue = jsonStringify(value)
		}

		prop := eslint.GetNode(node, "property")
		ctx.Report(eslint.Report{
			Node:      prop,
			MessageID: "useDot",
			Data:      map[string]any{"key": formattedValue},
			Fix: func(f *eslint.Fixer) *eslint.Fix {
				leftBracket := sc.GetTokenAfter(eslint.GetNode(node, "object"),
					eslint.TokenOpt{Filter: eslint.IsOpeningBracketToken})
				rightBracket := sc.GetLastToken(node)
				nextToken := sc.GetTokenAfter(node)

				// Don't fix if there are comments inside the brackets.
				if commentsExistBetween(sc, leftBracket, rightBracket) {
					return nil
				}

				var fixes []*eslint.Fix

				// Replace the brackets by an identifier.
				if !eslint.GetBool(node, "optional") {
					text := "."
					if eslint.IsDecimalInteger(eslint.GetNode(node, "object")) {
						text = " ."
					}
					fixes = append(fixes, f.InsertTextBefore(leftBracket, text))
				}
				fixes = append(fixes, f.ReplaceTextRange(
					[2]int{eslint.Start(leftBracket), eslint.End(rightBracket)}, strValue))

				// Insert a space after the property if it would merge with the
				// next token.
				if nextToken != nil &&
					eslint.End(rightBracket) == eslint.Start(nextToken) &&
					!canTokensBeAdjacent(strValue, nextToken) {
					fixes = append(fixes, f.InsertTextAfter(node, " "))
				}

				return mergeFixes(sc, f, fixes)
			},
		})
	}

	isStaticTemplateLiteral := func(node eslint.Node) bool {
		return eslint.NodeType(node) == "TemplateLiteral" && len(eslint.GetNodes(node, "expressions")) == 0
	}

	return map[string]func(eslint.Node){
		"MemberExpression": func(node eslint.Node) {
			prop := eslint.GetNode(node, "property")
			computed := eslint.GetBool(node, "computed")

			if computed && eslint.NodeType(prop) == "Literal" {
				v := eslint.Get(prop, "value")
				if literalTypesToCheck[jsTypeOf(v)] || eslint.IsNullLiteral(prop) {
					checkComputedProperty(node, v, true)
				}
			}

			if computed && isStaticTemplateLiteral(prop) {
				quasis := eslint.GetNodes(prop, "quasis")
				var cooked any
				if len(quasis) > 0 {
					cooked = eslint.GetMap(quasis[0], "value")["cooked"]
				}
				checkComputedProperty(node, cooked, false)
			}

			if !allowKeywords && !computed &&
				eslint.NodeType(prop) == "Identifier" &&
				jsKeywords[eslint.GetString(prop, "name")] {
				ctx.Report(eslint.Report{
					Node:      prop,
					MessageID: "useBrackets",
					Data:      map[string]any{"key": eslint.GetString(prop, "name")},
					Fix: func(f *eslint.Fixer) *eslint.Fix {
						dotToken := sc.GetTokenBefore(prop)

						/*
						 * A statement that starts with `let[` is parsed as a
						 * destructuring variable declaration, not a
						 * MemberExpression, so the dot cannot be removed.
						 */
						obj := eslint.GetNode(node, "object")
						if eslint.NodeType(obj) == "Identifier" &&
							eslint.GetString(obj, "name") == "let" &&
							!eslint.GetBool(node, "optional") {
							return nil
						}

						// Don't fix if there are comments between the dot and
						// the property name.
						if commentsExistBetween(sc, dotToken, prop) {
							return nil
						}

						var fixes []*eslint.Fix
						if !eslint.GetBool(node, "optional") {
							fixes = append(fixes, f.Remove(dotToken))
						}
						fixes = append(fixes, f.ReplaceText(prop, `["`+eslint.GetString(prop, "name")+`"]`))
						return mergeFixes(sc, f, fixes)
					},
				})
			}
		},
	}
}

// literalTypesToCheck is the rule's `new Set(["string", "boolean"])`.
var literalTypesToCheck = map[string]bool{"string": true, "boolean": true}

// jsTypeOf is the JS `typeof` for the values a Literal property can hold.
func jsTypeOf(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64, int:
		return "number"
	case nil:
		return "object"
	}
	return "object"
}

// jsString is JS String(value) for the property values this rule sees.
func jsString(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return trimFloat(t)
	case int:
		return fmt.Sprintf("%d", t)
	}
	return ""
}

func trimFloat(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", f), "0"), ".")
}

// jsonStringify mirrors JSON.stringify for the Literal values the rule passes
// to the message data: a string, a boolean or null.
func jsonStringify(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case string:
		var b strings.Builder
		b.WriteByte('"')
		for _, r := range t {
			switch r {
			case '"':
				b.WriteString(`\"`)
			case '\\':
				b.WriteString(`\\`)
			case '\n':
				b.WriteString(`\n`)
			case '\r':
				b.WriteString(`\r`)
			case '\t':
				b.WriteString(`\t`)
			case '\b':
				b.WriteString(`\b`)
			case '\f':
				b.WriteString(`\f`)
			default:
				if r < 0x20 {
					fmt.Fprintf(&b, `\u%04x`, r)
				} else {
					b.WriteRune(r)
				}
			}
		}
		b.WriteByte('"')
		return b.String()
	}
	return "null"
}

// jsRegexpToGo rewrites the JS-only regexp escapes a user pattern may contain
// into their Go equivalents (Go rejects `\uXXXX`).
var jsUnicodeEscape = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)

func jsRegexpToGo(pattern string) string {
	return jsUnicodeEscape.ReplaceAllString(pattern, `\x{$1}`)
}

// commentsExistBetween mirrors SourceCode#commentsExistBetween: is there a
// comment that starts at or after the end of left and ends at or before the
// start of right?
func commentsExistBetween(sc *eslint.SourceCode, left, right eslint.Node) bool {
	if left == nil || right == nil {
		return false
	}
	for _, c := range sc.GetAllComments() {
		if eslint.Start(c) >= eslint.End(left) {
			return eslint.End(c) <= eslint.Start(right)
		}
	}
	return false
}

// mergeFixes ports report-translator's mergeFixes: several fixes yielded by a
// generator fix become one fix spanning them all, with the source text between
// them preserved. Fixes must not overlap.
func mergeFixes(sc *eslint.SourceCode, f *eslint.Fixer, fixes []*eslint.Fix) *eslint.Fix {
	if len(fixes) == 0 {
		return nil
	}
	if len(fixes) == 1 {
		return fixes[0]
	}

	sorted := make([]*eslint.Fix, len(fixes))
	copy(sorted, fixes)
	// compareFixesByRange: by start, then by end.
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0; j-- {
			a, b := sorted[j-1], sorted[j]
			if a.Range[0] < b.Range[0] || (a.Range[0] == b.Range[0] && a.Range[1] <= b.Range[1]) {
				break
			}
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}

	originalText := sc.Text()
	start := sorted[0].Range[0]
	end := sorted[len(sorted)-1].Range[1]
	text := ""
	lastPos := -1 << 62

	for _, fix := range sorted {
		if fix.Range[0] >= 0 {
			from := maxInt(0, maxInt(start, lastPos))
			if from < fix.Range[0] {
				text += originalText[from:fix.Range[0]]
			}
		}
		text += fix.Text
		lastPos = fix.Range[1]
	}
	from := maxInt(0, maxInt(start, lastPos))
	if from < end {
		text += originalText[from:end]
	}

	return f.ReplaceTextRange([2]int{start, end}, text)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// canTokensBeAdjacent ports astUtils.canTokensBeAdjacent for the shape this
// rule uses it in: a string left value and a real token on the right. Only the
// last token of the left string matters, and for the identifier-shaped values
// passed here that token is never a punctuator, string, template or numeric
// token, so the outcome is decided by the right token.
func canTokensBeAdjacent(leftValue string, rightToken eslint.Node) bool {
	if rightToken == nil {
		return false
	}
	leftType := identifierTokenType(leftValue)
	rightType := eslint.NodeType(rightToken)

	if leftType == eslint.TokenPunctuator || rightType == eslint.TokenPunctuator {
		if leftType == eslint.TokenPunctuator && rightType == eslint.TokenPunctuator {
			plus := map[string]bool{"+": true, "++": true}
			minus := map[string]bool{"-": true, "--": true}
			rv := eslint.TokenValue(rightToken)
			return !((plus[leftValue] && plus[rv]) || (minus[leftValue] && minus[rv]))
		}
		if leftType == eslint.TokenPunctuator && leftValue == "/" {
			switch rightType {
			case "Block", "Line", eslint.TokenRegularExpression:
				return false
			}
			return true
		}
		return true
	}

	if leftType == eslint.TokenString || rightType == eslint.TokenString ||
		leftType == eslint.TokenTemplate || rightType == eslint.TokenTemplate {
		return true
	}

	if leftType != eslint.TokenNumeric && rightType == eslint.TokenNumeric &&
		strings.HasPrefix(eslint.TokenValue(rightToken), ".") {
		return true
	}

	if leftType == "Block" || rightType == "Block" || rightType == "Line" {
		return true
	}

	if rightType == eslint.TokenPrivateIdent {
		return true
	}

	return false
}

// identifierTokenType reports the espree token type the given source text
// tokenizes to. Callers only ever pass an identifier-shaped string.
func identifierTokenType(text string) string {
	switch text {
	case "true", "false":
		return eslint.TokenBoolean
	case "null":
		return eslint.TokenNull
	}
	if jsKeywords[text] {
		return eslint.TokenKeyword
	}
	return eslint.TokenIdentifier
}

// jsKeywords is eslint's rules/utils/keywords.js list (the ES3 keywords).
var jsKeywords = map[string]bool{
	"abstract": true, "boolean": true, "break": true, "byte": true, "case": true,
	"catch": true, "char": true, "class": true, "const": true, "continue": true,
	"debugger": true, "default": true, "delete": true, "do": true, "double": true,
	"else": true, "enum": true, "export": true, "extends": true, "false": true,
	"final": true, "finally": true, "float": true, "for": true, "function": true,
	"goto": true, "if": true, "implements": true, "import": true, "in": true,
	"instanceof": true, "int": true, "interface": true, "long": true, "native": true,
	"new": true, "null": true, "package": true, "private": true, "protected": true,
	"public": true, "return": true, "short": true, "static": true, "super": true,
	"switch": true, "synchronized": true, "this": true, "throw": true, "throws": true,
	"transient": true, "true": true, "try": true, "typeof": true, "var": true,
	"void": true, "volatile": true, "while": true, "with": true,
}
