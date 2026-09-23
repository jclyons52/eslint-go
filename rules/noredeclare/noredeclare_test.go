package noredeclare

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// The harness only sends Case.ECMAVersion/SourceType to the JS oracle, so the
// Go side's parserOptions have to be spelled out in the case config — and this
// rule is scope-sensitive: ESLint 8 defaults a config without parserOptions to
// ecmaVersion 5, where eslint-scope creates no block scopes at all.
var (
	modulePO = map[string]any{"sourceType": "module", "ecmaVersion": 2022}
	scriptPO = map[string]any{"sourceType": "script", "ecmaVersion": 2022}
)

// cfg builds a case config: parserOptions, the rule at severity sev with opts,
// and any extra keys (globals/env).
func cfg(po map[string]any, sev any, extra map[string]any, opts ...any) map[string]any {
	value := []any{sev}
	value = append(value, opts...)
	out := map[string]any{
		"parserOptions": po,
		"rules":         map[string]any{Name: value},
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func globals(m map[string]any) map[string]any {
	return map[string]any{"globals": m}
}

// TestParity runs the corpus against the real ESLint oracle.
//
// Note on sourceType: duplicate `function`/`let`/`class` declarations in one
// scope are early errors in module (strict) mode and the Go parser does not
// implement those early errors, so the redeclaration cases the rule exists to
// flag run in script mode — exactly as ESLint's own
// tests/lib/rules/no-redeclare.js does.
//
// Note on static blocks: `class C { static { … } }` is not exercised. The
// core's VisitorKeys omits "StaticBlock", so eslint-scope-go's "iteration"
// fallback walks the node's `parent` link and recurses until the stack
// overflows — a pre-existing, documented core gap (see the package comment in
// rules/nounusedvars/nounusedvars.go), unrelated to this rule.
func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Same scope, same declaration type → "redeclared".
		{ID: "var-var", Code: "var a = 3; var a = 10;", Config: cfg(modulePO, 2, nil)},
		{ID: "var-var-var", Code: "var a = 3; var a = 10; var a = 15;", Config: cfg(modulePO, 2, nil)},
		{ID: "var-in-block", Code: "var a = 1; { var a = 2; }", Config: cfg(modulePO, 2, nil)},
		{ID: "for-var", Code: "for (var i = 0; i < 10; i++) {} for (var i = 0; i < 10; i++) {}", Config: cfg(modulePO, 2, nil)},
		{ID: "arrow-param-var", Code: "const f = (a) => { var a = 1; };", Config: cfg(modulePO, 2, nil)},
		{ID: "module-var-var", Code: "var a = 1;\nvar a = 2;", Config: cfg(modulePO, 2, nil)},

		// Script mode: `function` redeclarations (legal in sloppy mode).
		{ID: "var-var-script", Code: "var a = 3; var a = 10;", Config: cfg(scriptPO, 2, nil)},
		{ID: "func-func", Code: "function a() {} function a() {}", Config: cfg(scriptPO, 2, nil)},
		{ID: "var-func", Code: "var a = 3; function a() {}", Config: cfg(scriptPO, 2, nil)},
		{ID: "func-var", Code: "function a() {} var a = 3;", Config: cfg(scriptPO, 2, nil)},

		// Duplicate `let`/`class` bindings in one scope are ES early errors:
		// acorn rejects them at parse time, so the oracle never reaches the
		// rule. The Go parser does not implement early errors, so these are a
		// documented parser divergence and cannot be parity cases. The
		// block/for/switch/class scope machinery is covered by the clean
		// cross-scope cases below instead.
		{ID: "clean-let-two-blocks", Code: "{ let a = 1; } { let a = 2; }", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-let-two-functions", Code: "function f() { let a = 1; } function g() { let a = 2; }", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-for-of-let", Code: "for (let a of x) {} for (let a of x) {}", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-for-in-let", Code: "for (let a in x) {} for (let a in x) {}", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-switch-nested-blocks", Code: "switch (x) { case 1: { let a; break; } case 2: { let a; break; } }", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-class-two-functions", Code: "function f() { class A {} } function g() { class A {} }", Config: cfg(modulePO, 2, nil)},

		// Function scopes.
		{ID: "param-var", Code: "function f(a) { var a = 1; }", Config: cfg(modulePO, 2, nil)},
		{ID: "func-expr-param-var", Code: "(function (a) { var a = 1; });", Config: cfg(modulePO, 2, nil)},
		{ID: "inner-var-var", Code: "function f() { var a; var a; }", Config: cfg(modulePO, 2, nil)},
		{ID: "inner-var-func", Code: "function f() { var a; function a() {} }", Config: cfg(modulePO, 2, nil)},
		{ID: "catch-var", Code: "try {} catch (e) { var e = 1; }", Config: cfg(modulePO, 2, nil)},

		// Clean: different scopes never collide.
		{ID: "clean-single", Code: "var a = 1;", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-nested", Code: "var a = 1; { let a = 2; }", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-two-functions", Code: "function f() { var a = 1; } function g() { var a = 2; }", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-let-block", Code: "let a = 1; { let a = 2; }", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-import", Code: "import a from \"m\"; a();", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-param-shadow", Code: "function f(a) { function g(a) { return a; } }", Config: cfg(modulePO, 2, nil)},

		// builtinGlobals (default true): a configured global counts as an
		// implicit first declaration.
		{ID: "builtin-default-on", Code: "var foo = 1;", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}))},
		{ID: "builtin-explicit-true", Code: "var foo = 1;", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}), map[string]any{"builtinGlobals": true})},
		{ID: "builtin-explicit-false", Code: "var foo = 1;", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}), map[string]any{"builtinGlobals": false})},
		{ID: "builtin-writable", Code: "var foo = 1;", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "writable"}))},
		{ID: "builtin-no-declaration", Code: "foo();", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}))},
		{ID: "builtin-function-decl", Code: "function foo() {}", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}))},
		{ID: "builtin-let-decl", Code: "let foo = 1;", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}))},
		{ID: "builtin-two-decls", Code: "var foo = 1; var foo = 2;", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}))},
		{ID: "builtin-other-name", Code: "var bar = 1;", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}))},
		{ID: "builtin-empty-options-object", Code: "var foo = 1;", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}), map[string]any{})},
		{ID: "builtin-env-new-global", Code: "function f() { var Map = 1; }", Config: cfg(modulePO, 2, map[string]any{"env": map[string]any{"es6": true}})},
		// The canonical builtinGlobals case (ESLint's own test suite): `top` is an
		// env-configured readonly global *and* has a syntax declaration, so the
		// configured global counts as a first declaration.
		{ID: "builtin-env-readonly-decl", Code: "var top = 0;", Config: cfg(scriptPO, 2, map[string]any{"env": map[string]any{"browser": true}})},
		// An ecmaVersion builtin redeclared by syntax (the ES5 base set).
		{ID: "builtin-base-redeclared", Code: "var Array = 1;", Config: cfg(scriptPO, 2, nil)},
		// writable config global redeclared by syntax.
		{ID: "builtin-writable-redeclared", Code: "var foo = 1;", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "writable"}))},

		// Severities.
		{ID: "warn-severity", Code: "var a = 1; var a = 2;", Config: cfg(modulePO, "warn", nil)},
		{ID: "off-severity", Code: "var a = 1; var a = 2;", Config: cfg(modulePO, "off", nil)},
	})
}
