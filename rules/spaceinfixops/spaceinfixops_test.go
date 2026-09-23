package spaceinfixops

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Doc examples: assignment, binary, logical, conditional, declarator.
		{ID: "assign-unspaced", Code: "a=b;\n", Fix: true},
		{ID: "assign-spaced", Code: "a = b;\n"},
		{ID: "binary-unspaced", Code: "a+b;\n", Fix: true},
		{ID: "binary-spaced", Code: "a + b;\n"},
		{ID: "binary-left-only", Code: "a+ b;\n", Fix: true},
		{ID: "binary-right-only", Code: "a +b;\n", Fix: true},
		{ID: "logical-unspaced", Code: "a&&b;\n", Fix: true},
		{ID: "logical-nullish", Code: "a??b;\n", Fix: true},
		{ID: "relational-unspaced", Code: "a<=b;\n", Fix: true},
		{ID: "exponent-unspaced", Code: "a**b;\n", Fix: true},
		{ID: "in-operator", Code: "a in b;\n"},
		{ID: "instanceof-operator", Code: "a instanceof b;\n"},

		// Variable declarators.
		{ID: "declarator-unspaced", Code: "var a=1;\n", Fix: true},
		{ID: "declarator-spaced", Code: "var a = 1;\n"},
		{ID: "let-declarator", Code: "let a=1, b=2;\n", Fix: true},
		{ID: "const-declarator", Code: "const a=1;\n", Fix: true},

		// Conditional expressions.
		{ID: "conditional-unspaced", Code: "a?b:c;\n", Fix: true},
		{ID: "conditional-question-only", Code: "a? b : c;\n", Fix: true},
		{ID: "conditional-colon-only", Code: "a ? b:c;\n", Fix: true},
		{ID: "conditional-spaced", Code: "a ? b : c;\n"},
		{ID: "conditional-nested", Code: "a?b?c:d:e;\n", Fix: true},

		// Assignment patterns (destructuring defaults, params, arrow params).
		{ID: "object-pattern-default", Code: "const {a=1} = b;\n", Fix: true},
		{ID: "object-pattern-default-spaced", Code: "const {a = 1} = b;\n"},
		{ID: "array-pattern-default", Code: "const [a=1] = b;\n", Fix: true},
		{ID: "function-param-default", Code: "function f(a=1) {}\n", Fix: true},
		{ID: "arrow-param-default", Code: "const f = (a=1) => a;\n", Fix: true},
		{ID: "arrow-body-unspaced", Code: "const f = a=>a;\n"},
		{ID: "nested-pattern-default", Code: "const {a: {b=1}} = c;\n", Fix: true},

		// Assignment expressions in expressions.
		{ID: "assign-in-call", Code: "f(a=b);\n", Fix: true},
		{ID: "assign-in-condition", Code: "if ((a=b)) {}\n", Fix: true},
		{ID: "chained-assign", Code: "a=b=c;\n", Fix: true},
		{ID: "compound-assign", Code: "a+=b;\n", Fix: true},
		{ID: "logical-assign", Code: "a||=b;\n", Fix: true},

		// Class fields (PropertyDefinition).
		{ID: "class-field-unspaced", Code: "class A { p=1; }\n", Fix: true},
		{ID: "class-field-spaced", Code: "class A { p = 1; }\n"},
		{ID: "class-field-static", Code: "class A { static p=1; }\n", Fix: true},
		{ID: "class-field-computed", Code: "class A { [a]=1; }\n", Fix: true},

		// int32Hint.
		{ID: "int32-hint-on", Code: "var x = a|0;\n", Options: []any{map[string]any{"int32Hint": true}}},
		{ID: "int32-hint-off", Code: "var x = a|0;\n", Fix: true},
		{ID: "int32-hint-on-other", Code: "var x = a|b;\n",
			Options: []any{map[string]any{"int32Hint": true}}, Fix: true},
		{ID: "int32-hint-spaced", Code: "var x = a | 0;\n", Options: []any{map[string]any{"int32Hint": true}}},
		{ID: "int32-hint-explicit-false", Code: "var x = a|0;\n",
			Options: []any{map[string]any{"int32Hint": false}}, Fix: true},

		// Spread and other non-infix shapes must be left alone.
		{ID: "spread", Code: "f(...args);\n"},
		{ID: "spread-in-array", Code: "var a = [...b, 1];\n"},
		{ID: "unary", Code: "var a = -1;\n"},
		{ID: "unary-not", Code: "var a = !b;\n"},
		{ID: "update", Code: "a++; --b;\n"},
		{ID: "member", Code: "a.b.c;\n"},
		{ID: "clean", Code: "var a = 1;\nvar b = a + 1;\nif (a && b) { c = a ? b : 1; }\n"},

		// Severity.
		{ID: "warn", Code: "a=b;\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "a=b;\n", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Non-ASCII: the reported column and fix range are UTF-16 code units.
		{ID: "nonascii-operand", Code: "var caf\u00e9=1;\n", Fix: true},
		{ID: "nonascii-string", Code: "var s=\"\u4f60\u597d\";\n", Fix: true},
		{ID: "nonascii-emoji", Code: "var s=\"\U0001f600\"+\"\u4f60\u597d\";\n", Fix: true},
		{ID: "nonascii-clean", Code: "var caf\u00e9 = 1;\n"},
	})
}
