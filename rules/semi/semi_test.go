package semi

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- "always" (the default) ----
		{ID: "always-clean", Code: "var a = 1;\nvar b = 2;\n"},
		{ID: "always-missing-var", Code: "var a = 1\nvar b = 2\n", Fix: true},
		{ID: "always-missing-expr", Code: "foo()\n", Fix: true},
		{ID: "always-missing-last-no-newline", Code: "var a = 1"},
		{ID: "always-missing-last-newline", Code: "var a = 1\n"},
		{ID: "always-missing-before-comment", Code: "var a = 1 // trailing\n"},
		{ID: "always-missing-multi", Code: "var a = 1\nfoo()\nbar()\n", Fix: true},
		{ID: "always-missing-in-function", Code: "function f() {\n  var a = 1\n}\n", Fix: true},
		{ID: "always-missing-nested", Code: "function f() {\n  if (a) {\n    b()\n  }\n}\n", Fix: true},
		{ID: "always-warn-severity", Code: "var a = 1\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "always-off-severity", Code: "var a = 1\n", Config: map[string]any{"rules": map[string]any{Name: "off"}}},
		{ID: "always-empty-program", Code: ""},
		{ID: "always-only-semi", Code: ";\n"},

		// ---- statement kinds ----
		{ID: "stmt-return", Code: "function f() {\n  return 1\n}\n", Fix: true},
		{ID: "stmt-return-void", Code: "function f() {\n  return\n}\n"},
		{ID: "stmt-throw", Code: "function f() {\n  throw new Error('x')\n}\n", Fix: true},
		{ID: "stmt-debugger", Code: "debugger\n", Fix: true},
		{ID: "stmt-break", Code: "while (a) {\n  break\n}\n", Fix: true},
		{ID: "stmt-continue", Code: "while (a) {\n  continue\n}\n", Fix: true},
		{ID: "stmt-dowhile", Code: "do {\n  a()\n} while (b)\n", Fix: true},
		{ID: "stmt-dowhile-clean", Code: "do {\n  a();\n} while (b);\n"},
		{ID: "stmt-import", Code: "import a from 'a'\n", Fix: true},
		{ID: "stmt-import-clean", Code: "import a from 'a';\n"},
		{ID: "stmt-export-all", Code: "export * from 'a'\n", Fix: true},
		{ID: "stmt-export-named-decl", Code: "export const a = 1\n", Fix: true},
		{ID: "stmt-export-named-spec", Code: "const a = 1;\nexport { a }\n", Fix: true},
		{ID: "stmt-export-default-expr", Code: "export default foo\n", Fix: true},
		{ID: "stmt-export-default-class", Code: "export default class {}\n"},
		{ID: "stmt-export-default-fn", Code: "export default function () {}\n"},

		// ---- variable declarations inside for heads ----
		{ID: "for-init-var", Code: "for (var i = 0; i < 10; i++) {\n  a()\n}\n", Fix: true},
		{ID: "for-in-var", Code: "for (var k in o) {\n  a()\n}\n", Fix: true},
		{ID: "for-of-var", Code: "for (var v of o) {\n  a()\n}\n", Fix: true},
		{ID: "for-in-let-const", Code: "for (const v of o) {\n  a()\n}\n"},
		{ID: "for-empty-init", Code: "for (; i < 10; i++) {\n  a()\n}\n"},

		// ---- one-line blocks ----
		{ID: "oneline-block-default", Code: "if (foo) { bar() }\n", Fix: true},
		{ID: "oneline-block-semi", Code: "if (foo) { bar(); }\n"},
		{ID: "oneline-block-omit-true", Code: "if (foo) { bar(); }\n", Options: []any{"always", map[string]any{"omitLastInOneLineBlock": true}}, Fix: true},
		{ID: "oneline-block-omit-false", Code: "if (foo) { bar() }\n", Options: []any{"always", map[string]any{"omitLastInOneLineBlock": false}}},
		{ID: "oneline-block-omit-multiline", Code: "if (foo) {\n  bar();\n}\n", Options: []any{"always", map[string]any{"omitLastInOneLineBlock": true}}},
		{ID: "oneline-block-omit-two-stmts", Code: "if (foo) { bar(); baz() }\n", Options: []any{"always", map[string]any{"omitLastInOneLineBlock": true}}, Fix: true},
		{ID: "oneline-block-omit-nested", Code: "function f() { if (a) { b(); } }\n", Options: []any{"always", map[string]any{"omitLastInOneLineBlock": true}}, Fix: true},
		// NOTE (core gap, not a port gap): the StaticBlock branch of
		// isLastInOneLinerBlock cannot be exercised here. `class A { static { … } }`
		// makes the core's scope analysis recurse forever: eslint-go's
		// VisitorKeys omits "StaticBlock" (eslint-visitor-keys has
		// StaticBlock: ["body"]) and eslint-scope-go's "iteration" fallback
		// (visitor.go iterateKeys) returns every key, including the `parent`
		// link the traverser attached, so the walk cycles between a node and its
		// parent. Any rule crashes on this input, before create() runs.

		// ---- one-line class bodies ----
		{ID: "oneline-class-omit-true", Code: "class A { foo() {}; }\n", Options: []any{"always", map[string]any{"omitLastInOneLineClassBody": true}}, Fix: true},
		{ID: "oneline-class-omit-default", Code: "class A { foo() {}; }\n"},
		{ID: "oneline-class-omit-multiline", Code: "class A {\n  foo() {};\n}\n", Options: []any{"always", map[string]any{"omitLastInOneLineClassBody": true}}},
		{ID: "oneline-class-field", Code: "class A { a = 1; }\n", Options: []any{"always", map[string]any{"omitLastInOneLineClassBody": true}}, Fix: true},
		{ID: "oneline-class-empty", Code: "class A {}\n", Options: []any{"always", map[string]any{"omitLastInOneLineClassBody": true}}},

		// ---- class fields (always) ----
		{ID: "classfield-clean", Code: "class A {\n  a = 1;\n}\n"},
		{ID: "classfield-missing", Code: "class A {\n  a = 1\n}\n", Fix: true},
		{ID: "classfield-no-init", Code: "class A {\n  a\n}\n", Fix: true},
		{ID: "classfield-static", Code: "class A {\n  static a = 1\n}\n", Fix: true},
		{ID: "classfield-computed", Code: "class A {\n  [a] = 1\n}\n", Fix: true},
		{ID: "classfield-method", Code: "class A {\n  m() {}\n}\n"},

		// ---- "never" ----
		{ID: "never-clean", Code: "var a = 1\nvar b = 2\n", Options: []any{"never"}},
		{ID: "never-extra", Code: "var a = 1;\nvar b = 2;\n", Options: []any{"never"}, Fix: true},
		{ID: "never-extra-single", Code: "var a = 1;\n", Options: []any{"never"}, Fix: true},
		{ID: "never-double-semi", Code: "var a = 1;;\n", Options: []any{"never"}, Fix: true},
		{ID: "never-semi-before-brace", Code: "if (a) { b(); }\n", Options: []any{"never"}, Fix: true},
		{ID: "never-oneliner", Code: "var a = 1; var b = 2\n", Options: []any{"never"}},
		{ID: "never-class-field", Code: "class A {\n  a = 1;\n}\n", Options: []any{"never"}, Fix: true},
		{ID: "never-class-field-get", Code: "class A {\n  get\n  foo() {}\n}\n", Options: []any{"never"}},
		{ID: "never-class-field-static", Code: "class A {\n  static\n  foo() {}\n}\n", Options: []any{"never"}},
		{ID: "never-class-field-static-static", Code: "class A {\n  static static;\n}\n", Options: []any{"never"}, Fix: true},
		{ID: "never-class-field-follow-star", Code: "class A {\n  a;\n  *gen() {}\n}\n", Options: []any{"never"}, Fix: true},
		{ID: "never-class-field-follow-in", Code: "class A {\n  a;\n  in;\n}\n", Options: []any{"never"}},
		{ID: "never-class-field-follow-instanceof", Code: "class A {\n  a;\n  instanceof;\n}\n", Options: []any{"never"}},
		{ID: "never-import", Code: "import a from 'a';\n", Options: []any{"never"}, Fix: true},
		{ID: "never-export-default-class", Code: "export default class {}\n", Options: []any{"never"}},
		{ID: "never-arrow-block", Code: "const f = () => {\n  a();\n}\n", Options: []any{"never"}, Fix: true},
		{ID: "never-arrow-block-follow", Code: "const f = () => {\n  a();\n}\nfoo()\n", Options: []any{"never"}, Fix: true},

		// ---- beforeStatementContinuationChars ----
		{ID: "bsc-always-hazard", Code: "import a from 'a'\n(function() {})()\n", Options: []any{"never", map[string]any{"beforeStatementContinuationChars": "always"}}, Fix: true},
		{ID: "bsc-always-nohazard", Code: "var a = 1\nvar b = 2\n", Options: []any{"never", map[string]any{"beforeStatementContinuationChars": "always"}}},
		{ID: "bsc-any-hazard", Code: "import a from 'a'\n(function() {})()\n", Options: []any{"never", map[string]any{"beforeStatementContinuationChars": "any"}}},
		{ID: "bsc-never-hazard", Code: "import a from 'a'\n(function() {})()\n", Options: []any{"never", map[string]any{"beforeStatementContinuationChars": "never"}}},
		{ID: "bsc-never-return", Code: "function f() {\n  return\n  x()\n}\n", Options: []any{"never", map[string]any{"beforeStatementContinuationChars": "never"}}},
		{ID: "bsc-always-classfield", Code: "class A {\n  a\n  [b]() {}\n}\n", Options: []any{"never", map[string]any{"beforeStatementContinuationChars": "always"}}},
		{ID: "bsc-default", Code: "var a = 1\nvar b = 2\n", Options: []any{"never"}},

		// ---- ASI hazard tokens before ----
		{ID: "hazard-bracket", Code: "var a = 1\n;[1, 2].forEach(f)\n", Options: []any{"never"}},
		{ID: "hazard-paren", Code: "var a = 1\n;(function() {})()\n", Options: []any{"never"}},
		{ID: "hazard-plus", Code: "var a = 1\n;+b\n", Options: []any{"never"}},
		{ID: "hazard-minus", Code: "var a = 1\n;-b\n", Options: []any{"never"}},
		{ID: "hazard-slash", Code: "var a = 1\n;/re/.test(b)\n", Options: []any{"never"}},
		{ID: "hazard-template", Code: "var a = 1\n;`t`\n", Options: []any{"never"}},
		{ID: "hazard-increment", Code: "var a = 1\n;++b\n", Options: []any{"never"}},
		{ID: "hazard-decrement", Code: "var a = 1\n;--b\n", Options: []any{"never"}},

		// ---- comments / templates ----
		{ID: "comment-between", Code: "var a = 1 // c\nvar b = 2\n", Fix: true},
		{ID: "block-comment-between", Code: "var a = 1 /* c */\nvar b = 2\n", Fix: true},
		{ID: "comment-only", Code: "// just a comment\n"},
		{ID: "template-stmt", Code: "const t = `a\nb`\n", Fix: true},
		{ID: "template-in-tag", Code: "tag`a`\n", Fix: true},
		{ID: "directive", Code: "'use strict'\nvar a = 1\n", Fix: true},

		// ---- non-ASCII ----
		{ID: "nonascii-ident", Code: "var caf\u00e9 = 1\n", Fix: true},
		{ID: "nonascii-string", Code: "var s = \"\u4f60\u597d\"\n", Fix: true},
		{ID: "nonascii-comment", Code: "var s = \"\u4f60\u597d\" // \u6ce8\u91ca\n", Fix: true},
		{ID: "nonascii-emoji", Code: "var s = \"\U0001f600\"\n", Fix: true},
		{ID: "nonascii-never", Code: "var s = \"\u4f60\u597d\";\n", Options: []any{"never"}, Fix: true},
		{ID: "nonascii-oneline-block", Code: "if (a) { b(); } // \u4f60\u597d\n", Options: []any{"always", map[string]any{"omitLastInOneLineBlock": true}}, Fix: true},
		{ID: "nonascii-crlf", Code: "var s = \"\u4f60\u597d\"\r\nvar t = 1\r\n", Fix: true},
	})
}
