package eslint

import "sort"

// token_store.go — the SourceCode token API ("cursors" in ESLint's
// source-code/token-store).
//
// ESLint exposes the token stream through cursor queries (getFirstToken,
// getTokenBefore, …) with {includeComments, skip, filter} options. This is the
// Go equivalent: one merged, offset-sorted list of tokens+comments with
// boundary searches, plus the options struct rules use.

// TokenOpt mirrors the options object accepted by the token query methods.
type TokenOpt struct {
	// IncludeComments makes comments visible to the query.
	IncludeComments bool
	// Skip moves past this many matching tokens before returning one.
	Skip int
	// Filter, when set, restricts which tokens count as matches.
	Filter func(Node) bool
}

func (o TokenOpt) matches(n Node) bool {
	if o.Filter == nil {
		return true
	}
	return o.Filter(n)
}

// tokenList is the merged, offset-sorted token+comment stream.
type tokenList struct {
	all      []Node // tokens and comments, sorted by start offset
	tokens   []Node
	comments []Node
}

func newTokenList(tokens, comments []Node) *tokenList {
	all := make([]Node, 0, len(tokens)+len(comments))
	all = append(all, tokens...)
	all = append(all, comments...)
	sort.SliceStable(all, func(i, j int) bool { return Start(all[i]) < Start(all[j]) })
	return &tokenList{all: all, tokens: tokens, comments: comments}
}

// stream returns the entries a query may see.
func (t *tokenList) stream(includeComments bool) []Node {
	if includeComments {
		return t.all
	}
	return t.tokens
}

// firstAtOrAfter returns the index of the first entry whose start is >= off.
func firstAtOrAfter(list []Node, off int) int {
	return sort.Search(len(list), func(i int) bool { return Start(list[i]) >= off })
}

// tokensInside returns the tokens contained in [start, end] on both edges.
func (t *tokenList) tokensInside(list []Node, start, end int, opt TokenOpt) []Node {
	var out []Node
	for i := firstAtOrAfter(list, start); i < len(list); i++ {
		item := list[i]
		if Start(item) > end {
			break
		}
		if End(item) > end {
			continue
		}
		if !opt.matches(item) {
			continue
		}
		out = append(out, item)
	}
	return out
}

// getTokens returns every visible token inside a node's range.
func (t *tokenList) getTokens(node Node, opt TokenOpt) []Node {
	list := t.stream(opt.IncludeComments)
	r := MustRange(node)
	return t.tokensInside(list, r[0], r[1], TokenOpt{Filter: opt.Filter})
}

// getFirstTokens returns the first count visible tokens inside a node.
func (t *tokenList) getFirstTokens(node Node, count int, opt TokenOpt) []Node {
	all := t.getTokens(node, opt)
	if count < 0 {
		count = 0
	}
	if count > len(all) {
		count = len(all)
	}
	return all[:count]
}

// getLastTokens returns the last count visible tokens inside a node.
func (t *tokenList) getLastTokens(node Node, count int, opt TokenOpt) []Node {
	all := t.getTokens(node, opt)
	if count < 0 {
		count = 0
	}
	if count > len(all) {
		count = len(all)
	}
	return all[len(all)-count:]
}

// getFirstToken returns the first visible token inside a node, honouring skip.
func (t *tokenList) getFirstToken(node Node, opt TokenOpt) Node {
	all := t.getTokens(node, opt)
	if opt.Skip >= len(all) {
		return nil
	}
	return all[opt.Skip]
}

// getLastToken returns the last visible token inside a node, honouring skip.
func (t *tokenList) getLastToken(node Node, opt TokenOpt) Node {
	all := t.getTokens(node, opt)
	if opt.Skip >= len(all) {
		return nil
	}
	return all[len(all)-1-opt.Skip]
}

// getTokenBefore returns the visible token immediately before a node/token,
// ignoring entries that overlap it.
func (t *tokenList) getTokenBefore(node Node, opt TokenOpt) Node {
	list := t.stream(opt.IncludeComments)
	boundary := Start(node)
	i := firstAtOrAfter(list, boundary)
	found := 0
	for j := i - 1; j >= 0; j-- {
		item := list[j]
		if End(item) > boundary {
			continue
		}
		if !opt.matches(item) {
			continue
		}
		if found == opt.Skip {
			return item
		}
		found++
	}
	return nil
}

// getTokenAfter returns the visible token immediately after a node/token.
func (t *tokenList) getTokenAfter(node Node, opt TokenOpt) Node {
	list := t.stream(opt.IncludeComments)
	boundary := End(node)
	found := 0
	for i := firstAtOrAfter(list, boundary); i < len(list); i++ {
		item := list[i]
		if Start(item) < boundary {
			continue
		}
		if !opt.matches(item) {
			continue
		}
		if found == opt.Skip {
			return item
		}
		found++
	}
	return nil
}

// getTokensBetween returns the visible tokens between two nodes/tokens.
func (t *tokenList) getTokensBetween(left, right Node, opt TokenOpt) []Node {
	list := t.stream(opt.IncludeComments)
	if right == nil {
		return nil
	}
	return t.tokensInside(list, End(left), Start(right), opt)
}

// commentsBefore / commentsAfter / commentsInside mirror the comment-specific
// queries (comments are always visible to these).
func (t *tokenList) commentsBefore(node Node) []Node {
	boundary := Start(node)
	var out []Node
	for _, c := range t.comments {
		if End(c) <= boundary {
			out = append(out, c)
		}
	}
	return out
}

func (t *tokenList) commentsAfter(node Node) []Node {
	boundary := End(node)
	var out []Node
	for _, c := range t.comments {
		if Start(c) >= boundary {
			out = append(out, c)
		}
	}
	return out
}

func (t *tokenList) commentsInside(node Node) []Node {
	r := MustRange(node)
	var out []Node
	for _, c := range t.comments {
		if Start(c) >= r[0] && End(c) <= r[1] {
			out = append(out, c)
		}
	}
	return out
}
