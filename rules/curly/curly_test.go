package curly

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- default ("all") ----
		{ID: "default-if-no-braces", Code: "if (foo) foo++;\n", Fix: true},
		{ID: "default-if-braces", Code: "if (foo) { foo++; }\n"},
		{ID: "default-if-else-braces", Code: "if (foo) { foo++; } else { bar(); }\n"},
		{ID: "default-if-else-no-braces", Code: "if (foo) foo();\nelse bar();\n", Fix: true},
		{ID: "default-while", Code: "while (foo) foo++;\n", Fix: true},
		{ID: "default-while-braces", Code: "while (foo) { foo++; }\n"},
		{ID: "default-do-while", Code: "do foo++; while (bar);\n", Fix: true},
		{ID: "default-do-while-braces", Code: "do { foo++; } while (bar);\n"},
		{ID: "default-for", Code: "for (;;) foo();\n", Fix: true},
		{ID: "default-for-in", Code: "for (var i in obj) foo(i);\n", Fix: true},
		{ID: "default-for-of", Code: "for (const x of xs) foo(x);\n", Fix: true},
		{ID: "default-for-of-braces", Code: "for (const x of xs) { foo(x); }\n"},
		{ID: "default-chain", Code: "if (a) b(); else if (c) d(); else e();\n", Fix: true},
		{ID: "default-chain-braces", Code: "if (a) { b(); } else if (c) { d(); } else { e(); }\n"},
		{ID: "default-nested-if-in-while", Code: "while (a) if (b) c();\n", Fix: true},
		{ID: "default-empty-statement", Code: "if (a); else b();\n", Fix: true},
		{ID: "default-labeled", Code: "foo: if (a) b();\n", Fix: true},
		{ID: "default-warn-severity", Code: "if (foo) foo++;\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}, Fix: true},
		{ID: "default-multiline-body", Code: "if (foo) bar(\n  1,\n  2);\n", Fix: true},
		{ID: "default-clean-function", Code: "function f() {\n  if (a) {\n    return 1;\n  }\n  return 2;\n}\n"},

		// ---- multi-line ----
		{ID: "multi-line-split-if-else", Code: "if (foo)\n  doSomething();\nelse\n  doSomethingElse();\n", Options: []any{"multi-line"}, Fix: true},
		{ID: "multi-line-collapsed", Code: "if (foo) { doSomething(); }\nelse { doSomethingElse(); }\n", Options: []any{"multi-line"}},
		{ID: "multi-line-collapsed-mixed", Code: "if (foo) { doSomething(); }\nelse doSomethingElse();\n", Options: []any{"multi-line"}},
		{ID: "multi-line-nested-call", Code: "if (foo) foo(\n  bar,\n  baz);\n", Options: []any{"multi-line"}, Fix: true},
		{ID: "multi-line-while", Code: "while (foo)\n  bar();\n", Options: []any{"multi-line"}, Fix: true},

		// ---- multi ----
		{ID: "multi-single-stmt-block", Code: "if (foo) { foo++; }\n", Options: []any{"multi"}, Fix: true},
		{ID: "multi-two-stmt-block", Code: "if (foo) { foo(); bar(); }\n", Options: []any{"multi"}},
		{ID: "multi-lexical-decl", Code: "if (foo) { let x = 1; }\n", Options: []any{"multi"}},
		{ID: "multi-no-block", Code: "if (foo) foo++;\n", Options: []any{"multi"}},
		{ID: "multi-do-while-space", Code: "do{foo()}\nwhile (bar);\n", Options: []any{"multi"}, Fix: true},
		{ID: "multi-do-while-same-line", Code: "do{foo()} while (bar);\n", Options: []any{"multi"}},
		{ID: "multi-else-block", Code: "if (foo) { bar(); } else { baz(); }\n", Options: []any{"multi"}, Fix: true},
		{ID: "multi-unsafe-if-else", Code: "if (a) {\n  if (b) c();\n} else {\n  d();\n}\n", Options: []any{"multi"}, Fix: true},
		{ID: "multi-nested-block-body", Code: "if (foo) { while (x) {} } baz();\n", Options: []any{"multi"}, Fix: true},
		{ID: "multi-no-fix-same-line", Code: "if (foo) { bar() } baz();\n", Options: []any{"multi"}},
		{ID: "multi-no-fix-increment", Code: "if (foo) { bar++ }\nbaz();\n", Options: []any{"multi"}},
		{ID: "multi-no-fix-unsafe-start", Code: "if (foo) { bar() }\n(baz)();\n", Options: []any{"multi"}},
		{ID: "multi-no-fix-slash-start", Code: "if (foo) { bar() }\n/x/.test(baz);\n", Options: []any{"multi"}},
		{ID: "multi-fix-next-line", Code: "if (foo) { bar() }\nbaz();\n", Options: []any{"multi"}, Fix: true},
		{ID: "multi-function-body-block", Code: "if (foo) { const f = function () {}; }\n", Options: []any{"multi"}},

		// ---- multi-or-nest ----
		{ID: "multi-or-nest-one-liner", Code: "if (foo) bar();\n", Options: []any{"multi-or-nest"}},
		{ID: "multi-or-nest-block-one-liner", Code: "if (foo) { bar(); }\n", Options: []any{"multi-or-nest"}, Fix: true},
		{ID: "multi-or-nest-split-one-liner", Code: "if (foo)\n  bar();\n", Options: []any{"multi-or-nest"}},
		{ID: "multi-or-nest-multiline-body", Code: "if (foo)\n  bar(\n  baz);\n", Options: []any{"multi-or-nest"}, Fix: true},
		{ID: "multi-or-nest-block-multiline", Code: "if (foo) { bar(\n  baz); }\n", Options: []any{"multi-or-nest"}},
		{ID: "multi-or-nest-comment-in-block", Code: "if (foo) { /* c */ bar(); }\n", Options: []any{"multi-or-nest"}},
		{ID: "multi-or-nest-empty-statement", Code: "if (foo);\n", Options: []any{"multi-or-nest"}},
		{ID: "multi-or-nest-empty-statement-block", Code: "if (foo) { ; }\n", Options: []any{"multi-or-nest"}, Fix: true},

		// ---- consistent ----
		{ID: "consistent-mixed-chain", Code: "if (foo) { bar(); } else baz();\n", Options: []any{"multi-or-nest", "consistent"}, Fix: true},
		{ID: "consistent-chain-multi", Code: "if (a) { b(); } else if (c) { d(); } else { e(); }\n", Options: []any{"multi", "consistent"}, Fix: true},
		{ID: "consistent-all-braced", Code: "if (a) { b(); } else if (c) { d(); } else { e(); }\n", Options: []any{"multi-line", "consistent"}},
		{ID: "consistent-mixed-multiline", Code: "if (a) {\n  b();\n} else c();\n", Options: []any{"multi-line", "consistent"}, Fix: true},

		// ---- non-ASCII (fix ranges are UTF-16 code units) ----
		{ID: "nonascii-ident", Code: "if (caf\u00e9) \u4f60\u597d();\n", Fix: true},
		{ID: "nonascii-string-body", Code: "if (a) console.log(\"h\u00e9llo\");\n", Fix: true},
		{ID: "nonascii-emoji-body", Code: "if (a) f(\"\U0001f600\");\n", Fix: true},
		{ID: "nonascii-multi-block", Code: "if (a) { f(\"\u4f60\u597d\"); }\n", Options: []any{"multi"}, Fix: true},
	})
}
