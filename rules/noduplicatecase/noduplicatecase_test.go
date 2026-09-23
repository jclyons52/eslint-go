package noduplicatecase

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Doc examples.
		{ID: "doc-incorrect", Code: "switch (a) {\n    case 1:\n        break;\n    case 2:\n        break;\n    case 1:\n        break;\n    default:\n        break;\n}"},
		{ID: "doc-correct", Code: "switch (a) {\n    case 1:\n        break;\n    case 2:\n        break;\n    case 3:\n        break;\n    default:\n        break;\n}"},

		// Literals of every kind.
		{ID: "number", Code: "switch (a) { case 1: break; case 1: break; }"},
		{ID: "string", Code: "switch (a) { case 'x': break; case 'x': break; }"},
		{ID: "boolean", Code: "switch (a) { case true: break; case true: break; }"},
		{ID: "null", Code: "switch (a) { case null: break; case null: break; }"},
		{ID: "regex", Code: "switch (a) { case /x/: break; case /x/: break; }"},
		{ID: "bigint", Code: "switch (a) { case 1n: break; case 1n: break; }"},
		{ID: "template", Code: "switch (a) { case `x`: break; case `x`: break; }"},
		{ID: "number-vs-string", Code: "switch (a) { case 1: break; case '1': break; }"},
		{ID: "number-vs-string-raw", Code: "switch (a) { case 1: break; case 1.0: break; }"},
		{ID: "hex-vs-decimal", Code: "switch (a) { case 0x10: break; case 16: break; }"},
		{ID: "true-vs-false", Code: "switch (a) { case true: break; case false: break; }"},
		{ID: "nan", Code: "switch (a) { case NaN: break; case NaN: break; }"},

		// Identifier and member tests.
		{ID: "identifier", Code: "switch (a) { case b: break; case b: break; }"},
		{ID: "identifier-diff", Code: "switch (a) { case b: break; case c: break; }"},
		{ID: "member", Code: "switch (a) { case obj.x: break; case obj.x: break; }"},
		{ID: "member-diff", Code: "switch (a) { case obj.x: break; case obj.y: break; }"},
		{ID: "call", Code: "switch (a) { case f(): break; case f(): break; }"},
		{ID: "call-diff", Code: "switch (a) { case f(): break; case g(): break; }"},
		{ID: "expr", Code: "switch (a) { case b + c: break; case b + c: break; }"},
		{ID: "expr-diff", Code: "switch (a) { case b + c: break; case c + b: break; }"},
		{ID: "expr-parens", Code: "switch (a) { case b + c: break; case (b + c): break; }"},
		{ID: "parens-identifier", Code: "switch (a) { case b: break; case (b): break; }"},
		{ID: "parens-both", Code: "switch (a) { case (b): break; case (b): break; }"},
		{ID: "typeof", Code: "switch (a) { case typeof b: break; case typeof b: break; }"},

		// default clauses have no test and must not be compared.
		{ID: "default-only", Code: "switch (a) { default: break; }"},
		{ID: "default-then-dup", Code: "switch (a) { default: break; case 1: break; case 1: break; }"},
		{ID: "default-middle", Code: "switch (a) { case 1: break; default: break; case 1: break; }"},
		{ID: "empty-switch", Code: "switch (a) { }"},

		// Three identical labels: the second and third are reported.
		{ID: "triple", Code: "switch (a) { case 1: case 1: case 1: break; }"},
		{ID: "four", Code: "switch (a) { case 1: break; case 1: break; case 1: break; case 1: break; }"},

		// Comments are not tokens.
		{ID: "comment-before", Code: "switch (a) { case 1: break; case /*x*/ 1: break; }"},
		{ID: "comment-inside", Code: "switch (a) { case 1: break; case 1 /*x*/: break; }"},
		{ID: "comment-line", Code: "switch (a) { case 1: break; case //x\n 1: break; }"},

		// Nested switches compare within their own statement.
		{ID: "nested", Code: "switch (a) { case 1: switch (b) { case 1: break; case 1: break; } }"},
		{ID: "nested-outer-dup", Code: "switch (a) { case 1: switch (b) { case 2: break; } case 1: break; }"},
		{ID: "nested-same-literal", Code: "switch (a) { case 1: switch (b) { case 1: break; } }"},
		{ID: "two-switches", Code: "switch (a) { case 1: break; } switch (b) { case 1: break; }"},
		{ID: "switch-in-function", Code: "function f(x) { switch (x) { case 1: break; case 1: break; } }"},

		// Clean files.
		{ID: "clean", Code: "switch (a) { case 1: break; case 2: break; default: break; }"},
		{ID: "clean-empty", Code: "const a = 1;\n"},

		// Non-ASCII labels.
		{ID: "non-ascii", Code: "switch (a) { case 'é': break; case 'é': break; }"},
		{ID: "non-ascii-ident", Code: "switch (a) { case é: break; case é: break; }"},
		{ID: "non-ascii-diff", Code: "switch (a) { case 'é': break; case 'e': break; }"},

		// Severity.
		{ID: "warn", Code: "switch (a) { case 1: break; case 1: break; }", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "switch (a) { case 1: break; case 1: break; }", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Several reports in one statement, and several statements.
		{ID: "multi", Code: "switch (a) { case 1: break; case 1: break; case 2: break; case 2: break; }"},
		{ID: "multi-statements", Code: "switch (a) { case 1: break; case 1: break; }\nswitch (b) { case 'x': break; case 'x': break; }\n"},
	})
}
