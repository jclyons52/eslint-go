package eqeqeq

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- doc examples: default ("always") --------------------------------
		{ID: "doc-default-incorrect-1", Code: "if (x == 42) { }"},
		{ID: "doc-default-incorrect-2", Code: "if (\"\" == text) { }"},
		{ID: "doc-default-incorrect-3", Code: "if (obj.getStuff() != undefined) { }"},
		{ID: "doc-default-correct-1", Code: "if (x === 42) { }"},
		{ID: "doc-default-correct-2", Code: "if (\"\" === text) { }"},
		{ID: "doc-default-correct-3", Code: "if (obj.getStuff() !== undefined) { }"},

		// ---- doc examples: {"null": "always"} --------------------------------
		{ID: "doc-null-always-incorrect", Code: "if (x === null) { }", Options: []any{"always", map[string]any{"null": "always"}}},
		{ID: "doc-null-always-correct", Code: "if (x == null) { }", Options: []any{"always", map[string]any{"null": "always"}}},

		// ---- doc examples: {"null": "never"} ---------------------------------
		{ID: "doc-null-never-incorrect", Code: "if (x == null) { }", Options: []any{"always", map[string]any{"null": "never"}}},
		{ID: "doc-null-never-correct", Code: "if (x === null) { }", Options: []any{"always", map[string]any{"null": "never"}}},

		// ---- doc examples: "smart" -------------------------------------------
		{ID: "doc-smart-incorrect", Code: "if (x == 42) { }", Options: []any{"smart"}},
		{ID: "doc-smart-correct-1", Code: "if (x === 42) { }", Options: []any{"smart"}},
		{ID: "doc-smart-correct-2", Code: "if (x == null) { }", Options: []any{"smart"}},
		{ID: "doc-smart-correct-3", Code: "if (typeof x == \"number\") { }", Options: []any{"smart"}},
		{ID: "doc-smart-correct-4", Code: "if (x == 1) { }", Options: []any{"smart", map[string]any{"null": "never"}}},

		// ---- doc examples: "allow-null" (deprecated) -------------------------
		{ID: "doc-allow-null-correct", Code: "if (x == null) { }", Options: []any{"allow-null"}},
		{ID: "doc-allow-null-incorrect", Code: "if (x == 1) { }", Options: []any{"allow-null"}},

		// ---- the operator matrix, unfixable vs fixable ------------------------
		{ID: "loose-eq-unfixable", Code: "a == b;", Fix: true},
		{ID: "loose-neq-unfixable", Code: "a != b;", Fix: true},
		{ID: "strict-clean", Code: "a === b; a !== b;"},
		{ID: "relational-ignored", Code: "a < b; a > b; a <= b; a >= b; a + b; a - b;"},
		{ID: "multiple", Code: "a == b; c != d; e === f;", Fix: true},

		// ---- null handling ---------------------------------------------------
		{ID: "null-default", Code: "a == null; a != null; a === null; a !== null;", Fix: true},
		{ID: "null-always", Code: "a == null; a != null; a === null; a !== null;",
			Options: []any{"always", map[string]any{"null": "always"}}, Fix: true},
		{ID: "null-never", Code: "a == null; a != null; a === null; a !== null;",
			Options: []any{"always", map[string]any{"null": "never"}}, Fix: true},
		{ID: "null-ignore", Code: "a == null; a != null; a === null; a !== null;",
			Options: []any{"always", map[string]any{"null": "ignore"}}, Fix: true},
		{ID: "null-literal-left", Code: "null == a; null !== a;",
			Options: []any{"always", map[string]any{"null": "never"}}, Fix: true},
		{ID: "null-vs-null", Code: "null == null;", Fix: true},
		{ID: "null-vs-null-never", Code: "null === null;",
			Options: []any{"always", map[string]any{"null": "never"}}, Fix: true},
		{ID: "undefined-not-null", Code: "a == undefined; a != undefined;", Fix: true},
		{ID: "void-not-null", Code: "a == void 0;", Fix: true},
		{ID: "smart-null-never", Code: "a == null; a === null;",
			Options: []any{"smart", map[string]any{"null": "never"}}, Fix: true},

		// ---- typeof comparisons (always fixable) ------------------------------
		{ID: "typeof-eq", Code: "typeof a == \"number\";", Fix: true},
		{ID: "typeof-neq", Code: "typeof a != \"number\";", Fix: true},
		{ID: "typeof-both-sides", Code: "typeof a == typeof b;", Fix: true},
		{ID: "typeof-right", Code: "\"number\" == typeof a;", Fix: true},
		{ID: "typeof-nested", Code: "typeof (a.b) == \"object\";", Fix: true},
		{ID: "smart-typeof", Code: "typeof a == \"number\";", Options: []any{"smart"}, Fix: true},

		// ---- literals of the same/different JS type ---------------------------
		{ID: "number-same", Code: "1 == 1;", Fix: true},
		{ID: "number-different", Code: "1 == 2;", Fix: true},
		{ID: "string-same", Code: "'a' == 'a';", Fix: true},
		{ID: "string-different", Code: "'a' == 'b';", Fix: true},
		{ID: "boolean-same", Code: "true == false;", Fix: true},
		{ID: "number-vs-string", Code: "1 == '1';", Fix: true},
		{ID: "string-vs-number", Code: "'1' == 1;", Fix: true},
		{ID: "null-vs-number", Code: "null == 1;", Fix: true},
		{ID: "regex-same", Code: "/a/ == /a/;", Fix: true},
		{ID: "regex-different", Code: "/a/ == /b/;", Fix: true},
		{ID: "bigint-same", Code: "1n == 1n;", Fix: true},
		{ID: "bigint-different", Code: "1n == 2n;", Fix: true},
		{ID: "bigint-vs-string", Code: "1n == '1';", Fix: true},
		{ID: "bigint-vs-number", Code: "1n == 1;", Fix: true},
		{ID: "template-not-literal", Code: "`a` == `a`;", Fix: true},
		{ID: "template-vs-string", Code: "`a` == 'a';", Fix: true},
		{ID: "array-not-literal", Code: "[] == [];", Fix: true},
		{ID: "object-not-literal", Code: "({}) == ({});", Fix: true},

		// ---- parentheses and comments around the operator ----------------------
		{ID: "parens-left", Code: "(a) == b;", Fix: true},
		{ID: "parens-literals", Code: "(1) == (1);", Fix: true},
		{ID: "parens-whole", Code: "(a == b);", Fix: true},
		{ID: "comment-before-op", Code: "a /* c */ == b;", Fix: true},
		{ID: "comment-after-op", Code: "a == /* c */ b;", Fix: true},
		{ID: "comment-line", Code: "a == // c\n b;", Fix: true},
		{ID: "multiline", Code: "const r = a\n    ==\n    b;", Fix: true},
		{ID: "multiline-fixable", Code: "const r = 1\n    ==\n    2;", Fix: true},

		// ---- nesting / control flow -------------------------------------------
		{ID: "in-if", Code: "if (a == b) { } else if (c != d) { }", Fix: true},
		{ID: "in-while", Code: "while (a == b) { }", Fix: true},
		{ID: "in-function", Code: "function f(a, b) { return a == b; }", Fix: true},
		{ID: "in-arrow", Code: "const f = (a) => a == 1;", Fix: true},
		{ID: "nested", Code: "if ((a == b) == (c == d)) { }", Fix: true},
		{ID: "chained", Code: "a == b == c;", Fix: true},
		{ID: "ternary", Code: "const r = a == b ? 1 : 2;", Fix: true},
		{ID: "in-switch", Code: "switch (a) { case 1: if (b == c) { } }", Fix: true},
		{ID: "logical", Code: "a && b == c || d != e;", Fix: true},

		// ---- non-ASCII sources (fix ranges are emitted in UTF-16 units) --------
		{ID: "non-ascii-unfixable", Code: "const é = 1; const b = 2; é == b;", Fix: true},
		{ID: "non-ascii-fixable", Code: "const é = 1; const b = 2; é == é;", Fix: true},
		{ID: "non-ascii-string-literal", Code: "const é = 'a'; é == 'a';", Fix: true},
		{ID: "emoji", Code: "const s = '😀'; s == '😀';", Fix: true},

		// ---- clean files -------------------------------------------------------
		{ID: "clean", Code: "const a = 1, b = 2;\nif (a === b) { }\nif (a !== b) { }\n"},
		{ID: "clean-empty", Code: "\n"},

		// ---- severity ----------------------------------------------------------
		{ID: "warn", Code: "a == b;", Config: map[string]any{"rules": map[string]any{Name: "warn"}}, Fix: true},
		{ID: "off", Code: "a == b;", Config: map[string]any{"rules": map[string]any{Name: "off"}}},
		{ID: "warn-fixable", Code: "typeof a == 'number';",
			Config: map[string]any{"rules": map[string]any{Name: "warn"}}, Fix: true},

		// ---- option shapes that are not the documented ones ---------------------
		{ID: "empty-options", Code: "a == b; a == null;", Options: []any{}, Fix: true},
		{ID: "options-null-empty-object", Code: "a == null; a === null;",
			Options: []any{"always", map[string]any{}}, Fix: true},
		{ID: "smart-with-null-always", Code: "a == null; a === null;",
			Options: []any{"smart", map[string]any{"null": "always"}}, Fix: true},
		{ID: "smart-with-null-ignore", Code: "a == null; a == b;",
			Options: []any{"smart", map[string]any{"null": "ignore"}}, Fix: true},
		{ID: "allow-null-with-object", Code: "a == null; a === null;",
			Options: []any{"allow-null", map[string]any{"null": "never"}}, Fix: true},
	})
}
