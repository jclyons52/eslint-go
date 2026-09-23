package noeval

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// mod builds a case config. The harness forwards a case's SourceType /
// ECMAVersion only to the JS oracle, so the Go side's parserOptions have to be
// spelled out in the config for the two sides to agree: the oracle's default is
// sourceType "module" / ecmaVersion 2022, while eslint-go's Config defaults to
// sourceType "script".
func mod(extra map[string]any) map[string]any {
	cfg := map[string]any{
		"parserOptions": map[string]any{"sourceType": "module", "ecmaVersion": 2022},
	}
	for k, v := range extra {
		cfg[k] = v
	}
	return cfg
}

// globals declares names as read-only globals so the rule's
// candidatesOfGlobalObject lookup finds a variable in the global scope.
func globals(names ...string) map[string]any {
	m := map[string]any{}
	for _, n := range names {
		m[n] = "readonly"
	}
	return m
}

// script is mod for sourceType "script": the only way to reach the rule's
// `this.eval` analysis. Module code is always strict, so `enterThisScope` marks
// every frame initialized and `isDefaultThisBinding` is never consulted; in a
// sloppy script it is, and the oracle then reports `this.eval` wherever the
// function is the default `this` binding.
func script(extra map[string]any) map[string]any {
	cfg := map[string]any{
		"parserOptions": map[string]any{"sourceType": "script", "ecmaVersion": 2022},
	}
	for k, v := range extra {
		cfg[k] = v
	}
	return cfg
}

