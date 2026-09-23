package nofuncassign

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// TestParity covers the rule's two handlers (FunctionDeclaration /
// FunctionExpression), the FunctionName defs[0] gate, and every shape of
// modifying reference astUtils.getModifyingReferences accepts: plain assign,
// compound assign, logical assign, update expressions, destructuring writes,
// for-in/of writes, and the initializer exception.
func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- modifying references to a function declaration name ----------
		{ID: "self-assign", Code: "function foo() { foo = 1; }"},
		{ID: "outer-assign", Code: "function foo() {} foo = 1;"},
		{ID: "two-writes", Code: "function foo() {} foo = 1; foo = 2;"},
		{ID: "compound", Code: "function foo() {} foo += 1;"},
		{ID: "postfix", Code: "function foo() {} foo++;"},
		{ID: "prefix", Code: "function foo() {} ++foo;"},
		{ID: "logical-assign", Code: "function foo() {} foo ??= 1;"},
		{ID: "array-destructuring", Code: "function foo() {} [foo] = [1];"},
		{ID: "object-destructuring", Code: "function foo() {} ({ a: foo } = { a: 1 });"},
		{ID: "from-other-function", Code: "function foo() {} function bar() { foo = 1; }"},
		{ID: "inside-iife", Code: "function foo() {} (function () { foo = 1; })();"},
		{ID: "param-plus-assign", Code: "function foo(a) { foo = a; }"},
		{ID: "in-block", Code: "function foo() {} if (true) { foo = 1; }"},
		{ID: "block-scoped-decl", Code: "{ function foo() {} foo = 1; }"},
		{ID: "for-of", Code: "function foo() {} for (foo of bar) {}"},
		{ID: "for-in", Code: "function foo() {} for (foo in bar) {}"},
		{ID: "exported", Code: "export function foo() {} foo = 1;"},

		// NOTE: `function foo() {} function foo() {} foo = 1;` is deliberately
		// absent — real ESLint in module (strict) mode rejects it at parse time
		// ("Identifier 'foo' has already been declared") while the Go parser
		// accepts it, so the case tests the parser, not the rule. See the
		// package report.

		// ---- named function expression: its own name is a FunctionName -----
		{ID: "named-func-expression", Code: "var a = function foo() { foo = 123; };"},
		{ID: "named-func-expression-outer", Code: "var a = function foo() {}; a = 1;"},

		// ---- no report -----------------------------------------------------
		{ID: "clean-two-decls", Code: "function foo() {} var bar = function () {};"},
		{ID: "clean-local-var", Code: "function foo() { var a = 1; a = 2; }"},
		{ID: "member-write", Code: "function foo() {} foo.bar = 1;"},
		{ID: "var-not-function", Code: "var foo = 1; foo = 2;"},
		{ID: "class-not-function", Code: "class foo {} foo = 1;"},
		{ID: "arrow-not-function", Code: "const foo = () => {}; foo = 1;"},
		{ID: "anonymous-expression", Code: "var a = function () { a = 1; };"},
		{ID: "default-param", Code: "function foo(a = 1) { a = 2; }"},
		{ID: "destructured-param", Code: "function foo({ a }) { a = 1; }"},
		{ID: "read-only", Code: "function foo() {} typeof foo;"},
		{ID: "empty", Code: ""},

		// ---- non-ASCII ------------------------------------------------------
		{ID: "non-ascii", Code: "function caf\u00e9() { caf\u00e9 = 1; }"},

		// ---- severity -------------------------------------------------------
		{ID: "warn", Code: "function foo() {} foo = 1;",
			Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "function foo() {} foo = 1;",
			Config: map[string]any{"rules": map[string]any{Name: "off"}}},
	})
}
