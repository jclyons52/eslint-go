package preferconst

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// --- default behaviour -------------------------------------------------
		{ID: "basic", Code: "let a = 1; a;", Fix: true},
		{ID: "const-untouched", Code: "const a = 1; a;"},
		{ID: "var-untouched", Code: "var a = 1; a;"},
		{ID: "no-init-no-read", Code: "let a;"},
		{ID: "reassigned", Code: "let a = 1; a = 2; a;"},
		{ID: "reassigned-op", Code: "let a = 1; a += 1;"},
		{ID: "reassigned-update", Code: "let a = 1; a++;"},
		{ID: "assigned-once", Code: "let a; a = 1;", Fix: true},
		{ID: "assigned-once-read", Code: "let a; a = 1; a;", Fix: true},
		{ID: "read-before-assign", Code: "let a; a; a = 1;"},
		{ID: "ignore-read-before-assign", Code: "let a; a; a = 1;",
			Options: []any{map[string]any{"ignoreReadBeforeAssign": true}}},
		{ID: "ignore-read-before-assign-off", Code: "let a; a; a = 1;",
			Options: []any{map[string]any{"ignoreReadBeforeAssign": false}}},
		{ID: "warn-severity", Code: "let a = 1; a;",
			Config: map[string]any{"rules": map[string]any{"prefer-const": "warn"}}},

		// --- scopes ------------------------------------------------------------
		{ID: "block-scope", Code: "{ let a = 1; a; }", Fix: true},
		{ID: "function-scope", Code: "function f() { let a = 1; return a; }", Fix: true},
		{ID: "arrow-body", Code: "const f = () => { let a = 1; return a; };", Fix: true},
		{ID: "nested-fn-reassign", Code: "let a = 1; function f() { a = 2; } a;"},
		{ID: "nested-fn-read", Code: "let a = 1; function f() { return a; } f();"},
		{ID: "assign-in-outer-scope-only", Code: "let a; function f() { a = 1; }"},
		{ID: "catch-param-scope", Code: "try { throw 1; } catch (e) { let a = e; a; }"},
		// StaticBlock: the core gap that made any `static { … }` crash scope
		// analysis is fixed (visitor keys + fallback), so this is covered again.
		{ID: "static-block-let", Code: "class C { static { let a = 1; a; } }", Fix: true},
		{ID: "class-field-initializer", Code: "class C { x = 1; m() { let a = 1; return a; } }", Fix: true},
		{ID: "class-declaration-name", Code: "class C { m() { return C; } } let a = 1; a;", Fix: true},
		{ID: "switch-case", Code: "switch (x) { case 1: { let a = 1; a; } }", Fix: true},

		// --- loops -------------------------------------------------------------
		{ID: "for-init-excluded", Code: "for (let i = 0; i < 3; i++) {}"},
		{ID: "for-in", Code: "for (let k in obj) { k; }", Fix: true},
		{ID: "for-of", Code: "for (let v of arr) { v; }", Fix: true},
		{ID: "for-of-destructure", Code: "for (let { a, b } of arr) { a; b; }", Fix: true},
		{ID: "for-in-destructure", Code: "for (let [a, b] in obj) { a; b; }", Fix: true},

		// --- destructuring -----------------------------------------------------
		{ID: "array-pattern", Code: "let [a, b] = arr; a; b;", Fix: true},
		{ID: "object-pattern", Code: "let { a, b } = obj; a; b;", Fix: true},
		{ID: "object-pattern-renamed", Code: "let { a: x, b: y } = obj; x; y;", Fix: true},
		{ID: "destructuring-mixed", Code: "let { a, b } = obj; a; b = 1;"},
		{ID: "destructuring-all-ok", Code: "let { a, b } = obj; a; b;",
			Options: []any{map[string]any{"destructuring": "all"}}, Fix: true},
		{ID: "destructuring-all-mixed", Code: "let { a, b } = obj; a; b = 1;",
			Options: []any{map[string]any{"destructuring": "all"}}},
		{ID: "destructuring-any-option", Code: "let { a, b } = obj; a; b = 1;",
			Options: []any{map[string]any{"destructuring": "any"}}},
		{ID: "destructuring-nested", Code: "let { a: { b } } = obj; b;", Fix: true},
		{ID: "destructuring-array-hole", Code: "let [a, , b] = arr; a; b;", Fix: true},
		{ID: "destructuring-default", Code: "let { a = 1 } = obj; a;", Fix: true},
		{ID: "destructuring-assignment", Code: "let a; ({ a } = obj);"},
		{ID: "destructuring-member", Code: "let a; ({ a: obj.x } = src);"},

		// --- multiple declarations in one statement ---------------------------
		{ID: "multi-declarator-all", Code: "let a = 1, b = 2; a; b;", Fix: true},
		{ID: "multi-declarator-partial", Code: "let a = 1, b = 2; a; b = 3;"},
		{ID: "multi-declarator-partial-fix", Code: "let a = 1, b = 2; a = 3; b;"},
		{ID: "multi-declarator-uninitialized", Code: "let a = 1, b; a; b;"},

		// --- edge tokens / text -------------------------------------------------
		{ID: "comment-mentions-name", Code: "// let a = 1;\nlet a = 1; a;", Fix: true},
		{ID: "string-mentions-name", Code: "let a = 1; const s = 'let a'; a;", Fix: true},
		{ID: "template-literal", Code: "let a = 1; const s = `x${a}`;", Fix: true},
		{ID: "non-ascii", Code: "let café = 1; café;", Fix: true},
		{ID: "crlf", Code: "let a = 1;\r\na;\r\n", Fix: true},
		{ID: "clean-file", Code: "const a = 1;\nfunction f() { return a; }\nf();"},

		// --- parameters / function forms ---------------------------------------
		{ID: "param-reassigned", Code: "function f(a) { a = 1; }"},
		{ID: "param-never-assigned", Code: "function f(a) { return a; }"},
		{ID: "destructuring-param", Code: "function f({ a }) { a; }"},
		{ID: "function-expression-name", Code: "const f = function g() { let a = 1; return a; };", Fix: true},
	})
}
