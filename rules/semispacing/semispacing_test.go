package semispacing

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- defaults (before: false, after: true) ----
		{ID: "default-clean", Code: "var a = 1;\nvar b = 2;\n"},
		{ID: "default-clean-same-line", Code: "var a = 1; var b = 2;\n"},
		{ID: "default-extra-before", Code: "var a = 1 ;\n", Fix: true},
		{ID: "default-extra-before-many", Code: "var a = 1   ;\n", Fix: true},
		{ID: "default-missing-after", Code: "var a = 1;var b = 2;\n", Fix: true},
		{ID: "default-missing-after-tight", Code: "var a = 'b';c = 1;\n", Fix: true},
		{ID: "default-both-wrong", Code: "var a = 1 ;var b = 2;\n", Fix: true},
		{ID: "default-trailing-space-eol", Code: "var a = 1; \nvar b = 2;\n"},
		{ID: "default-space-before-eol", Code: "var a = 1 ;\nvar b = 2;\n", Fix: true},
		{ID: "default-leading-semi", Code: "var a = 1\n;foo()\n"},
		{ID: "default-semi-before-brace", Code: "if (a) { b(); }\n"},
		{ID: "default-semi-before-paren", Code: "for (var i = 0; i < 1; i++) {}\n"},
		{ID: "default-warn", Code: "var a = 1 ;\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "default-off", Code: "var a = 1 ;\n", Config: map[string]any{"rules": map[string]any{Name: "off"}}},
		{ID: "default-empty", Code: ""},
		{ID: "default-empty-options", Code: "var a = 1;var b = 2;\n", Options: []any{map[string]any{}}},

		// ---- option matrix ----
		{ID: "before-true-clean", Code: "var a = 1 ;var b = 2;\n", Options: []any{map[string]any{"before": true}}},
		{ID: "before-true-missing", Code: "var a = 1;\nvar b = 2;\n", Options: []any{map[string]any{"before": true}}, Fix: true},
		{ID: "before-true-extra-after", Code: "var a = 1 ; var b = 2;\n", Options: []any{map[string]any{"before": true}}, Fix: true},
		{ID: "before-and-after-true", Code: "var a = 1;\n", Options: []any{map[string]any{"before": true, "after": true}}, Fix: true},
		{ID: "before-true-after-false", Code: "var a = 1 ; var b = 2 ;\n", Options: []any{map[string]any{"before": true, "after": false}}, Fix: true},
		{ID: "after-false-extra", Code: "var a = 1; var b = 2;\n", Options: []any{map[string]any{"after": false}}, Fix: true},
		{ID: "after-true-missing", Code: "var a = 1;var b = 2;\n", Options: []any{map[string]any{"after": true}}, Fix: true},
		{ID: "before-false-explicit", Code: "var a = 1 ;\n", Options: []any{map[string]any{"before": false, "after": true}}, Fix: true},
		{ID: "both-false", Code: "var a = 1 ; var b = 2 ;\n", Options: []any{map[string]any{"before": false, "after": false}}, Fix: true},

		// ---- statement kinds ----
		{ID: "stmt-var-decl", Code: "var a = 1;var b = 2;\n", Fix: true},
		{ID: "stmt-expr", Code: "foo() ;\n", Fix: true},
		{ID: "stmt-return", Code: "function f() {\n  return 1 ;\n}\n", Fix: true},
		{ID: "stmt-throw", Code: "function f() {\n  throw 1;var x = 1;\n}\n", Fix: true},
		{ID: "stmt-debugger", Code: "debugger ;\n", Fix: true},
		{ID: "stmt-break", Code: "while (a) {\n  break ;\n}\n", Fix: true},
		{ID: "stmt-continue", Code: "while (a) {\n  continue ;\n}\n", Fix: true},
		{ID: "stmt-dowhile", Code: "do {\n  a();\n} while (b) ;\n", Fix: true},
		{ID: "stmt-import", Code: "import a from 'a' ;\n", Fix: true},
		{ID: "stmt-import-missing", Code: "import a from 'a';import b from 'b';\n", Fix: true},
		{ID: "stmt-export-all", Code: "export * from 'a' ;\n", Fix: true},
		{ID: "stmt-export-named", Code: "const a = 1;\nexport { a } ;\n", Fix: true},
		{ID: "stmt-export-default", Code: "export default foo ;\n", Fix: true},
		{ID: "stmt-class-field", Code: "class A {\n  a = 1 ;\n}\n", Fix: true},
		{ID: "stmt-class-field-tight", Code: "class A {\n  a = 1;b = 2;\n}\n", Fix: true},
		{ID: "stmt-class-field-method", Code: "class A {\n  m() {}\n}\n"},

		// ---- for statements ----
		{ID: "for-head-clean", Code: "for (var i = 0; i < 10; i++) {\n  a();\n}\n"},
		{ID: "for-head-tight", Code: "for (var i = 0;i < 10;i++) {\n  a();\n}\n", Fix: true},
		{ID: "for-head-spaced-before", Code: "for (var i = 0 ; i < 10 ; i++) {\n  a();\n}\n", Fix: true},
		{ID: "for-head-no-init", Code: "for (; i < 10; i++) {\n  a();\n}\n"},
		{ID: "for-head-no-test", Code: "for (var i = 0; ; i++) {\n  a();\n}\n"},
		{ID: "for-head-empty", Code: "for (;;) {\n  a();\n}\n"},
		{ID: "for-head-tight-no-test", Code: "for (var i = 0 ;;i++) {\n  a();\n}\n", Fix: true},
		{ID: "for-in", Code: "for (var k in o) {\n  a();\n}\n"},
		{ID: "for-of", Code: "for (const v of o) {\n  a();\n}\n"},

		// ---- empty statements / doubled semicolons ----
		{ID: "empty-stmt", Code: ";\n"},
		{ID: "empty-stmt-leading", Code: ";foo()\n"},
		{ID: "double-semi", Code: "var a = 1;;\n"},
		{ID: "double-semi-spaced", Code: "var a = 1 ; ;\n"},
		{ID: "empty-for-body", Code: "for (var i = 0; i < 1; i++);\n"},

		// ---- comments ----
		{ID: "comment-between", Code: "var a = 1;/*c*/var b = 2;\n", Fix: true},
		{ID: "comment-between-spaced", Code: "var a = 1; /*c*/ var b = 2;\n"},
		{ID: "line-comment-after", Code: "var a = 1; // c\nvar b = 2;\n"},
		{ID: "comment-before-semi", Code: "var a = 1/*c*/ ;\n", Fix: true},
		{ID: "comment-only", Code: "// just a comment\n"},

		// ---- templates ----
		{ID: "template", Code: "const t = `a\nb` ;\n", Fix: true},
		{ID: "template-tag", Code: "tag`a`;var b = 1;\n", Fix: true},

		// ---- non-ASCII ----
		{ID: "nonascii-before", Code: "var s = \"\u4f60\u597d\" ;\n", Fix: true},
		{ID: "nonascii-after", Code: "var s = \"\u4f60\u597d\";var t = 1;\n", Fix: true},
		{ID: "nonascii-ident", Code: "var caf\u00e9 = 1 ;\n", Fix: true},
		{ID: "nonascii-comment", Code: "var s = \"\u4f60\u597d\"; /* \u6ce8\u91ca */ var t = 1;\n"},
		{ID: "nonascii-emoji", Code: "var s = \"\U0001f600\" ;var t = 1;\n", Fix: true},
		{ID: "nonascii-crlf", Code: "var s = \"\u4f60\u597d\" ;\r\nvar t = 1;\r\n", Fix: true},
		{ID: "nonascii-for", Code: "for (var i = 0;i < \u4f60;i++) {}\n", Fix: true},
	})
}
