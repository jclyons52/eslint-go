package nonewobject

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// parserOptions mirrors what the oracle driver always runs with (ecmaVersion
// 2022, sourceType module). The Go linter would otherwise default to an ES5
// scope analysis for a config that has no parserOptions, while the AST is an
// ES2022 module — block scopes (and so `let`/`const` shadowing) would be
// invisible to the rule.
var parserOptions = map[string]any{"ecmaVersion": 2022, "sourceType": "module"}

func TestParity(t *testing.T) {
	ruletest.CompareWith(t, Rule, []ruletest.Case{
		// Doc examples.
		{ID: "doc-incorrect", Code: "var a = new Object();"},
		{ID: "doc-correct", Code: "var a = {};"},

		// Every shape of `new Object`.
		{ID: "statement", Code: "new Object();"},
		{ID: "no-args", Code: "new Object;"},
		{ID: "with-arg", Code: "new Object(null);"},
		{ID: "nested", Code: "var a = new Object(new Object());"},
		{ID: "parenthesized-callee", Code: "new (Object)();"},
		{ID: "arrow-body", Code: "const f = () => new Object();"},
		{ID: "class-method", Code: "class A { m() { return new Object(); } }"},

		// A shadowing declaration suppresses the report.
		{ID: "shadow-param", Code: "function f(Object) { return new Object(); }"},
		{ID: "shadow-var", Code: "var Object = 1; new Object();"},
		{ID: "shadow-function", Code: "function Object() {} new Object();"},
		{ID: "shadow-block", Code: "function f() { var Object = 1; return new Object(); }"},
		{ID: "shadow-let-in-block", Code: "new Object(); { let Object = 1; Object(); }"},
		{ID: "report-before-block-let", Code: "new Object(); { let Object = 1; }"},
		{ID: "report-after-block-let", Code: "{ let Object = 1; } new Object();"},
		{ID: "shadow-var-in-block", Code: "new Object(); { var Object = 1; }"},
		{ID: "shadow-let-in-function", Code: "new Object(); function f() { let Object = 1; }"},
		{ID: "shadow-module-let", Code: "let Object = 1; new Object();"},
		{ID: "unrelated-block-let", Code: "new Object(); { let Other = 1; }"},

		// Not `Object`.
		{ID: "other-constructor", Code: "new Array();"},
		{ID: "lowercase", Code: "new object();"},
		{ID: "member-callee", Code: "new foo.Object();"},
		{ID: "global-member-callee", Code: "new window.Object();"},
		{ID: "call-not-new", Code: "Object();"},
		{ID: "class-extends", Code: "class A extends Object {}"},
		{ID: "object-literal", Code: "var a = { b: 1 };"},

		// Configured globals: the built-in stays a global.
		{ID: "global-readonly", Code: "new Object();", Config: map[string]any{
			"rules":   map[string]any{Name: 2},
			"globals": map[string]any{"Object": "readonly"},
		}},
		{ID: "global-off", Code: "new Object();", Config: map[string]any{
			"rules":   map[string]any{Name: 2},
			"globals": map[string]any{"Object": "off"},
		}},

		// Severity.
		{ID: "warn", Code: "new Object();", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "new Object();", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Clean file and non-ASCII source.
		{ID: "clean", Code: "var a = {}; var b = new Array(); var c = new Foo();\n"},
		{ID: "non-ascii", Code: "var é = new Object();"},
	}, ruletest.Options{ExtraConfig: map[string]any{"parserOptions": parserOptions}})
}
