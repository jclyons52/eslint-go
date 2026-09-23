package nolonelyif

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- basic rewrite ----
		{ID: "basic", Code: "if (a) { b(); } else { if (c) { d(); } }\n", Fix: true},
		{ID: "multiline", Code: "if (a) {\n  b();\n} else {\n  if (c) {\n    d();\n  }\n}\n", Fix: true},
		{ID: "no-space-after-else", Code: "if (a) { b(); } else{ if (c) { d(); } }\n", Fix: true},
		{ID: "else-newline", Code: "if (a) { b(); }\nelse {\n  if (c) { d(); }\n}\n", Fix: true},
		{ID: "inner-if-no-block", Code: "if (a) { b(); } else { if (c) d(); }\n", Fix: true},
		{ID: "inner-if-else", Code: "if (a) { b(); } else { if (c) { d(); } else { e(); } }\n", Fix: true},
		{ID: "inner-if-else-if", Code: "if (a) { b(); } else { if (c) { d(); } else if (e) { f(); } }\n", Fix: true},
		{ID: "outer-if-no-block", Code: "if (a) b(); else { if (c) { d(); } }\n", Fix: true},
		{ID: "warn-severity", Code: "if (a) { b(); } else { if (c) { d(); } }\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}, Fix: true},
		{ID: "inner-if-multiline-condition", Code: "if (a) { b(); } else {\n  if (\n    c\n  ) {\n    d();\n  }\n}\n", Fix: true},

		// ---- already fine ----
		{ID: "clean-else-if", Code: "if (a) { b(); } else if (c) { d(); }\n"},
		{ID: "clean-else-two-statements", Code: "if (a) { b(); } else { c(); d(); }\n"},
		{ID: "clean-no-else", Code: "if (a) { b(); }\n"},
		{ID: "clean-in-while-body", Code: "while (a) { if (b) c(); }\n"},
		{ID: "clean-in-function", Code: "function f() { if (a) { b(); } }\n"},
		{ID: "clean-else-empty-block", Code: "if (a) { b(); } else { }\n"},
		{ID: "clean-else-nested-block", Code: "if (a) { b(); } else { { if (c) d(); } }\n"},
		{ID: "clean-else-non-if", Code: "if (a) { b(); } else { c(); }\n"},
		{ID: "clean-else-return", Code: "if (a) { return 1; } else { return 2; }\n"},

		// ---- fixes that must be declined ----
		{ID: "comment-before-if", Code: "if (a) { b(); } else { /* c */ if (d) { e(); } }\n"},
		{ID: "comment-after-if", Code: "if (a) { b(); } else { if (d) { e(); } /* c */ }\n"},
		{ID: "line-comment-before", Code: "if (a) { b(); } else { // c\nif (d) { e(); } }\n"},
		{ID: "asi-same-line", Code: "if (a) { b(); } else { if (c) d() } e();\n"},
		{ID: "asi-paren-after", Code: "if (a) { b(); } else { if (c) d() } (function () {})();\n"},
		{ID: "asi-bracket-after", Code: "if (a) { b(); } else { if (c) d() } [1].forEach(f);\n"},
		{ID: "asi-plus-after", Code: "if (a) { b(); } else { if (c) d() } +1;\n"},
		{ID: "asi-backtick-after", Code: "if (a) { b(); } else { if (c) d() } `x`;\n"},
		{ID: "asi-minus-after", Code: "if (a) { b(); } else { if (c) d() } -1;\n"},
		{ID: "asi-increment-last", Code: "if (a) { b(); } else { if (c) d++ } e();\n"},
		{ID: "asi-decrement-last", Code: "if (a) { b(); } else { if (c) d-- } e();\n"},

		// ---- fixes that do apply ----
		{ID: "no-token-after", Code: "if (a) { b(); } else { if (c) d()\n}\n", Fix: true},
		{ID: "semicolon-last", Code: "if (a) { b(); } else { if (c) d(); } e();\n", Fix: true},
		{ID: "block-consequent-same-line", Code: "if (a) { b(); } else { if (c) { d(); } } e();\n", Fix: true},
		{ID: "two-lonely-ifs", Code: "if (a) { b(); } else { if (c) { d(); } else { if (e) f(); } }\n", Fix: true},
		{ID: "nested-lonely-ifs-deep", Code: "if (a) { b(); } else { if (c) { d(); } else { if (e) { f(); } else { if (g) h(); } } }\n", Fix: true},
		{ID: "lonely-if-in-function", Code: "function f() {\n  if (a) {\n    b();\n  } else {\n    if (c) {\n      d();\n    }\n  }\n}\n", Fix: true},
		{ID: "lonely-if-in-switch-case", Code: "switch (x) {\n  case 1:\n    if (a) { b(); } else { if (c) { d(); } }\n}\n", Fix: true},

		// ---- non-ASCII ----
		{ID: "nonascii-before", Code: "var s = \"h\u00e9llo\";\nif (a) { b(); } else { if (c) { d(); } }\n", Fix: true},
		{ID: "nonascii-inside", Code: "if (a) { b(); } else { if (c) { f(\"\u4f60\u597d\"); } }\n", Fix: true},
		{ID: "nonascii-emoji", Code: "if (a) { f(\"\U0001f600\"); } else { if (c) { d(); } }\n", Fix: true},
	})
}