func TestParity(t *testing.T) {
	cases := []ruletest.Case{
		// --- access via the global object -----------------------------------
		{ID: "window-eval", Code: `window.eval("var a = 0");`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "window-window-eval", Code: `window.window.eval("var a = 0");`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "window-window-window-eval", Code: `window.window.window.eval("var a = 0");`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "global-eval", Code: `global.eval("var a = 0");`,
			Config: mod(map[string]any{"globals": globals("global")})},
		{ID: "globalThis-eval", Code: `globalThis.eval("var a = 0");`,
			Config: mod(map[string]any{"globals": globals("globalThis")})},
		{ID: "window-computed-string", Code: `window["eval"]("var a = 0");`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "window-computed-template", Code: "window[`eval`](\"var a = 0\");",
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "window-computed-dynamic", Code: `var k = "eval"; window[k]("var a = 0");`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "window-eval-not-called", Code: `var f = window.eval;`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "window-eval-arg", Code: `foo(window.eval);`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "window-optional-member", Code: `window?.eval("var a = 0");`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "window-eval-optional-call", Code: `window.eval?.("var a = 0");`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "window-other-property", Code: `window.other("var a = 0");`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "window-undeclared", Code: `window.eval("var a = 0");`, Config: mod(nil)},
		{ID: "window-shadowed-by-param", Code: `function f(window) { window.eval("var a = 0"); }`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "window-shadowed-by-var", Code: `function f() { var window = {}; window.eval("var a = 0"); }`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "all-global-objects", Code: `global.eval("a"); window.eval("b"); globalThis.eval("c");`,
			Config: mod(map[string]any{"globals": globals("global", "window", "globalThis")})},
		{ID: "global-object-severity-warn", Code: `window.eval("var a = 0");`,
			Config: mod(map[string]any{
				"globals": globals("window"),
				"rules":   map[string]any{Name: 1},
			})},
		{ID: "global-object-nested", Code: `function f() { if (true) { window.eval("var a = 0"); } }`,
			Config: mod(map[string]any{"globals": globals("window")})},
		{ID: "global-object-multi-line", Code: "window\n    .eval(\"var a = 0\");",
			Config: mod(map[string]any{"globals": globals("window")})},

		// --- allowIndirect: only a bare direct `eval(...)` counts -----------
		// (the bare `eval` identifier cannot be parsed by the Go parser — see
		// the blocked-cases list at the bottom of this file)
		{ID: "allow-indirect-window", Code: `window.eval("var a = 0");`,
			Options: []any{map[string]any{"allowIndirect": true}},
			Config:  mod(map[string]any{"globals": globals("window")})},
		{ID: "allow-indirect-window-computed", Code: `window["eval"]("var a = 0");`,
			Options: []any{map[string]any{"allowIndirect": true}},
			Config:  mod(map[string]any{"globals": globals("window")})},
		{ID: "allow-indirect-clean", Code: `var a = 1;`,
			Options: []any{map[string]any{"allowIndirect": true}},
			Config:  mod(nil)},

		// --- `this.eval` ----------------------------------------------------
		// Module code is always strict, so `this` is never the global object:
		// the rule reports none of these, but every frame kind the rule tracks
		// (function, method, constructor, arrow, field) is traversed and the
		// report path stays silent. The sloppy-script block below is where the
		// same shapes actually report.
		{ID: "this-top-level", Code: `this.eval("var a = 0");`, Config: mod(nil)},
		{ID: "this-function", Code: `function foo() { this.eval("var a = 0"); }`, Config: mod(nil)},
		{ID: "this-function-expression", Code: `var foo = function () { this.eval("var a = 0"); };`, Config: mod(nil)},
		{ID: "this-named-function-expression", Code: `var foo = function bar() { this.eval("var a = 0"); };`, Config: mod(nil)},
		{ID: "this-arrow-top-level", Code: `var f = () => this.eval("var a = 0");`, Config: mod(nil)},
		{ID: "this-arrow-in-function", Code: `function f() { var g = () => this.eval("var a = 0"); return g; }`, Config: mod(nil)},
		{ID: "this-object-method", Code: `var obj = { foo() { this.eval("var a = 0"); } };`, Config: mod(nil)},
		{ID: "this-object-function", Code: `var obj = { foo: function () { this.eval("var a = 0"); } };`, Config: mod(nil)},
		{ID: "this-class-method", Code: `class A { foo() { this.eval("var a = 0"); } }`, Config: mod(nil)},
		{ID: "this-class-static-method", Code: `class A { static foo() { this.eval("var a = 0"); } }`, Config: mod(nil)},
		{ID: "this-class-constructor", Code: `class A { constructor() { this.eval("var a = 0"); } }`, Config: mod(nil)},
		{ID: "this-class-getter", Code: `class A { get foo() { return this.eval("var a = 0"); } }`, Config: mod(nil)},
		{ID: "this-class-field", Code: `class A { foo = this.eval("var a = 0"); }`, Config: mod(nil)},
		{ID: "this-computed-member", Code: `function f() { this["eval"]("var a = 0"); }`, Config: mod(nil)},
		{ID: "this-nested-function", Code: `function f() { function g() { this.eval("var a = 0"); } return g; }`, Config: mod(nil)},
		{ID: "this-other-property", Code: `function f() { this.other("var a = 0"); }`, Config: mod(nil)},
		{ID: "this-jsdoc-this-tag", Code: "/**\n * @this {Object}\n */\nfunction f() { this.eval(\"var a = 0\"); }", Config: mod(nil)},
		{ID: "this-iife-return", Code: `var obj = {}; obj.foo = (function () { return function () { this.eval("var a = 0"); }; })();`, Config: mod(nil)},
		{ID: "this-upper-case-name", Code: `function Foo() { this.eval("var a = 0"); }`, Config: mod(nil)},
		{ID: "this-bound", Code: `function f() { this.eval("var a = 0"); } f.bind(obj);`, Config: mod(nil)},

		// --- `this.eval` in a sloppy script: the default-`this` analysis ------
		// Every one of these reaches isDefaultThisBinding, and the oracle
		// reports exactly the frames whose `this` is the default binding.
		{ID: "script-this-top-level", Code: `this.eval("var a = 0");`, Config: script(nil)},
		{ID: "script-this-function", Code: `function foo() { this.eval("var a = 0"); }`, Config: script(nil)},
		{ID: "script-this-function-upper-var", Code: `var Foo = function () { this.eval("var a = 0"); };`, Config: script(nil)},
		{ID: "script-this-function-lower-var", Code: `var foo = function () { this.eval("var a = 0"); };`, Config: script(nil)},
		{ID: "script-this-named-function-expression", Code: `var foo = function bar() { this.eval("var a = 0"); };`, Config: script(nil)},
		{ID: "script-this-upper-case-declaration", Code: `function Foo() { this.eval("var a = 0"); }`, Config: script(nil)},
		{ID: "script-this-object-method", Code: `var obj = { foo() { this.eval("var a = 0"); } };`, Config: script(nil)},
		{ID: "script-this-object-function-prop", Code: `var obj = { foo: function () { this.eval("var a = 0"); } };`, Config: script(nil)},
		{ID: "script-this-class-method", Code: `class A { foo() { this.eval("var a = 0"); } }`, Config: script(nil)},
		{ID: "script-this-assigned-to-member", Code: `obj.foo = function () { this.eval("var a = 0"); };`, Config: script(nil)},
		{ID: "script-this-assigned-to-upper-var", Code: `Foo = function () { this.eval("var a = 0"); };`, Config: script(nil)},
		{ID: "script-this-assigned-to-lower-var", Code: `foo = function () { this.eval("var a = 0"); };`, Config: script(nil)},
		{ID: "script-this-arrow-top-level", Code: `var f = () => this.eval("var a = 0");`, Config: script(nil)},
		{ID: "script-this-arrow-in-function", Code: `function f() { var g = () => this.eval("var a = 0"); return g; }`, Config: script(nil)},
		{ID: "script-this-iife", Code: `(function () { this.eval("var a = 0"); })();`, Config: script(nil)},
		{ID: "script-this-bound-no-arg", Code: `var f = function () { this.eval("var a = 0"); }.bind();`, Config: script(nil)},
		{ID: "script-this-bound-with-arg", Code: `var f = function () { this.eval("var a = 0"); }.bind(obj);`, Config: script(nil)},
		{ID: "script-this-call-with-arg", Code: `var f = function () { this.eval("var a = 0"); }.call(obj);`, Config: script(nil)},
		{ID: "script-this-foreach-no-thisarg", Code: `[1].forEach(function () { this.eval("var a = 0"); });`, Config: script(nil)},
		{ID: "script-this-foreach-with-thisarg", Code: `[1].forEach(function () { this.eval("var a = 0"); }, obj);`, Config: script(nil)},
		{ID: "script-this-map-with-thisarg", Code: `[1].map(function () { this.eval("var a = 0"); }, obj);`, Config: script(nil)},
		{ID: "script-this-reflect-apply", Code: `Reflect.apply(function () { this.eval("var a = 0"); }, obj, []);`, Config: script(nil)},
		{ID: "script-this-array-from", Code: `Array.from([], function () { this.eval("var a = 0"); }, obj);`, Config: script(nil)},
		{ID: "script-this-jsdoc-this-tag", Code: "/**\n * @this {Object}\n */\nfunction f() { this.eval(\"var a = 0\"); }", Config: script(nil)},
		{ID: "script-this-jsdoc-this-inline", Code: `var f = /* @this obj */ function () { this.eval("var a = 0"); };`, Config: script(nil)},
		{ID: "script-this-returned-from-function", Code: `function outer() { return function () { this.eval("var a = 0"); }; }`, Config: script(nil)},
		{ID: "script-this-returned-from-iife", Code: `var obj = {}; obj.foo = (function () { return function () { this.eval("var a = 0"); }; })();`, Config: script(nil)},
		{ID: "script-this-conditional", Code: `var f = cond ? function () { this.eval("var a = 0"); } : null;`, Config: script(nil)},
		{ID: "script-this-logical", Code: `var f = a || function () { this.eval("var a = 0"); };`, Config: script(nil)},
		{ID: "script-this-other-property", Code: `function f() { this.other("var a = 0"); }`, Config: script(nil)},
		{ID: "script-this-nested-function", Code: `function f() { function g() { this.eval("var a = 0"); } return g; }`, Config: script(nil)},
		{ID: "script-this-computed-member", Code: `function f() { this["eval"]("var a = 0"); }`, Config: script(nil)},
		{ID: "script-this-class-constructor", Code: `class A { constructor() { this.eval("var a = 0"); } }`, Config: script(nil)},
		{ID: "script-this-class-static-method", Code: `class A { static foo() { this.eval("var a = 0"); } }`, Config: script(nil)},
		{ID: "script-this-class-field", Code: `class A { foo = this.eval("var a = 0"); }`, Config: script(nil)},
		{ID: "script-this-class-static-field", Code: `class A { static foo = this.eval("var a = 0"); }`, Config: script(nil)},
		{ID: "script-this-class-field-arrow", Code: `class A { foo = () => this.eval("var a = 0"); }`, Config: script(nil)},
		{ID: "script-this-class-field-function", Code: `class A { foo = function () { this.eval("var a = 0"); }; }`, Config: script(nil)},
		{ID: "script-this-class-field-computed-key", Code: `class A { [this.eval("var a = 0")] = 1; }`, Config: script(nil)},
		{ID: "script-this-class-field-nested-class", Code: `class A { foo = class B { bar = this.eval("var a = 0"); }; }`, Config: script(nil)},
		{ID: "script-this-class-field-outer-function", Code: `class A { foo = (function () { return function () { this.eval("var a = 0"); }; })(); }`, Config: script(nil)},
		{ID: "script-this-clean", Code: `var a = 1; function f() { return a; }`, Config: script(nil)},

		// --- clean files ----------------------------------------------------
		{ID: "clean-simple", Code: `var a = 1; function f() { return a; }`, Config: mod(nil)},
		{ID: "clean-member", Code: `var obj = { x: "foo" }, key = "x", value = obj[key];`, Config: mod(nil)},
		{ID: "clean-class", Code: `class A { foo() { return this.bar; } static baz() { return this.qux; } }`, Config: mod(nil)},
		{ID: "clean-class-field", Code: `class A { foo = 1; }`, Config: mod(nil)},
		{ID: "clean-evalish-names", Code: `var evaluate = 1; var evalish = 2; obj.evaluate();`, Config: mod(nil)},
		{ID: "clean-comment", Code: "// eval(\"var a = 0\");\nvar a = 1;", Config: mod(nil)},
	}

	cases = append(cases, bareEvalCases()...)
	ruletest.Compare(t, Rule, cases)
}

