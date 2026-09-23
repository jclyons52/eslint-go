package noalert

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// moduleParserOptions aligns the Go config with the parserOptions the oracle
// driver applies by default ({ecmaVersion: 2022, sourceType: "module"}).
// ruletest only forwards a case's SourceType/ECMAVersion to the oracle, so the
// Go side needs them spelled out in the config or it would analyse every file
// as an ES5 script — which changes what scope `this` at the top level sees.
func moduleParserOptions() ruletest.Options {
	return ruletest.Options{ExtraConfig: map[string]any{
		"parserOptions": map[string]any{"sourceType": "module", "ecmaVersion": 2022},
	}}
}

func TestParity(t *testing.T) {
	ruletest.CompareWith(t, Rule, []ruletest.Case{
		// The three prohibited identifiers, bare.
		{ID: "alert", Code: `alert("x");`},
		{ID: "confirm", Code: `confirm("x");`},
		{ID: "prompt", Code: `prompt("x");`},

		// Several calls in one file, including ones on later lines.
		{ID: "several", Code: "alert(1);\nconfirm(2);\nprompt(3);"},

		// Via the global object.
		{ID: "window-alert", Code: `window.alert("x");`},
		{ID: "window-confirm", Code: `window.confirm("x");`},
		{ID: "window-prompt", Code: `window.prompt("x");`},
		{ID: "window-computed", Code: `window["alert"]("x");`},
		{ID: "window-template-key", Code: "window[`alert`](\"x\");"},

		// `globalThis` is only the global object when the name is defined.
		{ID: "globalthis-undefined", Code: `globalThis.prompt("x");`},
		{
			ID:     "globalthis-defined",
			Code:   `globalThis.prompt("x");`,
			Config: map[string]any{"globals": map[string]any{"globalThis": "readonly"}},
		},
		{
			ID:     "globalthis-shadowed",
			Code:   `function f(globalThis) { globalThis.alert("x"); }`,
			Config: map[string]any{"globals": map[string]any{"globalThis": "readonly"}},
		},

		// Not the global object: no report.
		{ID: "window-window", Code: `window.window.alert("x");`},
		{ID: "other-object", Code: `foo.alert("x");`},
		{ID: "member-of-call", Code: `getWindow().alert("x");`},
		{ID: "call-of-member", Code: `alert.call(null, "x");`},
		{ID: "this-module-scope", Code: `this.alert("x");`},
		{ID: "this-in-function", Code: `function f() { this.alert("x"); }`},
		{ID: "this-in-arrow", Code: `const f = () => this.alert("x");`},
		{ID: "not-a-call", Code: `alert;`},
		{ID: "new-alert", Code: `new alert();`},
		{ID: "tagged-template", Code: "alert`x`;"},

		// Shadowing suppresses the bare-identifier form.
		{ID: "shadowed-param", Code: `function f(alert) { alert("x"); }`},
		{ID: "shadowed-var", Code: `var alert = function () {}; alert("x");`},
		{ID: "shadowed-const-block", Code: `{ const alert = () => {}; alert("x"); }`},
		{ID: "shadowed-window", Code: `function f(window) { window.alert("x"); }`},
		{ID: "shadowed-window-var", Code: `var window = {}; window.alert("x");`},
		{ID: "shadowed-in-inner-scope", Code: `function f() { const prompt = () => {}; prompt("x"); }`},
		{ID: "outer-scope-unshadowed", Code: `const prompt = () => {}; function f() { alert("x"); }`},

		// Nested scopes and class bodies.
		{ID: "nested-function", Code: `function f() { alert("x"); }`},
		{ID: "arrow-body", Code: `const f = () => { confirm("x"); };`},
		{ID: "class-method", Code: `class C { m() { prompt("x"); } }`},
		{ID: "class-static-method", Code: `class C { static m() { alert("x"); } }`},
		// NOTE (core gap, not a port gap): a class static block
		// (`class C { static { alert("x"); } }`) cannot be exercised. Any source
		// containing `static { … }` makes the core's scope analysis recurse until
		// the stack overflows — eslint-go's VisitorKeys omits "StaticBlock"
		// (eslint-visitor-keys has StaticBlock: ["body"]) and eslint-scope-go's
		// "iteration" fallback (visitor.go iterateKeys) returns every key,
		// including the `parent` link the traverser attached, so the walk cycles
		// between a node and its parent. It crashes before create() runs, for
		// every rule — the same gap rules/semi, rules/preferconst and
		// rules/noextrasemi document.
		{ID: "iife", Code: `(function () { alert("x"); })();`},
		{ID: "deeply-nested", Code: `function f() { return function () { return function () { alert("x"); }; }; }`},

		// Optional chaining: the callee is a ChainExpression.
		{ID: "optional-window", Code: `window?.alert("x");`},
		{ID: "optional-bare", Code: `alert?.("x");`},
		{ID: "optional-window-computed", Code: `window?.["alert"]("x");`},

		// Severities.
		{ID: "warn", Code: `alert("x");`, Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: `alert("x");`, Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Clean files.
		{ID: "clean", Code: `foo();`},
		{ID: "clean-nested", Code: `function f() { return 1; }`},
		{ID: "clean-member", Code: `obj.alert;`},

		// Non-ASCII: emitted columns are UTF-16 code units.
		{ID: "non-ascii-string", Code: `alert("héllo → 世界 🎉");`},
		{ID: "non-ascii-window", Code: `window.prompt("naïve café 日本語");`},
		{ID: "non-ascii-after", Code: "const 日本 = 1;\nalert(\"x\");"},
		{ID: "non-ascii-comment", Code: "// 世界\nalert(\"x\");"},
	}, moduleParserOptions())
}
