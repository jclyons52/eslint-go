package nodupeargs

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Doc examples.
		{ID: "declaration", Code: "function foo(a, b) {}"},
		{ID: "declaration-single", Code: "function foo(a) {}"},
		{ID: "declaration-none", Code: "function foo() {}"},
		{ID: "expression", Code: "var f = function (a, b) {};"},
		{ID: "named-expression", Code: "var f = function g(a, b) {};"},

		// Distinct names in one list: never reported.
		{ID: "three-params", Code: "function foo(a, b, c) {}"},
		{ID: "defaults", Code: "function foo(a, b = 1) {}"},
		{ID: "rest", Code: "function foo(a, ...rest) {}"},
		{ID: "destructuring", Code: "function foo({ a, b }, [c]) {}"},

		// The same name in sibling scopes is not a duplicate.
		{ID: "nested-scopes", Code: "function outer(a) { function inner(a) {} return inner; }"},
		{ID: "block-scope", Code: "function outer(a) { { let b = a; return b; } }"},

		// Only FunctionDeclaration/FunctionExpression are visited (an arrow's
		// parameter list is validated by the parser).
		{ID: "arrow", Code: "var f = (a, b) => a + b;"},
		{ID: "arrow-single", Code: "var f = a => a;"},
		{ID: "method", Code: "var o = { m(a, b) {} };"},
		{ID: "class-method", Code: "class C { m(a, b) {} }"},
		{ID: "constructor", Code: "class C { constructor(a, b) {} }"},
		{ID: "getter-setter", Code: "var o = { get x() { return 1; }, set x(v) {} };"},

		// Clean file.
		{ID: "clean", Code: "function f(a, b) { return a + b; }\nfunction g(c) { return c; }\n"},

		// Non-ASCII parameter names.
		{ID: "non-ascii", Code: "function f(é, b) { return é + b; }"},

		// Severity.
		{ID: "warn", Code: "function foo(a, b) {}", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "function foo(a, b) {}", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Duplicate parameters. ESLint only accepts these in sloppy script
		// mode, so the cases that exercise the rule's report path are
		// sourceType "script" (the Go linter's own default — the harness does
		// not copy SourceType into the Go config, and espree-go always parses
		// as a module, so both sides see the same AST).
		{ID: "dupe-declaration", Code: "function foo(a, a) {}", SourceType: "script"},
		{ID: "dupe-expression", Code: "var f = function (a, a) {};", SourceType: "script"},
		{ID: "dupe-second", Code: "function foo(a, b, b) {}", SourceType: "script"},
		{ID: "dupe-three", Code: "function foo(a, b, a) {}", SourceType: "script"},
		{ID: "dupe-triple", Code: "function foo(a, a, a) {}", SourceType: "script"},
		{ID: "dupe-nested", Code: "function outer(a, a) { function inner(b, b) {} return inner; }", SourceType: "script"},
		{ID: "dupe-non-ascii", Code: "function foo(é, é) {}", SourceType: "script"},
		{ID: "dupe-warn", Code: "function foo(a, a) {}", SourceType: "script", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "dupe-separate-functions", Code: "function f(a, a) {}\nfunction g(b, b) {}\n", SourceType: "script"},

		// Duplicate parameters in a non-simple list (or in module code) are an
		// acorn early error, not a rule report: real espree rejects them with
		// "Argument name clash" and so does acorn-go (checkParams → checkClashes
		// in checkLValSimple), so these three are compared as parse errors.
		{ID: "dupe-nonsimple-default", Code: "function foo(a, a = 1) {}"},
		{ID: "dupe-nonsimple-pattern", Code: "function foo({ a }, { a }) {}"},
		{ID: "dupe-arrow", Code: "var f = (a, a) => a;"},
	})
}
