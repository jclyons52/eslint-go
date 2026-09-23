package noshadow

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

func opts(m map[string]any) []any { return []any{m} }

func globals(m map[string]any) map[string]any {
	return map[string]any{"globals": m}
}

// TestParity runs the corpus against the real ESLint oracle.
//
// Static blocks are covered: the core's VisitorKeys now carries "StaticBlock"
// and eslint-scope-go's iteration fallback no longer walks the `parent` link,
// so a `class C { static { … } }` no longer overflows scope analysis.
func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		{ID: "static-block-shadow", Code: "let a = 1; class C { static { let a = 2; a; } }"},
		// Shadowed function scope.
		{ID: "shadow-global-var", Code: "var a = 1; function f() { var a = 2; }", Config: cfg(modulePO, 2, nil)},
		{ID: "shadow-global-let", Code: "let a = 1; function f() { let a = 2; }", Config: cfg(modulePO, 2, nil)},
		{ID: "shadow-param", Code: "var a = 1; function f(a) {}", Config: cfg(modulePO, 2, nil)},
		{ID: "shadow-nested-functions", Code: "function f() { var a; function g() { var a; } }", Config: cfg(modulePO, 2, nil)},
		{ID: "shadow-function-decl", Code: "function a() {} function f() { function a() {} }", Config: cfg(modulePO, 2, nil)},
		{ID: "shadow-import", Code: "import { a } from \"m\"; function f() { var a; }", Config: cfg(modulePO, 2, nil)},
		{ID: "shadow-global-in-arrow", Code: "var a = 1; const f = () => { var a = 2; };", Config: cfg(modulePO, 2, nil)},

		// Shadowed block scope.
		{ID: "shadow-block", Code: "function f() { let a = 1; { let a = 2; } }", Config: cfg(modulePO, 2, nil)},
		{ID: "shadow-block-in-block", Code: "{ let a = 1; { let a = 2; } }", Config: cfg(modulePO, 2, nil)},
		{ID: "shadow-for-of", Code: "let x = 1; for (const x of y) {}", Config: cfg(modulePO, 2, nil)},
		{ID: "shadow-switch-case", Code: "let a = 1; switch (x) { case 1: let a = 2; break; }", Config: cfg(modulePO, 2, nil)},

		// Shadowed catch scope.
		{ID: "shadow-catch-param", Code: "var e = 1; try {} catch (e) {}", Config: cfg(modulePO, 2, nil)},
		{ID: "shadow-catch-block", Code: "let e = 1; try {} catch (err) { let e = 2; }", Config: cfg(modulePO, 2, nil)},

		// Shadowed class scope / class names.
		{ID: "shadow-class-in-function", Code: "class A {} function f() { class A {} }", Config: cfg(modulePO, 2, nil)},
		{ID: "shadow-class-in-block", Code: "class A {} { class A {} }", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-class-own-name", Code: "class A {}", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-class-method", Code: "class A { m() { return 1; } }", Config: cfg(modulePO, 2, nil)},

		// Clean: no shadowing.
		{ID: "clean-different-names", Code: "function f() { var a; } function g() { var b; }", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-sibling-blocks", Code: "{ let a = 1; } { let a = 2; }", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-param-and-var-same-scope", Code: "function f(a) { var a = 1; }", Config: cfg(modulePO, 2, nil)},
		{ID: "clean-inner-declared-after-use", Code: "function f() { var a; } var b = 1;", Config: cfg(modulePO, 2, nil)},
		// An eslint-scope *implicit* global (sloppy-mode assignment to an
		// undeclared name) is not a shadowing target: it lives in
		// `scope.implicit`, which getVariableByName does not consult.
		{ID: "clean-implicit-global", Code: "a = 1; function f() { var a = 2; }", Config: cfg(scriptPO, 2, nil)},
		{ID: "clean-implicit-global-builtin", Code: "a = 1; function f() { var a = 2; }", Config: cfg(scriptPO, 2, nil, opts(map[string]any{"builtinGlobals": true})...)},

		// isOnInitializer: `var a = function a() {}` and friends.
		{ID: "on-initializer-fn-expr-name", Code: "var a = function a() {};", Config: cfg(modulePO, 2, nil)},
		{ID: "on-initializer-class-expr-name", Code: "var a = 1; (class a {});", Config: cfg(modulePO, 2, nil)},
		{ID: "on-initializer-fn-param", Code: "var a = function(a) {};", Config: cfg(modulePO, 2, nil)},
		{ID: "on-initializer-inner-fn-decl", Code: "var a = function() { function a() {} };", Config: cfg(modulePO, 2, nil)},

		// hoist.
		{ID: "hoist-default-fn-outer", Code: "function f() { function g() {} } function g() {}", Config: cfg(modulePO, 2, nil)},
		{ID: "hoist-never-fn-outer", Code: "function f() { function g() {} } function g() {}", Config: cfg(modulePO, 2, nil, opts(map[string]any{"hoist": "never"})...)},
		{ID: "hoist-all-fn-outer", Code: "function f() { function g() {} } function g() {}", Config: cfg(modulePO, 2, nil, opts(map[string]any{"hoist": "all"})...)},
		{ID: "hoist-default-var-outer", Code: "function f() { var g; } var g = 1;", Config: cfg(modulePO, 2, nil)},
		{ID: "hoist-all-var-outer", Code: "function f() { var g; } var g = 1;", Config: cfg(modulePO, 2, nil, opts(map[string]any{"hoist": "all"})...)},
		{ID: "hoist-never-var-outer", Code: "function f() { var g; } var g = 1;", Config: cfg(modulePO, 2, nil, opts(map[string]any{"hoist": "never"})...)},

		// allow.
		{ID: "allow-hit", Code: "var a = 1; function f() { var a = 2; }", Config: cfg(modulePO, 2, nil, opts(map[string]any{"allow": []any{"a"}})...)},
		{ID: "allow-miss", Code: "var a = 1; function f() { var a = 2; }", Config: cfg(modulePO, 2, nil, opts(map[string]any{"allow": []any{"b"}})...)},
		{ID: "allow-multiple", Code: "var a = 1; var b = 1; function f() { var a = 2; var b = 2; }", Config: cfg(modulePO, 2, nil, opts(map[string]any{"allow": []any{"a", "b"}})...)},
		{ID: "allow-partial", Code: "var a = 1; var b = 1; function f() { var a = 2; var b = 2; }", Config: cfg(modulePO, 2, nil, opts(map[string]any{"allow": []any{"a"}})...)},
		{ID: "allow-empty-list", Code: "var a = 1; function f() { var a = 2; }", Config: cfg(modulePO, 2, nil, opts(map[string]any{"allow": []any{}})...)},
		{ID: "empty-options-object", Code: "var a = 1; function f() { var a = 2; }", Config: cfg(modulePO, 2, nil, map[string]any{})},

		// ignoreOnInitialization.
		{ID: "ignore-on-init-on", Code: "var a = foo(a => a);", Config: cfg(modulePO, 2, nil, opts(map[string]any{"ignoreOnInitialization": true})...)},
		{ID: "ignore-on-init-off", Code: "var a = foo(a => a);", Config: cfg(modulePO, 2, nil)},
		{ID: "ignore-on-init-unrelated", Code: "var a = 1; function f() { var a = 2; }", Config: cfg(modulePO, 2, nil, opts(map[string]any{"ignoreOnInitialization": true})...)},
		{ID: "ignore-on-init-param", Code: "var a = foo(a => a);", Config: cfg(modulePO, 2, nil, opts(map[string]any{"ignoreOnInitialization": false})...)},

		// builtinGlobals.
		{ID: "builtin-on", Code: "function f() { var foo = 1; }", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}), opts(map[string]any{"builtinGlobals": true})...)},
		{ID: "builtin-off", Code: "function f() { var foo = 1; }", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}), opts(map[string]any{"builtinGlobals": false})...)},
		{ID: "builtin-default", Code: "function f() { var foo = 1; }", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}))},
		{ID: "builtin-writable", Code: "function f() { var foo = 1; }", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "writable"}), opts(map[string]any{"builtinGlobals": true})...)},
		{ID: "builtin-env", Code: "function f() { var Map = 1; }", Config: cfg(scriptPO, 2, map[string]any{"env": map[string]any{"es6": true}}, opts(map[string]any{"builtinGlobals": true})...)},
		{ID: "builtin-env-readonly-no-decl", Code: "function f() { var top = 1; }", Config: cfg(scriptPO, 2, map[string]any{"env": map[string]any{"browser": true}}, opts(map[string]any{"builtinGlobals": true})...)},
		{ID: "builtin-env-readonly-with-decl", Code: "var top = 1; function f() { var top = 2; }", Config: cfg(scriptPO, 2, map[string]any{"env": map[string]any{"browser": true}}, opts(map[string]any{"builtinGlobals": true})...)},
		{ID: "builtin-declared-global-keeps-line", Code: "var foo = 1; function f() { var foo = 2; }", Config: cfg(scriptPO, 2, globals(map[string]any{"foo": "readonly"}), opts(map[string]any{"builtinGlobals": true})...)},

		// Severities.
		{ID: "warn-severity", Code: "var a = 1; function f() { var a = 2; }", Config: cfg(modulePO, "warn", nil)},
		{ID: "off-severity", Code: "var a = 1; function f() { var a = 2; }", Config: cfg(modulePO, "off", nil)},
	})
}
