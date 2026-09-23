package nothrowliteral

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func severity(level any) map[string]any {
	return map[string]any{"rules": map[string]any{Name: level}}
}

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Literals: not error objects.
		{ID: "string", Code: "throw \"foo\";"},
		{ID: "number", Code: "throw 0;"},
		{ID: "boolean", Code: "throw true;"},
		{ID: "null", Code: "throw null;"},
		{ID: "object-literal", Code: "throw {};"},
		{ID: "array-literal", Code: "throw [];"},
		{ID: "template", Code: "throw `err`;"},
		{ID: "regex", Code: "throw /x/;"},
		{ID: "bigint", Code: "throw 1n;"},
		{ID: "this", Code: "throw this;"},
		{ID: "void-zero", Code: "throw void 0;"},
		{ID: "function-expression", Code: "throw function () {};"},
		{ID: "arrow-function", Code: "throw () => {};"},
		{ID: "class-expression", Code: "throw class {};"},

		// `undefined` gets its own message.
		{ID: "undefined", Code: "throw undefined;"},
		{ID: "undefined-paren", Code: "throw (undefined);"},

		// Possibly-error expressions.
		{ID: "identifier", Code: "throw err;"},
		{ID: "new-expression", Code: "throw new Error(\"x\");"},
		{ID: "call-expression", Code: "throw foo();"},
		{ID: "member-expression", Code: "throw obj.err;"},
		{ID: "tagged-template", Code: "throw tag`err`;"},
		{ID: "chain", Code: "throw a?.b;"},
		{ID: "await", Code: "throw await x;"},
		{ID: "yield", Code: "function* g() { throw yield x; }"},

		// AssignmentExpression: `=`/`&&=` follow the right side, `||=`/`??=`
		// follow either side, every other operator is arithmetic.
		{ID: "assign-eq-literal", Code: "throw a = 1;"},
		{ID: "assign-eq-identifier", Code: "throw a = b;"},
		{ID: "assign-and-eq-literal", Code: "throw a &&= 1;"},
		{ID: "assign-or-eq-identifier", Code: "throw a ||= 1;"},
		{ID: "assign-nullish-eq-literal", Code: "throw a ??= 1;"},
		{ID: "assign-plus-eq", Code: "throw a += 1;"},
		{ID: "assign-bitand-eq", Code: "throw a &= 1;"},

		// SequenceExpression: only the last expression matters.
		{ID: "sequence-last-literal", Code: "throw (foo(), \"bar\");"},
		{ID: "sequence-last-identifier", Code: "throw (foo(), bar);"},
		{ID: "sequence-empty-ish", Code: "throw (0, 1);"},

		// LogicalExpression: `&&` follows the right side, others either side.
		{ID: "logical-or-identifiers", Code: "throw a || b;"},
		{ID: "logical-or-literal", Code: "throw a || 1;"},
		{ID: "logical-and-literal", Code: "throw a && \"x\";"},
		{ID: "logical-and-identifiers", Code: "throw a && b;"},
		{ID: "logical-nullish-literal", Code: "throw a ?? 1;"},

		// ConditionalExpression: either branch.
		{ID: "conditional-identifiers", Code: "throw x ? y : z;"},
		{ID: "conditional-literals", Code: "throw x ? 1 : 2;"},
		{ID: "conditional-mixed", Code: "throw x ? y : 1;"},

		// Severities.
		{ID: "warn-severity", Code: "throw \"foo\";", Config: severity("warn")},
		{ID: "off-severity", Code: "throw \"foo\";", Config: severity("off")},
	})
}
