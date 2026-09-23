package quotes

import (
	"regexp"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "quotes"

// Rule is a port of eslint/lib/rules/quotes.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "layout",
		Docs: eslint.RuleDocs{
			Description: "Enforce the consistent use of either backticks, double, or single quotes",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/quotes",
		},
		Fixable: "code",
		Schema: []any{
			map[string]any{
				"enum": []any{"single", "double", "backtick"},
			},
			map[string]any{
				"anyOf": []any{
					map[string]any{"enum": []any{"avoid-escape"}},
					map[string]any{
						"type": "object",
						"properties": map[string]any{
							"avoidEscape":           map[string]any{"type": "boolean"},
							"allowTemplateLiterals": map[string]any{"type": "boolean"},
						},
						"additionalProperties": false,
					},
				},
			},
		},
		Messages: map[string]string{
			"wrongQuotes": "Strings must use {{description}}.",
		},
	},
	Create: create,
}

// quoteSetting is one entry of QUOTE_SETTINGS.
type quoteSetting struct {
	quote          string
	alternateQuote string
	description    string
}

var quoteSettings = map[string]quoteSetting{
	"double":   {quote: "\"", alternateQuote: "'", description: "doublequote"},
	"single":   {quote: "'", alternateQuote: "\"", description: "singlequote"},
	"backtick": {quote: "`", alternateQuote: "\"", description: "backtick"},
}

// avoidEscape is the deprecated string form of the second option.
const avoidEscapeOption = "avoid-escape"

// unescapedLinebreakRe is the rule's UNESCAPED_LINEBREAK_PATTERN: a newline
// preceded by an even number of backslashes (i.e. not escaped).
var unescapedLinebreakRe = regexp.MustCompile(`(?:^|[^\\])(?:\\\\)*[\r\n\x{2028}\x{2029}]`)

// convertRe is the rule's convert() regexp:
// /\\(\$\{|\r\n?|\n|.)|["'`]|\$\{|(\r\n?|\n)/gu
var convertRe = regexp.MustCompile("\\\\(\\$\\{|\\r\\n?|\\n|.)|[\"'`]|\\$\\{|(\\r\\n?|\\n)")

// octalOrNonOctalDecimalEscapeRe is astUtils'
// OCTAL_OR_NON_OCTAL_DECIMAL_ESCAPE_PATTERN: /^(?:[^\\]|\\.)*\\(?:[1-9]|0[0-9])/su
var octalOrNonOctalDecimalEscapeRe = regexp.MustCompile(`(?s)^(?:[^\\]|\\.)*\\(?:[1-9]|0[0-9])`)

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	quoteOption := ctx.OptionString(0, "")
	settings, settingsOK := quoteSettings[quoteOption]
	if !settingsOK {
		settings = quoteSettings["double"]
		settingsOK = true
	}

	allowTemplateLiterals := false
	avoidEscape := false
	if opts, ok := ctx.Option(1).(map[string]any); ok {
		allowTemplateLiterals = opts["allowTemplateLiterals"] == true
		avoidEscape = opts["avoidEscape"] == true
	}
	// Deprecated string form.
	if s, ok := ctx.Option(1).(string); ok && s == avoidEscapeOption {
		avoidEscape = true
	}

	return map[string]func(eslint.Node){
		"Literal": func(node eslint.Node) {
			rawVal := eslint.GetString(node, "raw")
			val := eslint.Get(node, "value")

			// `typeof node.value === "string"` — a bigint literal's Go value is
			// also a string, so it has to be excluded explicitly.
			if _, isString := val.(string); !isString || isBigintLiteral(node) {
				return
			}

			isValid := (quoteOption == "backtick" && isAllowedAsNonBacktick(sc, node)) ||
				isJSXLiteral(node) ||
				isSurroundedBy(rawVal, settings.quote)

			if !isValid && avoidEscape {
				isValid = isSurroundedBy(rawVal, settings.alternateQuote) &&
					strings.Contains(rawVal, settings.quote)
			}

			if isValid {
				return
			}

			setting := settings
			raw := rawVal
			ctx.Report(eslint.Report{
				Node:      node,
				MessageID: "wrongQuotes",
				Data:      map[string]any{"description": setting.description},
				Fix: func(f *eslint.Fixer) *eslint.Fix {
					if quoteOption == "backtick" && hasOctalOrNonOctalDecimalEscapeSequence(raw) {
						// An octal or non-octal decimal escape sequence in a
						// template literal would be a syntax error.
						return nil
					}
					return f.ReplaceText(node, convert(setting, raw))
				},
			})
		},

		"TemplateLiteral": func(node eslint.Node) {
			// Don't report if backticks are expected or a template literal
			// feature is in use.
			if allowTemplateLiterals || quoteOption == "backtick" || isUsingFeatureOfTemplateLiteral(node) {
				return
			}

			setting := settings
			ctx.Report(eslint.Report{
				Node:      node,
				MessageID: "wrongQuotes",
				Data:      map[string]any{"description": setting.description},
				Fix: func(f *eslint.Fixer) *eslint.Fix {
					if isTopLevelExpressionStatement(eslint.Parent(node)) && !isParenthesised(sc, node) {
						// TemplateLiterals aren't actually directives, but
						// fixing them might turn them into directives and
						// change the behaviour of the code.
						return nil
					}
					return f.ReplaceText(node, convert(setting, sc.GetText(node)))
				},
			})
		},
	}
}

