package nowhitespacebeforeproperty

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Doc examples.
		{ID: "clean-dot", Code: "foo.bar;\n"},
		{ID: "space-before-dot", Code: "foo .bar;\n", Fix: true},
		{ID: "space-after-dot", Code: "foo. bar;\n", Fix: true},
		{ID: "spaces-around-dot", Code: "foo . bar;\n", Fix: true},
		{ID: "clean-bracket", Code: "foo[bar];\n"},
		{ID: "space-before-bracket", Code: "foo [bar];\n", Fix: true},
		{ID: "space-inside-bracket", Code: "foo[ bar];\n", Fix: true},
		{ID: "spaces-around-bracket", Code: "foo [ bar ];\n", Fix: true},

		// Chained members: only the offending link is reported.
		{ID: "chain-clean", Code: "a.b.c;\n"},
		{ID: "chain-middle", Code: "a .b.c;\n", Fix: true},
		{ID: "chain-outer", Code: "a.b .c;\n", Fix: true},
		{ID: "chain-both", Code: "a .b .c;\n", Fix: true},
		{ID: "chain-mixed", Code: "a [b] .c;\n", Fix: true},

		// Different lines: never reported.
		{ID: "newline-before-dot", Code: "foo\n.bar;\n"},
		{ID: "newline-indented-dot", Code: "foo\n  .bar;\n"},
		{ID: "newline-before-bracket", Code: "foo\n[bar];\n"},

		// Numeric object: reported but not fixed (5.toString() is a SyntaxError).
		{ID: "decimal-integer-object", Code: "5 .toString();\n"},
		{ID: "decimal-integer-member", Code: "5 .toString;\n"},
		{ID: "parenthesised-number", Code: "(5).toString();\n"},
		{ID: "float-object", Code: "5.5 .toString();\n", Fix: true},
		{ID: "hex-object", Code: "0x5 .toString();\n", Fix: true},

		// Optional chaining.
		{ID: "optional-clean", Code: "a?.b;\n"},
		{ID: "optional-space-before", Code: "a ?.b;\n", Fix: true},
		{ID: "optional-space-after", Code: "a?. b;\n", Fix: true},
		{ID: "optional-spaces-around", Code: "a ?. b;\n", Fix: true},
		{ID: "optional-computed-clean", Code: "a?.[b];\n"},
		{ID: "optional-computed-space", Code: "a?.[ b];\n", Fix: true},
		{ID: "optional-computed-before", Code: "a ?.[b];\n", Fix: true},

		// Comments block the fix.
		{ID: "comment-before-dot", Code: "foo /* c */ .bar;\n"},
		{ID: "line-comment-before-dot", Code: "foo // c\n.bar;\n"},
		{ID: "comment-before-bracket", Code: "foo /* c */ [bar];\n"},
		{ID: "comment-clean", Code: "foo.bar; /* c */\n"},

		// Other member shapes.
		{ID: "computed-expression", Code: "foo[a + b];\n"},
		{ID: "computed-string", Code: "foo['bar'];\n"},
		{ID: "computed-space-before", Code: "foo ['bar'];\n", Fix: true},
		{ID: "this-member", Code: "this .foo;\n", Fix: true},
		{ID: "nested-object", Code: "f(a .b);\n", Fix: true},
		{ID: "clean", Code: "var a = foo.bar;\nvar b = foo['bar'];\nvar c = foo?.bar;\n"},

		// Severity.
		{ID: "warn", Code: "foo .bar;\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "foo .bar;\n", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Non-ASCII: the reported column and fix range are UTF-16 code units.
		{ID: "nonascii-object", Code: "caf\u00e9 .bar;\n", Fix: true},
		{ID: "nonascii-property", Code: "foo .caf\u00e9;\n", Fix: true},
		{ID: "nonascii-computed", Code: "foo ['\u4f60\u597d'];\n", Fix: true},
		{ID: "nonascii-emoji", Code: "foo .\"\U0001f600\";\n"},
		{ID: "nonascii-clean", Code: "caf\u00e9.bar;\n"},
	})
}
