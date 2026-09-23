package noextrabooleancast

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// logical is the enforceForLogicalOperands option value.
var logical = []any{map[string]any{"enforceForLogicalOperands": true}}

// parserOptions mirrors what the oracle driver always runs with (ecmaVersion
// 2022, sourceType module), so both implementations lint the same program.
var parserOptions = map[string]any{"ecmaVersion": 2022, "sourceType": "module"}

func TestParity(t *testing.T) {
	ruletest.CompareWith(t, Rule, []ruletest.Case{
		// Doc examples.
		{ID: "doc-negation", Code: "if (!!foo) {}"},
		{ID: "doc-call", Code: "if (Boolean(foo)) {}"},
		{ID: "doc-valid", Code: "if (foo) {}"},
		{ID: "doc-valid-call", Code: "var foo = Boolean(bar);"},

		// Every boolean-context node type.
		{ID: "while", Code: "while (!!foo) {}"},
		{ID: "do-while", Code: "do {} while (!!foo);"},
		{ID: "for", Code: "for (; !!foo;) {}"},
		{ID: "conditional-negation", Code: "!!foo ? bar : baz;"},
		{ID: "conditional-call", Code: "Boolean(foo) ? bar : baz;"},
		{ID: "negated-call", Code: "if (!Boolean(foo)) {}"},

		// Not a boolean context: no report.
		{ID: "assignment-negation", Code: "var x = !!foo;"},
		{ID: "assignment-call", Code: "var x = Boolean(foo);"},
		{ID: "single-negation", Code: "if (!foo) {}"},
		{ID: "return-call", Code: "function f() { return Boolean(foo); }"},

		// Nested negations report once per `!!` pair.
		{ID: "quadruple", Code: "if (!!!!foo) {}"},
		{ID: "call-of-negation", Code: "if (Boolean(!!foo)) {}"},

		// The zero/two argument forms of the call.
		{ID: "call-zero-args", Code: "if (Boolean()) {}"},
		{ID: "negated-call-zero-args", Code: "if (!Boolean()) {}"},
		{ID: "call-two-args", Code: "if (Boolean(a, b)) {}"},
		{ID: "call-spread", Code: "if (Boolean(...foo)) {}"},

		// enforceForLogicalOperands.
		{ID: "logical-default-off", Code: "if (!!foo || bar) {}"},
		{ID: "logical-enabled-negation", Code: "if (!!foo || bar) {}", Options: logical},
		{ID: "logical-enabled-both", Code: "if (!!foo && !!bar) {}", Options: logical},
		{ID: "logical-enabled-call", Code: "if (Boolean(foo) || bar) {}", Options: logical},
		{ID: "logical-disabled-call", Code: "if (Boolean(foo) || bar) {}"},
		{ID: "logical-enabled-nested", Code: "if (!!(foo || !!bar)) {}", Options: logical},
		{ID: "logical-enabled-not-boolean-context", Code: "var x = !!foo || bar;", Options: logical},

		// Optional chaining (ChainExpression).
		{ID: "chain-negation", Code: "if (!!foo?.bar) {}"},
		{ID: "chain-call", Code: "if (Boolean(foo?.bar)) {}"},

		// Comments inside the reported node disable the fix.
		{ID: "comment-in-call", Code: "if (Boolean(/* c */ foo)) {}"},
		{ID: "comment-in-negation", Code: "if (!!/* c */ foo) {}"},

		// Severity.
		{ID: "warn", Code: "if (!!foo) {}", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "if (!!foo) {}", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Clean file and non-ASCII source.
		{ID: "clean", Code: "if (foo) {}\nwhile (bar) {}\nfor (; baz;) {}\n"},
		{ID: "non-ascii", Code: "if (!!é) {}"},

		// Fixes.
		{ID: "fix-negation", Code: "if (!!foo) {}", Fix: true},
		{ID: "fix-call", Code: "if (Boolean(foo)) {}", Fix: true},
		{ID: "fix-call-zero-args", Code: "if (Boolean()) {}", Fix: true},
		{ID: "fix-negated-call", Code: "if (!Boolean(foo)) {}", Fix: true},
		{ID: "fix-negated-call-zero-args", Code: "if (!Boolean()) {}", Fix: true},
		{ID: "fix-conditional", Code: "!!foo ? bar : baz;", Fix: true},
		{ID: "fix-sequence-in-call", Code: "if (Boolean(!!(a, b))) {}", Fix: true},
		{ID: "fix-sequence", Code: "if (!!(a, b)) {}", Fix: true},
		{ID: "fix-binary-in-call", Code: "if (!Boolean(a + b)) {}", Fix: true},
		{ID: "fix-comment", Code: "if (Boolean(/* c */ foo)) {}", Fix: true},
		{ID: "fix-two-args", Code: "if (Boolean(a, b)) {}", Fix: true},
		{ID: "fix-spread", Code: "if (Boolean(...foo)) {}", Fix: true},
		{ID: "fix-logical", Code: "if (!!foo || bar) {}", Options: logical, Fix: true},
		{ID: "fix-multiple", Code: "if (!!foo && Boolean(bar)) {}", Fix: true},
		{ID: "fix-quadruple", Code: "if (!!!!foo) {}", Fix: true},
		{ID: "fix-assignment", Code: "if (Boolean(a = b)) {}", Fix: true},
		{ID: "fix-multiline", Code: "if (\n    Boolean(\n        foo\n    )\n) {}", Fix: true},
		{ID: "fix-non-ascii", Code: "if (!!é) {}", Fix: true},
		{ID: "fix-conditional-in-call", Code: "if (Boolean(a ? b : c)) {}", Fix: true},
		{ID: "fix-logical-in-call", Code: "if (!Boolean(a || b)) {}", Fix: true},
		{ID: "fix-arrow-in-call", Code: "if (Boolean(() => x)) {}", Fix: true},
		{ID: "fix-triple-negation", Code: "if (!!!foo) {}", Fix: true},
		{ID: "fix-multiline-negation", Code: "if (\n    !!foo\n) {}", Fix: true},
	}, ruletest.Options{ExtraConfig: map[string]any{"parserOptions": parserOptions}})
}
