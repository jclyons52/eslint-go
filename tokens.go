package eslint

// tokens.go — the small token predicates ESLint's rules use constantly
// (eslint/lib/rules/utils/ast-utils.js + eslint-utils). Values match the
// esprima/espree token vocabulary espree-go emits.

// Token type names (espree/esprima).
const (
	TokenBoolean           = "Boolean"
	TokenIdentifier        = "Identifier"
	TokenKeyword           = "Keyword"
	TokenNull              = "Null"
	TokenNumeric           = "Numeric"
	TokenPunctuator        = "Punctuator"
	TokenString            = "String"
	TokenRegularExpression = "RegularExpression"
	TokenTemplate          = "Template"
	TokenPrivateIdent      = "PrivateIdentifier"
)

// Comment type names.
const (
	CommentLine    = "Line"
	CommentBlock   = "Block"
	CommentShebang = "Shebang"
)

// IsCommentToken reports whether a token-like node is a comment.
func IsCommentToken(n Node) bool {
	switch NodeType(n) {
	case CommentLine, CommentBlock, CommentShebang:
		return true
	}
	return false
}

// TokenValue returns a token's source text (its `value`).
func TokenValue(n Node) string { return GetString(n, "value") }

// IsTokenWithType reports whether n is a token of the given esprima type.
func IsTokenWithType(n Node, typ string) bool { return NodeType(n) == typ }

// IsPunctuatorToken reports whether n is a punctuator; when value is non-empty
// it must match exactly.
func IsPunctuatorToken(n Node, value string) bool {
	if NodeType(n) != TokenPunctuator {
		return false
	}
	return value == "" || TokenValue(n) == value
}

// IsKeywordToken reports whether the token is a JS keyword token.
func IsKeywordToken(n Node) bool { return IsTokenWithType(n, TokenKeyword) }

// IsIdentifierToken reports whether the token is an identifier token.
func IsIdentifierToken(n Node) bool { return IsTokenWithType(n, TokenIdentifier) }

// IsStringToken reports whether the token is a string literal token.
func IsStringToken(n Node) bool { return IsTokenWithType(n, TokenString) }

// IsNumericToken reports whether the token is a numeric literal token.
func IsNumericToken(n Node) bool { return IsTokenWithType(n, TokenNumeric) }

// IsNotToken reports whether the token is the `!` punctuator.
func IsNotToken(n Node) bool { return IsPunctuatorToken(n, "!") }

// IsCommaToken reports whether the token is a comma.
func IsCommaToken(n Node) bool { return IsPunctuatorToken(n, ",") }

// IsSemicolonToken reports whether the token is a semicolon.
func IsSemicolonToken(n Node) bool { return IsPunctuatorToken(n, ";") }

// IsOpeningBraceToken reports whether the token is `{`.
func IsOpeningBraceToken(n Node) bool { return IsPunctuatorToken(n, "{") }

// IsClosingBraceToken reports whether the token is `}`.
func IsClosingBraceToken(n Node) bool { return IsPunctuatorToken(n, "}") }

// IsOpeningParenToken reports whether the token is `(`.
func IsOpeningParenToken(n Node) bool { return IsPunctuatorToken(n, "(") }

// IsClosingParenToken reports whether the token is `)`.
func IsClosingParenToken(n Node) bool { return IsPunctuatorToken(n, ")") }

// IsOpeningBracketToken reports whether the token is `[`.
func IsOpeningBracketToken(n Node) bool { return IsPunctuatorToken(n, "[") }

// IsClosingBracketToken reports whether the token is `]`.
func IsClosingBracketToken(n Node) bool { return IsPunctuatorToken(n, "]") }

// IsDotToken reports whether the token is `.`.
func IsDotToken(n Node) bool { return IsPunctuatorToken(n, ".") }

// IsStarToken reports whether the token is `*`.
func IsStarToken(n Node) bool { return IsPunctuatorToken(n, "*") }

// IsColonToken reports whether the token is `:`.
func IsColonToken(n Node) bool { return IsPunctuatorToken(n, ":") }

// IsArrowToken reports whether the token is `=>`.
func IsArrowToken(n Node) bool { return IsPunctuatorToken(n, "=>") }

// TokenEndsStatement reports whether a token can terminate a statement, which
// is how ASI-sensitive rules decide whether a semicolon is required.
func TokenEndsStatement(n Node) bool {
	switch TokenValue(n) {
	case ")", "]", "}", "++", "--":
		return true
	}
	switch NodeType(n) {
	case TokenIdentifier, TokenString, TokenNumeric, TokenBoolean, TokenNull,
		TokenRegularExpression, TokenTemplate, TokenPrivateIdent:
		return true
	}
	if NodeType(n) == TokenKeyword {
		switch TokenValue(n) {
		case "this", "super", "true", "false", "null":
			return true
		}
	}
	return false
}