// Formerly blocked cases, now in the corpus: the bare `eval` identifier used to
// be rejected by the Go parser ("The keyword 'eval' is reserved") because
// acorn-go folded acorn's reservedWords (strict) and reservedWordsStrictBind
// sets together and applied them to references. acorn only forbids `eval` and
// `arguments` in *binding* positions. The static-block case was blocked by the
// scope analyser's `parent`-link recursion (see rules/novar, rules/noalert).
// Both are fixed, so these run and are compared against the oracle like any
// other case.
func bareEvalCases() []ruletest.Case {
	allowIndirect := []any{map[string]any{"allowIndirect": true}}
	return []ruletest.Case{
		{ID: "bare-direct-call", Code: `eval("var a = 0");`, Config: mod(nil)},
		{ID: "bare-doc-example", Code: "var obj = { x: \"foo\" }, key = \"x\", value = eval(\"obj.\" + key);", Config: mod(nil)},
		{ID: "bare-indirect-sequence", Code: `(0, eval)("var a = 0");`, Config: mod(nil)},
		{ID: "bare-indirect-alias", Code: `var foo = eval; foo("var a = 0");`, Config: mod(nil)},
		{ID: "bare-indirect-argument", Code: `function f(cb) { cb(); } f(eval);`, Config: mod(nil)},
		{ID: "bare-indirect-returned", Code: `function f() { return eval; }`, Config: mod(nil)},
		{ID: "bare-optional-call", Code: `eval?.("var a = 0");`, Config: mod(nil)},
		{ID: "bare-direct-allow-indirect", Code: `eval("var a = 0");`, Options: allowIndirect, Config: mod(nil)},
		{ID: "bare-indirect-allow-indirect", Code: `(0, eval)("var a = 0");`, Options: allowIndirect, Config: mod(nil)},
		{ID: "bare-optional-allow-indirect", Code: `eval?.("var a = 0");`, Options: allowIndirect, Config: mod(nil)},
		{ID: "static-block-this-eval", Code: `class A { static { this.eval("var a = 0"); } }`, Config: script(globals("window"))},
	}
}