// isBigintLiteral reports whether a Literal is a bigint (JS `typeof value` is
// "bigint", never "string").
func isBigintLiteral(node eslint.Node) bool {
	_, ok := node["bigint"]
	return ok
}

// convert switches quoting between ' " and `, escaping and unescaping as
// necessary (the rule's QUOTE_SETTINGS.*.convert).
func convert(setting quoteSetting, str string) string {
	newQuote := setting.quote
	if str == "" {
		return str
	}
	oldQuote := str[:1]
	if newQuote == oldQuote {
		return str
	}
	if len(str) < 2 {
		return str
	}
	return newQuote + replaceConvert(str[1:len(str)-1], oldQuote, newQuote) + newQuote
}

// replaceConvert is the body of convert()'s String.prototype.replace callback.
func replaceConvert(s, oldQuote, newQuote string) string {
	matches := convertRe.FindAllStringSubmatchIndex(s, -1)
	if matches == nil {
		return s
	}
	var sb strings.Builder
	last := 0
	for _, m := range matches {
		sb.WriteString(s[last:m[0]])
		match := s[m[0]:m[1]]
		escaped := ""
		hasEscaped := m[2] >= 0
		if hasEscaped {
			escaped = s[m[2]:m[3]]
		}
		newline := ""
		if m[4] >= 0 {
			newline = s[m[4]:m[5]]
		}
		switch {
		case hasEscaped && (escaped == oldQuote || (oldQuote == "`" && escaped == "${")):
			// Unescape.
			sb.WriteString(escaped)
		case match == newQuote || (newQuote == "`" && match == "${"):
			// Escape.
			sb.WriteString("\\" + match)
		case newline != "" && oldQuote == "`":
			// Escape newlines.
			sb.WriteString("\\n")
		default:
			sb.WriteString(match)
		}
		last = m[1]
	}
	sb.WriteString(s[last:])
	return sb.String()
}

// isSurroundedBy is astUtils.isSurroundedBy.
func isSurroundedBy(val, character string) bool {
	return len(val) >= 1 && strings.HasPrefix(val, character) && strings.HasSuffix(val, character)
}

// hasOctalOrNonOctalDecimalEscapeSequence is
// astUtils.hasOctalOrNonOctalDecimalEscapeSequence.
func hasOctalOrNonOctalDecimalEscapeSequence(rawString string) bool {
	return octalOrNonOctalDecimalEscapeRe.MatchString(rawString)
}

// isJSXLiteral is the rule's isJSXLiteral. The Go parser has no JSX support,
// so no node can ever be part of JSX; the check is kept for fidelity.
func isJSXLiteral(node eslint.Node) bool {
	switch eslint.NodeType(eslint.Parent(node)) {
	case "JSXAttribute", "JSXElement", "JSXFragment":
		return true
	}
	return false
}

