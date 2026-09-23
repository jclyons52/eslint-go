package nounsafenegation

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// TestParity is expected to FAIL until the core can emit rule suggestions.
//
// The rule is ported in full (operators, ordering-relation option,
// parenthesised-operand check, loc = the left operand, messageId/data), and
// every message field it can produce matches the oracle byte for byte. What it
// cannot match is the oracle's `suggestions` array: no-unsafe-negation sets
// `meta.hasSuggestions: true` and passes a two-entry `suggest` list on *every*
// report, while eslint.Message has no `suggestions` field and eslint.Report has
// no `Suggest` field (docs/porting-rules.md pitfall 6 — "the message shape is
// not implemented"). So each of the 30 reporting cases below differs from the
// oracle by exactly one key, `suggestions`, and by nothing else (verified: 0
// non-suggestion differences).
//
// The corpus is deliberately complete — the suggestion-bearing cases are the
// rule's entire behaviour, so removing them to go green would leave the rule
// unverified. Making these cases pass needs a core change:
//
//	Message: add `Suggestions []Suggestion` (+ MarshalJSON emission, in
//	         ESLint's key order: after endLine/endColumn, before fix) and
//	Report:  add `Suggest []SuggestDescriptor`.
//
// Do not treat a green run of this file as parity until that lands.
func TestParity(t *testing.T) {
	ordering := []any{map[string]any{"enforceForOrderingRelations": true}}
	orderingOff := []any{map[string]any{"enforceForOrderingRelations": false}}

	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- doc examples ----------------------------------------------------
		{ID: "doc-incorrect-1", Code: "if (!key in object) {\n    // ...\n}"},
		{ID: "doc-incorrect-2", Code: "if (!obj instanceof Ctor) {\n    // ...\n}"},
		{ID: "doc-correct-1", Code: "if (!(key in object)) {\n    // ...\n}"},
		{ID: "doc-correct-2", Code: "if (!(obj instanceof Ctor)) {\n    // ...\n}"},

		// ---- the operators the rule always checks ----------------------------
		{ID: "in", Code: "!a in b;"},
		{ID: "instanceof", Code: "!a instanceof b;"},
		{ID: "in-parens", Code: "(!a) in b;"},
		{ID: "instanceof-parens", Code: "(!a) instanceof b;"},
		{ID: "in-double-parens", Code: "((!a)) in b;"},
		{ID: "in-arg-parens", Code: "!(a) in b;"},
		{ID: "no-negation", Code: "a in b; a instanceof b;"},
		{ID: "negation-on-right", Code: "a in !b; a instanceof !b;"},
		{ID: "negation-both", Code: "!a in !b;"},

		// ---- ordering relations: off by default ------------------------------
		{ID: "ordering-default", Code: "!a < b; !a > b; !a <= b; !a >= b;"},
		{ID: "ordering-on", Code: "!a < b; !a > b; !a <= b; !a >= b;", Options: ordering},
		{ID: "ordering-off-explicit", Code: "!a < b;", Options: orderingOff},
		{ID: "ordering-parens", Code: "(!a) < b; (!a) <= b;", Options: ordering},
		{ID: "ordering-arg-parens", Code: "!(a) < b;", Options: ordering},
		{ID: "ordering-no-negation", Code: "a < b; a >= b;", Options: ordering},
		{ID: "ordering-negation-on-right", Code: "a < !b;", Options: ordering},
		{ID: "ordering-on-other-ops", Code: "!a == b; !a === b; !a != b; !a + b; !a - b; !a && b;",
			Options: ordering},
		{ID: "ordering-on-in", Code: "!a in b; !a instanceof b;", Options: ordering},

		// ---- double negation and other unary shapes --------------------------
		{ID: "double-negation", Code: "!!a in b;"},
		{ID: "triple-negation", Code: "!!!a in b;"},
		{ID: "negated-literal", Code: "!\"a\" in b;"},
		{ID: "negated-number", Code: "!1 in b;"},
		{ID: "void-left", Code: "void a in b;"},
		{ID: "typeof-left", Code: "typeof a in b;"},
		{ID: "not-in-unary", Code: "!(a in b);"},
		{ID: "delete-left", Code: "delete a.b in c;"},

		// ---- comments between the operand and the operator --------------------
		{ID: "comment-before-operator", Code: "!a /* c */ in b;"},
		{ID: "comment-inside-parens", Code: "(!a /* c */) in b;"},
		{ID: "comment-line", Code: "!a // c\n in b;"},
		{ID: "multiline", Code: "const r = !a\n    in\n    b;"},
		{ID: "multiline-parens", Code: "const r = (!a)\n    in\n    b;"},

		// ---- nesting ----------------------------------------------------------
		{ID: "in-if", Code: "if (!a in b) { }"},
		{ID: "in-while", Code: "while (!a in b) { }"},
		{ID: "in-return", Code: "function f() { return !a in b; }"},
		{ID: "in-arrow", Code: "const f = (a) => !a in b;"},
		{ID: "in-ternary", Code: "const r = !a in b ? 1 : 2;"},
		{ID: "chained-in", Code: "!a in b in c;"},
		{ID: "chained-and", Code: "!a in b && !c instanceof d;"},
		{ID: "nested-call", Code: "f(!a in b);"},
		{ID: "nested-if", Code: "if (x) { if (!a in b) { } }"},
		{ID: "several", Code: "!a in b; !c instanceof d; !e < f;", Options: ordering},

		// ---- clean files -------------------------------------------------------
		{ID: "clean", Code: "if (!(key in object)) { }\nif (!(obj instanceof Ctor)) { }\n"},
		{ID: "clean-empty", Code: "\n"},

		// ---- non-ASCII ---------------------------------------------------------
		{ID: "non-ascii", Code: "if (!é in b) { }"},
		{ID: "non-ascii-string", Code: "!\"é\" in b;"},

		// ---- severity ----------------------------------------------------------
		{ID: "warn", Code: "!a in b;", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "!a in b;", Config: map[string]any{"rules": map[string]any{Name: "off"}}},
		{ID: "warn-ordering", Code: "!a < b;",
			Config: map[string]any{"rules": map[string]any{Name: []any{"warn", map[string]any{"enforceForOrderingRelations": true}}}}},
	})
}
