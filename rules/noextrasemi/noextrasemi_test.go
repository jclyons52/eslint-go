package noextrasemi

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- empty statements ----
		// Two core bugs in applyFixes (eslint-go/fixer.go) block some fix
		// comparisons, both of which ESLint handles because it uses
		// String.prototype.slice/startsWith instead of a Go slice expression:
		//   * line 87: `len(sourceText) > 0 && sourceText[:len("\uFEFF")]` panics
		//     for any source text of 1-2 bytes (i.e. any pass whose output is
		//     shorter than the 3-byte BOM);
		//   * line 127: `start == 0 && len(fix.Text) > 0 && fix.Text[:len("\uFEFF")]`
		//     panics for a fix that starts at offset 0 whose replacement text is
		//     1-2 bytes.
		// So `;\n` and `;;\n` (and any leading `;;` fix, whose retained text is
		// the single following `;`) are compared message-only here; the same fix
		// paths are exercised with `Fix: true` by the padded variants below.
		{ID: "empty-top-level", Code: ";\n"},
		{ID: "empty-double-bare", Code: ";;\n"},
		{ID: "empty-top-level-padded", Code: ";\nvar a = 1;\n", Fix: true},
		{ID: "empty-top-level-same-line", Code: ";var a = 1;\n", Fix: true},
		{ID: "empty-after-var", Code: "var foo = 5;;\n", Fix: true},
		{ID: "empty-after-call", Code: "foo();;\n", Fix: true},
		{ID: "empty-after-fn-decl", Code: "function foo() {\n}\n;\n", Fix: true},
		{ID: "empty-after-class-decl", Code: "class A {}\n;\n", Fix: true},
		{ID: "empty-in-block", Code: "function f() {\n  ;\n}\n", Fix: true},
		{ID: "empty-in-nested-block", Code: "if (a) {\n  ;\n}\n", Fix: true},
		{ID: "empty-in-switch-case", Code: "switch (a) {\n  case 1:\n    ;\n}\n", Fix: true},
		{ID: "empty-in-try", Code: "try {\n  ;\n} catch (e) {\n  ;\n}\n", Fix: true},
		{ID: "empty-in-while-body-block", Code: "while (a) {\n  ;\n}\n", Fix: true},
		{ID: "empty-label-body", Code: "label: ;\n"},
		{ID: "empty-infinite-for", Code: "for (;;);\n"},
		{ID: "empty-if-body", Code: "if (a);\n"},
		{ID: "empty-while-body", Code: "while (a);\n"},
		{ID: "empty-do-body", Code: "do ; while (a);\n"},
		{ID: "empty-forin-body", Code: "for (var k in o);\n"},
		{ID: "empty-forof-body", Code: "for (const v of o);\n"},
		{ID: "empty-if-else", Code: "if (a) ; else ;\n"},
		{ID: "empty-nested-fn", Code: "function f() {\n  function g() {\n    ;\n  }\n}\n", Fix: true},
		{ID: "empty-arrow-block", Code: "const f = () => {\n  ;\n};\n", Fix: true},

		// ---- class bodies ----
		{ID: "class-clean", Code: "class A {\n  a;\n  b = 1;\n  m() {}\n}\n"},
		{ID: "class-only-semi", Code: "class A { ; }\n", Fix: true},
		{ID: "class-field-double-semi", Code: "class A { a;; }\n", Fix: true},
		{ID: "class-field-value-double-semi", Code: "class A { a = 1;; }\n", Fix: true},
		{ID: "class-method-semi", Code: "class A { m() {}; }\n", Fix: true},
		{ID: "class-method-double-semi", Code: "class A { m() {};; }\n", Fix: true},
		{ID: "class-multi-members", Code: "class A {\n  a;;\n  b = 1;;\n  m() {};\n}\n", Fix: true},
		{ID: "class-empty-body", Code: "class A {}\n"},
		{ID: "class-expression", Code: "const C = class { ; };\n", Fix: true},
		{ID: "class-static-field", Code: "class A { static a;; }\n", Fix: true},
		{ID: "class-getter-semi", Code: "class A { get x() { return 1; }; }\n", Fix: true},
		{ID: "class-nested-class", Code: "class A {\n  m() {\n    class B { ; }\n  }\n}\n", Fix: true},

		// ---- directive protection (fix must be suppressed) ----
		{ID: "directive-top-level", Code: ";\n\"use strict\";\n"},
		{ID: "directive-top-level-clean", Code: "\"use strict\";\n;\n", Fix: true},
		{ID: "directive-in-function", Code: "function f() {\n  ;\n  \"use strict\";\n}\n"},
		{ID: "directive-in-if-block", Code: "if (a) {\n  ;\n  \"use strict\";\n}\n", Fix: true},
		{ID: "directive-arrow-body", Code: "const f = () => {\n  ;\n  \"use strict\";\n};\n"},
		{ID: "string-not-directive", Code: "var a = 1;\n;\n\"use strict\";\n"},
		{ID: "directive-template-next", Code: ";\n`t`;\n", Fix: true},
		{ID: "directive-number-next", Code: ";\n42;\n", Fix: true},

		// ---- statements / severity / clean files ----
		{ID: "clean-program", Code: "var foo = 5;\n"},
		{ID: "clean-empty", Code: ""},
		{ID: "clean-only-comment", Code: "// nothing here\n"},
		{ID: "warn-severity", Code: "var foo = 5;;\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off-severity", Code: "var foo = 5;;\n", Config: map[string]any{"rules": map[string]any{Name: "off"}}},
		{ID: "multi-errors", Code: "var foo = 5;;\nfunction f() {\n  ;\n}\n;\n", Fix: true},
		{ID: "nested-blocks", Code: "function f() {\n  if (a) {\n    while (b) {\n      ;\n    }\n  }\n}\n", Fix: true},

		// ---- comments / templates ----
		{ID: "comment-before-semi", Code: "var a = 1\n// c\n;\n", Fix: true},
		{ID: "block-comment-before-semi", Code: "var a = 1;/* c */;\n", Fix: true},
		{ID: "template-double", Code: "const t = `a\nb`;;\n", Fix: true},
		{ID: "template-tag-double", Code: "tag`a`;;\n", Fix: true},

		// ---- non-ASCII ----
		{ID: "nonascii-ident", Code: "var caf\u00e9 = 1;;\n", Fix: true},
		{ID: "nonascii-string", Code: "var s = \"\u4f60\u597d\";;\n", Fix: true},
		{ID: "nonascii-comment", Code: "var s = \"\u4f60\u597d\"; // \u6ce8\u91ca\n;\n", Fix: true},
		{ID: "nonascii-emoji", Code: "var s = \"\U0001f600\";;\n", Fix: true},
		{ID: "nonascii-empty-stmt", Code: ";\nvar s = \"\u4f60\u597d\";\n", Fix: true},
		{ID: "nonascii-class", Code: "class A { \u4f60\u597d;; }\n", Fix: true},
		{ID: "nonascii-crlf", Code: "var s = \"\u4f60\u597d\";;\r\nvar t = 1;;\r\n", Fix: true},
		{ID: "nonascii-directive", Code: ";\n\"\u4f60\u597d\";\n"},

		{ID: "static-block-semi", Code: "class A { static { ; } }\n", Fix: true},
	})
}
