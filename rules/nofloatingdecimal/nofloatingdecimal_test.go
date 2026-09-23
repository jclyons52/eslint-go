package nofloatingdecimal

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Doc examples.
		{ID: "leading", Code: "var num = .5;\n", Fix: true},
		{ID: "trailing", Code: "var num = 5.;\n", Fix: true},
		{ID: "valid-leading", Code: "var num = 0.5;\n"},
		{ID: "valid-trailing", Code: "var num = 5.0;\n"},

		// Both ends of the same expression.
		{ID: "both-sides", Code: "var num = .5 + .5;\n", Fix: true},
		{ID: "trailing-expression", Code: "var num = 2. * 3.;\n", Fix: true},

		// Sign-prefixed (the fix has to stay adjacent to the unary operator).
		{ID: "negative-leading", Code: "var num = -.5;\n", Fix: true},
		{ID: "positive-leading", Code: "var num = +.5;\n", Fix: true},
		{ID: "negative-trailing", Code: "var num = -5.;\n", Fix: true},

		// Preceded by a keyword token with no space at all: the fix must add one
		// so `typeof.5` does not become `typeof0.5`.
		{ID: "typeof-adjacent", Code: "typeof.5;\n", Fix: true},

		// Exponent forms.
		{ID: "leading-exponent", Code: "var num = .5e3;\n", Fix: true},
		{ID: "trailing-exponent", Code: "var num = 5.e3;\n"},
		{ID: "plain-exponent", Code: "var num = 5e3;\n"},
		{ID: "leading-big-exponent", Code: "var num = .5E-3;\n", Fix: true},

		// Contexts.
		{ID: "in-condition", Code: "if (x === .5) { y(); }\n", Fix: true},
		{ID: "in-array", Code: "[.5, 1., 0.25]\n", Fix: true},
		{ID: "in-call", Code: "f(.5);\n", Fix: true},
		{ID: "in-return", Code: "function f() {\n  return .5;\n}\n", Fix: true},
		{ID: "after-newline", Code: "var num =\n.5;\n", Fix: true},
		{ID: "member-call", Code: "(5.).toString();\n", Fix: true},
		{ID: "nested", Code: "var a = { b: .5, c: [1.] };\n", Fix: true},

		// Not numbers: strings, bigints and regexes must not be touched.
		{ID: "string-dot", Code: "var s = \".\";\n"},
		{ID: "bigint", Code: "var b = 5n;\n"},
		{ID: "regex", Code: "var re = /5./;\n"},
		{ID: "hex", Code: "var h = 0x5;\n"},

		// Severity.
		{ID: "warn", Code: "var num = .5;\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "var num = .5;\n", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Clean file.
		{ID: "clean", Code: "var a = 0.5;\nvar b = 1.25;\nvar c = 1e3;\n"},

		// Non-ASCII: the fix range and reported column are UTF-16 code units.
		{ID: "nonascii-leading", Code: "var caf\u00e9 = .5;\n", Fix: true},
		{ID: "nonascii-before", Code: "var s = \"\u4f60\u597d\";\nvar n = .5;\n", Fix: true},
		{ID: "nonascii-trailing", Code: "var s = \"\U0001f600\";\nvar n = 5.;\n", Fix: true},

		// `.5` at offset 0: the fix inserts "0" at range [0,0], i.e. a replacement
		// text shorter than the 3-byte UTF-8 BOM — the shape that used to panic in
		// the core's applyFixes (it sliced fix.Text by len(bom); ESLint uses
		// String.prototype.startsWith). Fixed, so the offset-0 leading form is
		// compared with its fix like every other leading form.
		{ID: "statement", Code: ".5;\n", Fix: true},
	})
}
