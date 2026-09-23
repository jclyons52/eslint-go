package noselfcompare

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Doc examples.
		{ID: "doc-incorrect", Code: "if (x === x) { }"},
		{ID: "doc-correct", Code: "if (x === y) { }"},

		// Every operator in the rule's set, and the ones outside it.
		{ID: "eq", Code: "x === x;"},
		{ID: "neq", Code: "x !== x;"},
		{ID: "loose-eq", Code: "x == x;"},
		{ID: "loose-neq", Code: "x != x;"},
		{ID: "lt", Code: "x < x;"},
		{ID: "lte", Code: "x <= x;"},
		{ID: "gt", Code: "x > x;"},
		{ID: "gte", Code: "x >= x;"},
		{ID: "add", Code: "x + x;"},
		{ID: "strict-diff", Code: "x === y;"},
		{ID: "and-or", Code: "x && x;"},
		{ID: "nullish", Code: "x ?? x;"},
		{ID: "in", Code: "x in x;"},
		{ID: "instanceof", Code: "x instanceof x;"},
		{ID: "shift", Code: "x >> x;"},
		{ID: "pipe", Code: "x | x;"},
		{ID: "exp", Code: "x ** x;"},

		// All operators in one file (report order follows traversal).
		{ID: "all-operators", Code: "x === x; x !== x; x == x; x != x; x > x; x < x; x >= x; x <= x;"},

		// Token-level comparison: parens live outside the operand's range.
		{ID: "parens-both", Code: "(x) === (x);"},
		{ID: "parens-one", Code: "(x) === x;"},
		{ID: "parens-expr", Code: "(a + b) === (a + b);"},
		{ID: "parens-nested", Code: "((x)) === (x);"},

		// Comments are not tokens.
		{ID: "comment-left", Code: "x /* c */ === x;"},
		{ID: "comment-right", Code: "x === /* c */ x;"},
		{ID: "comment-line", Code: "x === // c\n x;"},

		// Member/call expressions.
		{ID: "member-same", Code: "a.b === a.b;"},
		{ID: "member-diff", Code: "a.b === a.c;"},
		{ID: "member-order", Code: "a.b === b.a;"},
		{ID: "call-same", Code: "f() === f();"},
		{ID: "call-diff-args", Code: "f(1) === f(2);"},
		{ID: "index-same", Code: "a[0] === a[0];"},
		{ID: "optional-chain", Code: "a?.b === a?.b;"},

		// Literals.
		{ID: "number-same", Code: "1 === 1;"},
		{ID: "string-diff", Code: "a === 'b';"},
		{ID: "string-same", Code: "'a' === 'a';"},
		{ID: "number-vs-string", Code: "1 === '1';"},
		{ID: "regex-same", Code: "/a/ === /a/;"},
		{ID: "bigint-same", Code: "1n === 1n;"},
		{ID: "template-same", Code: "`a` === `a`;"},
		{ID: "nan", Code: "NaN === NaN;"},
		{ID: "null", Code: "null === null;"},
		{ID: "undefined", Code: "undefined === undefined;"},

		// `this`, unary and update expressions.
		{ID: "this", Code: "this.x === this.x;"},
		{ID: "typeof", Code: "typeof x === typeof x;"},
		{ID: "negate", Code: "-x === -x;"},
		{ID: "not", Code: "!x === !x;"},

		// Nesting and control flow.
		{ID: "nested-if", Code: "if (a) { if (b.c == b.c) { } }"},
		{ID: "function", Code: "function f(a) { return a === a; }"},
		{ID: "arrow", Code: "const f = (a) => a !== a;"},
		{ID: "multiline", Code: "const r = a.b\n    ===\n    a.b;"},
		{ID: "nested-parens-deep", Code: "if (((a === a))) { }"},
		{ID: "ternary", Code: "const r = a === a ? 1 : 2;"},

		// Both operands a parenthesised expression statement.
		{ID: "sequence", Code: "(a, b) === (a, b);"},
		{ID: "unary-paren", Code: "(typeof a) === (typeof a);"},

		// Non-ASCII operands (token comparison).
		{ID: "non-ascii", Code: "const é = 1; é === é;"},
		{ID: "non-ascii-diff", Code: "const é = 1; é === e;"},
		{ID: "non-ascii-string", Code: "'é' === 'é'; 'é' === 'e';"},

		// A clean file reports nothing.
		{ID: "clean", Code: "const a = 1; const b = 2; if (a === b) { }\nif (a !== b) { }\n"},

		// Severity.
		{ID: "warn", Code: "x === x;", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "x === x;", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Explicit config object (parserOptions through the harness's Config path).
		{ID: "explicit-config", Code: "x === x;", Config: map[string]any{
			"rules":         map[string]any{Name: 2},
			"parserOptions": map[string]any{"ecmaVersion": 2022, "sourceType": "module"},
		}},
	})
}
