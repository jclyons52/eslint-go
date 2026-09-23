package novar

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// --- basic reports + fixes -------------------------------------------
		{ID: "basic", Code: "var x = 1;", Fix: true},
		{ID: "no-init", Code: "var x;", Fix: true},
		{ID: "multiple-declarators", Code: "var a = 1, b = 2;", Fix: true},
		{ID: "no-semicolon", Code: "var x = 1\n", Fix: true},
		{ID: "clean-let-const", Code: "let x = 1;\nconst y = 2;\n"},
		{ID: "clean-empty", Code: ""},
		{ID: "string-and-number", Code: "var s = \"a\", n = 0;", Fix: true},
		{ID: "template-literal", Code: "var t = `x${1}y`;", Fix: true},

		// --- severities ------------------------------------------------------
		{ID: "warn-severity", Code: "var x = 1;", Config: map[string]any{
			"rules": map[string]any{Name: "warn"},
		}},
		{ID: "off-severity", Code: "var x = 1;", Config: map[string]any{
			"rules": map[string]any{Name: "off"},
		}},

		// --- scopes ----------------------------------------------------------
		{ID: "nested-block", Code: "if (a) { var x = 1; }", Fix: true},
		{ID: "inside-function", Code: "function f() { var x = 1; return x; }", Fix: true},
		{ID: "closure", Code: "function f() { var x = 1; return function () { return x; }; }", Fix: true},
		{ID: "arrow-closure", Code: "const f = () => { var x = 1; return () => x; };", Fix: true},
		{ID: "class-method", Code: "class C { m() { var x = 1; return x; } }", Fix: true},
		{ID: "static-block", Code: "class C { static { var x = 1; } }", Fix: true},
		{ID: "used-from-outside", Code: "function f() { if (true) { var x = 1; } return x; }", Fix: true},
		{ID: "shadow-inner-var", Code: "function f() { var x = 1; { var x = 2; } return x; }", Fix: true},

		// --- loops -----------------------------------------------------------
		{ID: "for-init", Code: "for (var i = 0; i < 10; i++) {}", Fix: true},
		{ID: "for-init-no-test", Code: "for (var i = 0; ; i++) { break; }", Fix: true},
		{ID: "for-of", Code: "for (var x of list) { use(x); }", Fix: true},
		{ID: "for-in", Code: "for (var k in obj) { use(k); }", Fix: true},
		{ID: "for-of-array-pattern", Code: "for (var [a, b] of list) { use(a, b); }", Fix: true},
		{ID: "while-no-init", Code: "while (a) { var x; use(x); }", Fix: true},
		{ID: "do-while-no-init", Code: "do { var x; use(x); } while (a);", Fix: true},
		{ID: "loop-closure", Code: "for (var i = 0; i < 3; i++) { setTimeout(function () { console.log(i); }); }", Fix: true},
		{ID: "loop-closure-arrow", Code: "for (var i = 0; i < 3; i++) { f(() => i); }", Fix: true},
		{ID: "for-of-closure", Code: "for (var x of list) { f(function () { return x; }); }", Fix: true},
		{ID: "nested-loop-closure", Code: "for (var i = 0; i < 3; i++) { for (var j = 0; j < 3; j++) { f(() => i + j); } }", Fix: true},

		// --- redeclaration / switch / statement position ---------------------
		{ID: "redeclared", Code: "var x = 1;\nvar x = 2;\nuse(x);", Fix: true},
		{ID: "redeclared-let", Code: "var x = 1;\nlet y = 2;\nvar y = 3;\nuse(x, y);", Fix: true},
		{ID: "switch-case", Code: "switch (a) { case 1: var x = 1; use(x); }", Fix: true},
		{ID: "switch-case-default", Code: "switch (a) { default: var x = 1; use(x); }", Fix: true},
		{ID: "statement-position", Code: "if (a) var x = 1;", Fix: true},
		{ID: "label-statement", Code: "loop: for (;;) { var x = 1; break loop; }", Fix: true},

		// --- TDZ -------------------------------------------------------------
		{ID: "self-reference", Code: "var x = x;", Fix: true},
		{ID: "self-reference-array", Code: "var [a = a] = list;", Fix: true},
		{ID: "self-reference-object", Code: "var { a = a } = obj;", Fix: true},
		{ID: "reference-before-declarator", Code: "var a = b, b = 1;", Fix: true},
		{ID: "self-reference-function-init", Code: "var f = function () { return f; };", Fix: true},
		{ID: "self-reference-arrow-init", Code: "var f = () => f;", Fix: true},
		{ID: "for-in-self-reference", Code: "for (var a in a) {}", Fix: true},
		{ID: "for-of-self-reference", Code: "for (var a of a) {}", Fix: true},

		// --- member / call / export shapes -----------------------------------
		{ID: "member-expression", Code: "var o = {};\no.x = 1;\nuse(o.x);", Fix: true},
		{ID: "call-expression", Code: "var f = g();", Fix: true},
		{ID: "export-var", Code: "export var x = 1;", Fix: true},
		{ID: "export-default-var", Code: "export default (function () { var x = 1; return x; });", Fix: true},
		{ID: "nested-functions-multiple", Code: "function a() { var x = 1; function b() { var y = 2; return x + y; } return b; }", Fix: true},
		{ID: "comments", Code: "// leading\nvar x = 1; // trailing\n/* block */\nuse(x);", Fix: true},
		{ID: "non-ascii", Code: "var é = 1;\nuse(é);", Fix: true},

		// --- parse errors ----------------------------------------------------
		{ID: "fatal-parse-error", Code: "var x = ;", Fix: true},
	})
}
