package nonewwrappers

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// parserOptions mirrors what the oracle driver always runs with (ecmaVersion
// 2022, sourceType module). Without them the Go linter would analyze the file
// with ES5 scope semantics while the AST is an ES2022 module, so block scopes
// (`let`/`const` shadowing) would be invisible to the rule.
var parserOptions = map[string]any{"ecmaVersion": 2022, "sourceType": "module"}

func TestParity(t *testing.T) {
	ruletest.CompareWith(t, Rule, []ruletest.Case{
		// Doc examples.
		{ID: "doc-string", Code: "var a = new String('hello');"},
		{ID: "doc-correct", Code: "var a = String('hello');"},

		// Every wrapper object, with and without arguments.
		{ID: "number", Code: "var a = new Number(1);"},
		{ID: "boolean", Code: "var a = new Boolean(true);"},
		{ID: "no-args", Code: "new String();"},
		{ID: "nested", Code: "new String(new Number(1));"},
		{ID: "parenthesized-callee", Code: "new (String)();"},
		{ID: "statement", Code: "new Boolean(false);"},
		{ID: "if-test", Code: "if (new Boolean(false)) {}"},
		{ID: "arrow-body", Code: "const f = () => new Number(1);"},
		{ID: "class-method", Code: "class A { m() { return new Boolean(true); } }"},
		{ID: "multiple", Code: "new String('a');\nnew Number(1);\nnew Boolean(false);\n"},

		// Not wrapper objects.
		{ID: "object", Code: "var a = new Object();"},
		{ID: "array", Code: "var a = new Array();"},
		{ID: "lowercase", Code: "new string();"},
		{ID: "member-callee", Code: "new foo.String();"},
		{ID: "global-member-callee", Code: "new window.Number();"},
		{ID: "call-not-new", Code: "Number('1');"},

		// A shadowing declaration suppresses the report.
		{ID: "shadow-param", Code: "function f(String) { return new String('x'); }"},
		{ID: "shadow-var", Code: "var Number = 1; new Number();"},
		{ID: "shadow-var-hoisted", Code: "new String(); var String = 1;"},
		{ID: "shadow-function", Code: "function Boolean() {} new Boolean(true);"},
		{ID: "shadow-block", Code: "function f() { var String = 1; return new String(); }"},
		{ID: "shadow-let-in-block", Code: "new String(); { let String = 1; String(); }"},
		{ID: "shadow-module-let", Code: "let String = 1; new String();"},
		{ID: "shadow-in-class", Code: "class A { m(String) { return new String('x'); } }"},

		// Configured globals: the built-in stays a global.
		{ID: "global-readonly", Code: "new String('x');", Config: map[string]any{
			"rules":   map[string]any{Name: 2},
			"globals": map[string]any{"String": "readonly"},
		}},
		{ID: "global-writable", Code: "new String('x');", Config: map[string]any{
			"rules":   map[string]any{Name: 2},
			"globals": map[string]any{"String": "writable"},
		}},
		// `"off"` deletes the global, so the name is undeclared and the rule
		// reports nothing — the core removes it from the global scope.
		{ID: "global-off", Code: "new String('x');", Config: map[string]any{
			"rules":   map[string]any{Name: 2},
			"globals": map[string]any{"String": "off"},
		}},

		// Severity.
		{ID: "warn", Code: "new String('x');", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "new String('x');", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Clean file and non-ASCII source.
		{ID: "clean", Code: "var a = new Object(); var b = new Array(); var c = String(1);\n"},
		{ID: "non-ascii", Code: "var a = new String('é');"},
	}, ruletest.Options{ExtraConfig: map[string]any{"parserOptions": parserOptions}})
}
