package eslint

import (
	"regexp"
	"strconv"

	espree "github.com/jclyons52/espree-go"
)

// parse.go — the parse step. ESLint hands the text to espree with
// {loc, range, comment, tokens} all on and, on failure, converts the parser's
// SyntaxError into a fatal LintMessage.

// ParseResult is a parsed program plus its token and comment streams.
type ParseResult struct {
	AST      Node
	Tokens   []Node
	Comments []Node
}

// Parse parses text with espree-go at acorn's latest ECMAScript version.
//
// sourceType is reported on the Program node (and drives scope analysis), but
// the underlying acorn-go parser is module-only: script-only syntax (`with`,
// legacy octal literals, duplicate parameter names outside sloppy mode) is
// therefore rejected. That is a documented divergence from ESLint 8, which
// parses script mode; see README "Known divergences".
func Parse(text, sourceType string) (*ParseResult, error) {
	if sourceType == "" {
		sourceType = "script"
	}
	v, err := espree.Parse(text, &espree.Options{
		SourceType:  sourceType,
		EcmaVersion: "latest",
		Loc:         true,
		Range:       true,
		Comment:     true,
		Tokens:      true,
	})
	if err != nil {
		return nil, err
	}
	prog, ok := v.(map[string]any)
	if !ok {
		return nil, &ParseError{Message: "parser returned a non-program value"}
	}
	return &ParseResult{
		AST:      prog,
		Tokens:   nodesOf(prog["tokens"]),
		Comments: nodesOf(prog["comments"]),
	}, nil
}

// ParseError is returned for unparsable input.
type ParseError struct {
	Message string
	// Line is 1-based, Column 1-based (esprima convention, as espree reports).
	Line   int
	Column int
}

func (e *ParseError) Error() string { return e.Message }

// errorPosition matches acorn's " (line:column)" message suffix. acorn-go
// formats parser errors that way; real espree attaches lineNumber/column
// properties instead and leaves them out of the message.
var errorPosition = regexp.MustCompile(`\s*\((\d+):(\d+)\)\s*$`)

// normalizeParseError converts a parser error into espree's shape: the message
// with any position suffix removed, plus 1-based line/column. espree-go already
// normalizes errors into espree's SyntaxError shape (message without suffix,
// 1-based line and column); acorn-style " (line:column)" suffixes are still
// handled for any error that reaches us unnormalized.
func normalizeParseError(err error) *ParseError {
	if se, ok := err.(*espree.SyntaxError); ok {
		return &ParseError{Message: se.Message, Line: se.Line, Column: se.Column}
	}
	msg := err.Error()
	line, col := 1, 1
	if m := errorPosition.FindStringSubmatch(msg); m != nil {
		msg = msg[:len(msg)-len(m[0])]
		if l, e1 := strconv.Atoi(m[1]); e1 == nil {
			line = l
		}
		if c, e2 := strconv.Atoi(m[2]); e2 == nil {
			col = c + 1 // acorn's suffix column is 0-based; espree reports 1-based
		}
	}
	return &ParseError{Message: msg, Line: line, Column: col}
}

// nodesOf coerces a []any of node maps into []Node.
func nodesOf(v any) []Node {
	arr, ok := v.([]any)
	if !ok {
		if ms, ok := v.([]map[string]any); ok {
			out := make([]Node, 0, len(ms))
			for _, m := range ms {
				out = append(out, Node(m))
			}
			return out
		}
		return []Node{}
	}
	out := make([]Node, 0, len(arr))
	for _, el := range arr {
		if m, ok := el.(Node); ok {
			out = append(out, m)
		}
	}
	return out
}
