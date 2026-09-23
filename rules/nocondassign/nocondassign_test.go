package nocondassign

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// TestParity covers both option values: the default "except-parens" branch
// (statement handlers + the paren guards, with the ForStatement single-paren
// exception) and the "always" branch (AssignmentExpression handler walking up
// to the nearest conditional ancestor, stopping at function boundaries).
func TestParity(t *testing.T) {
	always := []any{"always"}
	exceptParens := []any{"except-parens"}

	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- except-parens (default): the `missing` report ----------------
		{ID: "if-assign", Code: "if (x = 0) {}"},
		{ID: "if-double-parens", Code: "if ((x = 0)) {}"},
		{ID: "while-assign", Code: "while (x = 0) {}"},
		{ID: "while-double-parens", Code: "while ((x = 0)) {}"},
		{ID: "do-while-assign", Code: "do {} while (x = 0);"},
		{ID: "do-while-double-parens", Code: "do {} while ((x = 0));"},
		{ID: "for-assign", Code: "for (; x = 0; ) {}"},
		{ID: "for-single-parens", Code: "for (; (x = 0); ) {}"},
		{ID: "for-double-parens", Code: "for (; ((x = 0)); ) {}"},
		{ID: "for-init-assign", Code: "for (x = 0; ; ) {}"},
		{ID: "conditional-assign", Code: "var y = (x = 0) ? 1 : 2;"},
		{ID: "conditional-double-parens", Code: "var y = ((x = 0)) ? 1 : 2;"},
		{ID: "conditional-plain-test", Code: "var y = a ? (x = 0) : 1;"},

		// ---- except-parens: tests that are not a bare assignment -----------
		{ID: "nested-logical", Code: "if (a && (b = c)) {}"},
		{ID: "call-in-test", Code: "if (foo(x = 1)) {}"},
		{ID: "nested-if", Code: "if (a) { if (b = c) {} }"},
		{ID: "comment-in-parens", Code: "if ((x = 0 /* c */)) {}"},
		{ID: "multiline", Code: "if (\n  x = 0\n) {}"},
		{ID: "clean", Code: "if (a === b) { while (c) {} }"},
		{ID: "non-ascii", Code: "if (caf\u00e9 = 1) {}"},
		{ID: "except-parens-explicit", Code: "if (x = 0) {}", Options: exceptParens},

		// ---- always: the `unexpected` report -------------------------------
		{ID: "always-if", Code: "if (x = 0) {}", Options: always},
		{ID: "always-if-parens", Code: "if ((x = 0)) {}", Options: always},
		{ID: "always-while", Code: "while (x = 0) {}", Options: always},
		{ID: "always-do-while", Code: "do {} while (x = 0);", Options: always},
		{ID: "always-for", Code: "for (; x = 0; ) {}", Options: always},
		{ID: "always-conditional", Code: "var y = (x = 0) ? 1 : 2;", Options: always},
		{ID: "always-nested-logical", Code: "if ((x = 0) && b) {}", Options: always},
		{ID: "always-nested-assign", Code: "if (x = (y = 1)) {}", Options: always},
		{ID: "always-nested-if", Code: "if (a) { if (b = c) {} }", Options: always},
		{ID: "always-in-function", Code: "function f() { if (x = 0) {} }", Options: always},

		// ---- always: no conditional ancestor -------------------------------
		{ID: "always-function-boundary", Code: "if ((function () { return (x = 0); })()) {}", Options: always},
		{ID: "always-arrow", Code: "const f = () => (x = 0);", Options: always},
		{ID: "always-for-init", Code: "for (x = 0; ; ) {}", Options: always},
		{ID: "always-conditional-consequent", Code: "var y = a ? (x = 0) : 1;", Options: always},
		{ID: "always-clean", Code: "if (a === b) {}", Options: always},

		// ---- severity -------------------------------------------------------
		{ID: "warn", Code: "if (x = 0) {}", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "always-warn", Code: "if (x = 0) {}",
			Config: map[string]any{"rules": map[string]any{Name: []any{"warn", "always"}}}},
	})
}
