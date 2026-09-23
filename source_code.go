package eslint

import (
	"regexp"
	"strings"

	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// source_code.go — ESLint's SourceCode. It is the object every rule receives:
// the text, the AST, the token/comment streams (via the cursor queries in
// token_store.go), line bookkeeping, and the scope manager.

// SourceCode mirrors eslint/lib/source-code/source-code.js for the surface the
// ported rules use.
type SourceCode struct {
	text         string
	ast          Node
	tokens       []Node
	comments     []Node
	store        *tokenList
	lineIndex    *LineIndex
	units        *codeUnitTable
	scopeManager *eslintscope.ScopeManager
	hasBOM       bool
	scopeCache   map[uintptr]*eslintscope.Scope
}

// NewSourceCode builds a SourceCode. text must already have any BOM stripped
// (Linter does that, as ESLint does) — hasBOM only records that it was there.
func NewSourceCode(text string, ast Node, tokens, comments []Node, hasBOM bool) *SourceCode {
	if tokens == nil {
		tokens = []Node{}
	}
	if comments == nil {
		comments = []Node{}
	}
	return &SourceCode{
		text:       text,
		ast:        ast,
		tokens:     tokens,
		comments:   comments,
		store:      newTokenList(tokens, comments),
		lineIndex:  NewLineIndex(text),
		units:      newCodeUnitTable(text),
		hasBOM:     hasBOM,
		scopeCache: map[uintptr]*eslintscope.Scope{},
	}
}

// Text returns the source text (BOM-stripped).
func (s *SourceCode) Text() string { return s.text }

// AST returns the Program node.
func (s *SourceCode) AST() Node { return s.ast }

// HasBOM reports whether the original input began with a byte order mark.
func (s *SourceCode) HasBOM() bool { return s.hasBOM }

// Tokens returns the code tokens (comments excluded).
func (s *SourceCode) Tokens() []Node { return s.tokens }

// Comments returns the comment tokens.
func (s *SourceCode) Comments() []Node { return s.comments }

// AllTokens returns tokens and comments merged in source order.
func (s *SourceCode) AllTokens() []Node { return s.store.all }

// Lines returns the source split into lines (SourceCode.lines).
func (s *SourceCode) Lines() []string { return s.lineIndex.Lines() }

// LineIndex exposes the offset ↔ line/column mapping.
func (s *SourceCode) LineIndex() *LineIndex { return s.lineIndex }

// GetText returns the source text of a node (or the whole text for nil).
func (s *SourceCode) GetText(node Node) string {
	if node == nil {
		return s.text
	}
	return s.GetTextRange(MustRange(node))
}

// GetTextRange returns the text covered by an offset range, clamped.
func (s *SourceCode) GetTextRange(r [2]int) string {
	start, end := r[0], r[1]
	if start < 0 {
		start = 0
	}
	if end > len(s.text) {
		end = len(s.text)
	}
	if start > end {
		return ""
	}
	return s.text[start:end]
}

// GetTextBetween returns the text between two nodes/tokens.
func (s *SourceCode) GetTextBetween(a, b Node) string {
	return s.GetTextRange([2]int{End(a), Start(b)})
}

// ---- token queries (the cursor API) ----

// GetFirstToken returns the first token of a node.
func (s *SourceCode) GetFirstToken(node Node, opts ...TokenOpt) Node {
	return s.store.getFirstToken(node, firstOpt(opts))
}

// GetLastToken returns the last token of a node.
func (s *SourceCode) GetLastToken(node Node, opts ...TokenOpt) Node {
	return s.store.getLastToken(node, firstOpt(opts))
}

// GetTokenBefore returns the token immediately before a node/token.
func (s *SourceCode) GetTokenBefore(node Node, opts ...TokenOpt) Node {
	return s.store.getTokenBefore(node, firstOpt(opts))
}

// GetTokenAfter returns the token immediately after a node/token.
func (s *SourceCode) GetTokenAfter(node Node, opts ...TokenOpt) Node {
	return s.store.getTokenAfter(node, firstOpt(opts))
}

// GetFirstTokens returns the first n tokens of a node.
func (s *SourceCode) GetFirstTokens(node Node, count int, opts ...TokenOpt) []Node {
	return s.store.getFirstTokens(node, count, firstOpt(opts))
}

// GetLastTokens returns the last n tokens of a node.
func (s *SourceCode) GetLastTokens(node Node, count int, opts ...TokenOpt) []Node {
	return s.store.getLastTokens(node, count, firstOpt(opts))
}

// GetTokens returns all tokens of a node.
func (s *SourceCode) GetTokens(node Node, opts ...TokenOpt) []Node {
	return s.store.getTokens(node, firstOpt(opts))
}

// GetTokensBetween returns the tokens between two nodes/tokens.
func (s *SourceCode) GetTokensBetween(left, right Node, opts ...TokenOpt) []Node {
	return s.store.getTokensBetween(left, right, firstOpt(opts))
}

// GetCommentsBefore returns comments ending at or before the node.
func (s *SourceCode) GetCommentsBefore(node Node) []Node { return s.store.commentsBefore(node) }

// GetCommentsAfter returns comments starting at or after the node.
func (s *SourceCode) GetCommentsAfter(node Node) []Node { return s.store.commentsAfter(node) }

// GetCommentsInside returns comments inside the node's range.
func (s *SourceCode) GetCommentsInside(node Node) []Node { return s.store.commentsInside(node) }

// GetAllComments returns every comment.
func (s *SourceCode) GetAllComments() []Node { return s.comments }

// GetNodeByRangeIndex returns the innermost node containing an offset
// (SourceCode.getNodeByRangeIndex). It is used by fixers that need the syntax
// node owning a token.
func (s *SourceCode) GetNodeByRangeIndex(index int) Node {
	var result Node
	var walk func(node Node)
	walk = func(node Node) {
		if node == nil {
			return
		}
		start, end := Start(node), End(node)
		if start <= index && index < end {
			result = node
		} else {
			return
		}
		for _, key := range VisitorKeys[NodeType(node)] {
			switch child := node[key].(type) {
			case []any:
				for _, el := range child {
					walk(asNode(el))
				}
			case Node:
				walk(child)
			}
		}
	}
	walk(s.ast)
	return result
}

// GetIndexFromLoc converts a {line,column} point to a byte offset.
func (s *SourceCode) GetIndexFromLoc(line, col int) int { return s.lineIndex.Index(line, col) }

// GetLocFromIndex converts a byte offset to 1-based line / 0-based column.
func (s *SourceCode) GetLocFromIndex(offset int) (line, col int) { return s.lineIndex.Loc(offset) }

var (
	blockCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/`)
	lineCommentRe  = regexp.MustCompile(`(?m)//.*$`)
	whitespaceRe   = regexp.MustCompile(`\s`)
)

// IsSpaceBetween reports whether only whitespace (ignoring comments) separates
// two nodes/tokens — SourceCode.isSpaceBetweenTokens.
func (s *SourceCode) IsSpaceBetween(first, second Node) bool {
	if End(first) > Start(second) {
		return false
	}
	text := s.GetTextBetween(first, second)
	text = blockCommentRe.ReplaceAllString(text, "")
	text = lineCommentRe.ReplaceAllString(text, "")
	return whitespaceRe.MatchString(text)
}

// IsSpaceBetweenTokens is the token-flavoured alias ESLint exposes separately.
func (s *SourceCode) IsSpaceBetweenTokens(first, second Node) bool {
	if NodeType(first) == "JSXText" || NodeType(second) == "JSXText" {
		return false
	}
	return s.IsSpaceBetween(first, second)
}

// BetweenTokensContain only whitespace reports whether the raw text between two
// positions contains no non-whitespace characters.
func (s *SourceCode) BetweenTokensContain(a, b Node) string { return s.GetTextBetween(a, b) }

func firstOpt(opts []TokenOpt) TokenOpt {
	if len(opts) == 0 {
		return TokenOpt{}
	}
	return opts[0]
}

// TrimmedText is a small helper for rules that compare raw source (e.g. quotes).
func (s *SourceCode) TrimmedText(node Node) string { return strings.TrimSpace(s.GetText(node)) }
