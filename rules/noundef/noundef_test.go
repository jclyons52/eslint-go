package noundef

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	globals := func(names ...string) map[string]any {
		g := map[string]any{}
		for _, n := range names {
			g[n] = "readonly"
		}
		return map[string]any{"globals": g}
	}
	ruletest.Compare(t, Rule, []ruletest.Case{
		{ID: "undeclared-read", Code: "foo();"},
		{ID: "declared", Code: "var foo = 1; foo();"},
		{ID: "let-in-block", Code: "{ let foo = 1; foo(); }"},
		{ID: "function-scope", Code: "function f() { return bar; }"},
		{ID: "param", Code: "function f(a) { return a; }"},
		{ID: "typeof-default", Code: "typeof foo;"},
		{ID: "typeof-option-on", Code: "typeof foo;", Options: []any{map[string]any{"typeof": true}}},
		{ID: "typeof-declared", Code: "var foo = 1; typeof foo;"},
		{ID: "configured-global", Code: "foo();", Config: globals("foo")},
		{ID: "configured-global-unused-other", Code: "bar();", Config: globals("foo")},
		{ID: "import-binding", Code: "import { a } from 'm'; a();"},
		{ID: "import-missing-use", Code: "import { a } from 'm'; b();"},
		{ID: "catch-param", Code: "try {} catch (e) { e(); }"},
		{ID: "class-decl", Code: "class A {} new A();"},
		{ID: "class-undeclared-extends", Code: "class A extends B {}"},
		{ID: "arrow-params", Code: "const f = (x, y) => x + y + z;"},
		{ID: "for-of-destructure", Code: "for (const { a, b } of xs) { a(); }"},
		{ID: "nested-shadow", Code: "var a = 1; function f() { var a = 2; return a; }"},
		{ID: "assignment-undeclared", Code: "function f() { undeclared = 1; }"},
		{ID: "template-literal", Code: "const s = `${thing}`;"},
		{ID: "member-expression", Code: "obj.prop;"},
		{ID: "member-on-declared", Code: "const obj = {}; obj.prop;"},
		{ID: "shorthand-property", Code: "const a = 1; const o = { a };"},
		{ID: "exported-fn-used", Code: "export function f() {} f();"},
	})
}
