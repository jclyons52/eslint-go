package useisnan

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	switchCase := func(v bool) []any {
		return []any{map[string]any{"enforceForSwitchCase": v}}
	}
	indexOf := func(v bool) []any {
		return []any{map[string]any{"enforceForIndexOf": v}}
	}
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Doc examples.
		{ID: "doc-eq", Code: "if (foo == NaN) {\n    // ...\n}"},
		{ID: "doc-neq", Code: "if (foo != NaN) {\n    // ...\n}"},
		{ID: "doc-eq-number-nan", Code: "if (foo == Number.NaN) {\n    // ...\n}"},
		{ID: "doc-neq-number-nan", Code: "if (foo != Number.NaN) {\n    // ...\n}"},
		{ID: "doc-switch", Code: "switch (foo) {\n    case NaN:\n        bar();\n        break;\n    case 1:\n        baz();\n        break;\n    case 2:\n        qux();\n        break;\n}",
			Options: []any{map[string]any{"enforceForSwitchCase": true}}},
		{ID: "doc-indexof", Code: "var hasNaN = myArray.indexOf(NaN) >= 0;",
			Options: []any{map[string]any{"enforceForIndexOf": true}}},

		// Every operator the rule's regex accepts, both operand positions.
		{ID: "ops-nan-left", Code: "NaN < x; NaN > x; NaN <= x; NaN >= x; NaN == x; NaN != x; NaN === x; NaN !== x;"},
		{ID: "ops-nan-right", Code: "x < NaN; x > NaN; x <= NaN; x >= NaN; x == NaN; x != NaN; x === NaN; x !== NaN;"},
		{ID: "ops-not-matched", Code: "x + NaN; x - NaN; x * NaN; NaN && x; NaN || x; x ?? NaN; NaN ** x; x in NaN;"},
		{ID: "single-char-ops", Code: "NaN < x; NaN > x; x < NaN; x > NaN;"},
		{ID: "assignment", Code: "x = NaN;"},
		{ID: "unary", Code: "typeof NaN; -NaN; !NaN;"},

		// Number.NaN in all its static-name forms.
		{ID: "number-dot-nan", Code: "x === Number.NaN;"},
		{ID: "number-bracket-string", Code: "x === Number['NaN'];"},
		{ID: "number-computed-dynamic", Code: "x === Number[NaN];"},
		{ID: "number-other-prop", Code: "x === Number.MAX_VALUE;"},
		{ID: "number-nan-reversed", Code: "Number.NaN === x;"},
		{ID: "other-object", Code: "x === Math.NaN;"},
		{ID: "member-chain", Code: "x === a.b.NaN;"},
		{ID: "optional-member", Code: "x === Number?.NaN;"},

		// A shadowed NaN is still reported (the rule does not consult scopes).
		{ID: "shadowed-nan", Code: "function f(NaN) { return x === NaN; }"},
		{ID: "local-nan", Code: "const NaN = 1; if (x == NaN) { }"},

		// switch enforcement, default and explicit.
		{ID: "switch-default", Code: "switch (NaN) { case NaN: break; }"},
		{ID: "switch-number-nan", Code: "switch (Number.NaN) { case Number.NaN: break; }"},
		{ID: "switch-only-discriminant", Code: "switch (NaN) { case y: break; }"},
		{ID: "switch-only-case", Code: "switch (y) { case NaN: break; }"},
		{ID: "switch-middle-case", Code: "switch (y) { case 1: break; case NaN: break; case 2: break; }"},
		{ID: "switch-default-clause", Code: "switch (y) { default: case NaN: break; }"},
		{ID: "switch-no-cases", Code: "switch (NaN) { }"},
		{ID: "switch-empty-object-option", Code: "switch (NaN) { case NaN: break; }", Options: []any{map[string]any{}}},
		{ID: "switch-option-false", Code: "switch (NaN) { case NaN: break; }", Options: switchCase(false)},
		{ID: "switch-option-true", Code: "switch (NaN) { case NaN: break; }", Options: switchCase(true)},
		{ID: "switch-nested", Code: "function f() { switch (NaN) { case NaN: break; } }"},
		{ID: "switch-clean", Code: "switch (y) { case 1: break; default: break; }"},

		// indexOf / lastIndexOf enforcement.
		{ID: "indexof-off-by-default", Code: "a.indexOf(NaN);"},
		{ID: "indexof-empty-option", Code: "a.indexOf(NaN);", Options: []any{map[string]any{}}},
		{ID: "indexof-on", Code: "a.indexOf(NaN);", Options: indexOf(true)},
		{ID: "lastindexof-on", Code: "a.lastIndexOf(NaN);", Options: indexOf(true)},
		{ID: "indexof-mixed", Code: "a.indexOf(NaN); a.lastIndexOf(NaN); a.includes(NaN); a.indexOf(NaN, 1); a.indexOf(x);",
			Options: indexOf(true)},
		{ID: "indexof-no-args", Code: "a.indexOf();", Options: indexOf(true)},
		{ID: "indexof-two-args", Code: "a.indexOf(NaN, 0);", Options: indexOf(true)},
		{ID: "indexof-number-nan", Code: "a.indexOf(Number.NaN);", Options: indexOf(true)},
		{ID: "indexof-computed-name", Code: "a['indexOf'](NaN);", Options: indexOf(true)},
		{ID: "indexof-computed-dynamic", Code: "a[m](NaN);", Options: indexOf(true)},
		{ID: "indexof-chain", Code: "a?.indexOf(NaN);", Options: indexOf(true)},
		{ID: "indexof-chain-deep", Code: "a?.b?.indexOf(NaN);", Options: indexOf(true)},
		{ID: "indexof-non-member-callee", Code: "indexOf(NaN);", Options: indexOf(true)},
		{ID: "indexof-on-call", Code: "f().indexOf(NaN);", Options: indexOf(true)},
		{ID: "indexof-nested", Code: "if (a.indexOf(NaN) !== -1) { }", Options: indexOf(true)},
		{ID: "indexof-both-options", Code: "switch (NaN) { case NaN: break; }\na.indexOf(NaN);",
			Options: []any{map[string]any{"enforceForSwitchCase": true, "enforceForIndexOf": true}}},
		{ID: "indexof-both-options-off", Code: "switch (NaN) { case NaN: break; }\na.indexOf(NaN);",
			Options: []any{map[string]any{"enforceForSwitchCase": false, "enforceForIndexOf": false}}},

		// Clean files.
		{ID: "clean", Code: "if (Number.isNaN(x)) { }\nconst i = a.indexOf(1);\nswitch (x) { case 1: break; }\n"},
		{ID: "clean-isnan-call", Code: "isNaN(x); Number.isNaN(x);"},

		// Non-ASCII source around the reports.
		{ID: "non-ascii", Code: "const é = NaN;\nif (é === NaN) { }"},

		// Severity.
		{ID: "warn", Code: "x === NaN;", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "x === NaN;", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Multiple reports in one file, ordering by position.
		{ID: "multi", Code: "x === NaN;\nswitch (NaN) { case NaN: break; }\ny !== Number.NaN;",
			Options: []any{map[string]any{"enforceForSwitchCase": true}}},
	})
}