// isParenthesised is astUtils.isParenthesised (the single-pair check the rule
// uses, which is not the same as the core's multi-paren IsParenthesized).
func isParenthesised(sc *eslint.SourceCode, node eslint.Node) bool {
	previousToken := sc.GetTokenBefore(node)
	nextToken := sc.GetTokenAfter(node)
	return previousToken != nil && nextToken != nil &&
		eslint.TokenValue(previousToken) == "(" &&
		eslint.End(previousToken) <= eslint.Start(node) &&
		eslint.TokenValue(nextToken) == ")" &&
		eslint.Start(nextToken) >= eslint.End(node)
}

// isTopLevelExpressionStatement is astUtils.isTopLevelExpressionStatement.
func isTopLevelExpressionStatement(node eslint.Node) bool {
	if eslint.NodeType(node) != "ExpressionStatement" {
		return false
	}
	parent := eslint.Parent(node)
	return eslint.NodeType(parent) == "Program" ||
		(eslint.NodeType(parent) == "BlockStatement" && eslint.IsFunction(eslint.Parent(parent)))
}

// isDirective is the rule's isDirective.
func isDirective(sc *eslint.SourceCode, node eslint.Node) bool {
	if eslint.NodeType(node) != "ExpressionStatement" {
		return false
	}
	expression := eslint.GetNode(node, "expression")
	if eslint.NodeType(expression) != "Literal" || isBigintLiteral(expression) {
		return false
	}
	if _, ok := eslint.Get(expression, "value").(string); !ok {
		return false
	}
	return !isParenthesised(sc, expression)
}

// isExpressionInOrJustAfterDirectivePrologue is the rule's
// isExpressionInOrJustAfterDirectivePrologue.
func isExpressionInOrJustAfterDirectivePrologue(sc *eslint.SourceCode, node eslint.Node) bool {
	parent := eslint.Parent(node)
	if !isTopLevelExpressionStatement(parent) {
		return false
	}
	block := eslint.Parent(parent)

	for _, statement := range eslint.GetNodes(block, "body") {
		if eslint.SameNode(statement, parent) {
			return true
		}
		if !isDirective(sc, statement) {
			break
		}
	}
	return false
}

// isAllowedAsNonBacktick is the rule's isAllowedAsNonBacktick.
func isAllowedAsNonBacktick(sc *eslint.SourceCode, node eslint.Node) bool {
	parent := eslint.Parent(node)
	switch eslint.NodeType(parent) {
	case "ExpressionStatement":
		return !isParenthesised(sc, node) &&
			isExpressionInOrJustAfterDirectivePrologue(sc, node)

	case "Property", "PropertyDefinition", "MethodDefinition":
		return eslint.SameNode(eslint.GetNode(parent, "key"), node) &&
			!eslint.GetBool(parent, "computed")

	case "ImportDeclaration", "ExportNamedDeclaration":
		return eslint.SameNode(eslint.GetNode(parent, "source"), node)

	case "ExportAllDeclaration":
		return eslint.SameNode(eslint.GetNode(parent, "exported"), node) ||
			eslint.SameNode(eslint.GetNode(parent, "source"), node)

	case "ImportSpecifier":
		return eslint.SameNode(eslint.GetNode(parent, "imported"), node)

	case "ExportSpecifier":
		return eslint.SameNode(eslint.GetNode(parent, "local"), node) ||
			eslint.SameNode(eslint.GetNode(parent, "exported"), node)
	}
	return false
}

// isUsingFeatureOfTemplateLiteral is the rule's
// isUsingFeatureOfTemplateLiteral.
func isUsingFeatureOfTemplateLiteral(node eslint.Node) bool {
	parent := eslint.Parent(node)
	if eslint.NodeType(parent) == "TaggedTemplateExpression" &&
		eslint.SameNode(node, eslint.GetNode(parent, "quasi")) {
		return true
	}

	if len(eslint.GetNodes(node, "expressions")) > 0 {
		return true
	}

	quasis := eslint.GetNodes(node, "quasis")
	if len(quasis) >= 1 {
		raw := ""
		if value := eslint.GetMap(quasis[0], "value"); value != nil {
			raw, _ = value["raw"].(string)
		}
		if unescapedLinebreakRe.MatchString(raw) {
			return true
		}
	}

	return false
}
