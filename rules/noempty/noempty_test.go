package noempty

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// TestParity covers both handlers (BlockStatement / SwitchStatement), every
// guard in the BlockStatement handler (non-empty body, function parent,
// allowEmptyCatch, comment inside), and both option values.
//
// KNOWN CORE GAP (docs/porting-rules.md pitfall 6). no-empty sets
// `meta.hasSuggestions: true` and passes a one-entry `suggest` array on *every*
// BlockStatement report, so the oracle's message for those cases carries a
// `suggestions` key:
//
//	"suggestions": [{"messageId":"suggestComment","data":{"type":"block"},
//	                 "fix":{"range":[10,10],"text":" /* empty */ "},
//	                 "desc":"Add comment inside empty block statement."}]
//
// eslint.Message has no `suggestions` field and eslint.Report has no `Suggest`
// field, so the port cannot emit it: every BlockStatement case below differs
// from the oracle by exactly that one key and by nothing else (verified: 0
// non-suggestion differences — the message, messageId, line, column, nodeType
// and endLine/endColumn all match). The SwitchStatement report carries no
// suggestion and therefore matches the oracle byte for byte.
//
// The corpus keeps the suggestion-bearing cases on purpose: the BlockStatement
// report *is* the rule. Dropping them to go green would leave the rule
// unverified. Making them pass needs a core change:
//
//	Message: add `Suggestions []Suggestion` (+ MarshalJSON emission, in
//	         ESLint's key order: after endLine/endColumn, before fix) and
//	Report:  add `Suggest []SuggestDescriptor`.
//
// Do not treat a green run of this file as parity until that lands.
func TestParity(t *testing.T) {
	allowEmptyCatch := []any{map[string]any{"allowEmptyCatch": true}}
	allowEmptyCatchOff := []any{map[string]any{"allowEmptyCatch": false}}

	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- BlockStatement: the suggestion-bearing reports ----------------
		{ID: "if-block", Code: "if (foo) {}"},
		{ID: "if-block-multiline", Code: "if (foo) {\n}"},
		{ID: "else-block", Code: "if (foo) { bar(); } else {}"},
		{ID: "while-block", Code: "while (foo) {}"},
		{ID: "for-block", Code: "for (;;) {}"},
		{ID: "do-while-block", Code: "do {} while (foo);"},
		{ID: "standalone-block", Code: "{}"},
		{ID: "labelled-block", Code: "foo: {}"},
		{ID: "finally-block", Code: "try { bar(); } finally {}"},
		{ID: "nested-block", Code: "if (foo) { if (bar) {} }"},
		{ID: "catch-block", Code: "try { foo(); } catch (e) {}"},

		// ---- BlockStatement: guarded, no report ----------------------------
		{ID: "function-body", Code: "function foo() {}"},
		{ID: "arrow-body", Code: "const f = () => {};"},
		{ID: "method-body", Code: "class C { m() {} }"},
		{ID: "constructor-body", Code: "class C { constructor() {} }"},
		{ID: "getter-body", Code: "class C { get x() {} }"},
		{ID: "comment-inside", Code: "if (foo) { /* comment */ }"},
		{ID: "line-comment-inside", Code: "if (foo) {\n  // comment\n}"},
		{ID: "non-empty", Code: "if (foo) { bar(); }"},
		{ID: "catch-block-allowed", Code: "try { foo(); } catch (e) {}", Options: allowEmptyCatch},
		{ID: "if-block-not-catch", Code: "if (foo) {}", Options: allowEmptyCatch},
		{ID: "catch-block-explicit-off", Code: "try { foo(); } catch (e) {}", Options: allowEmptyCatchOff},

		// ---- SwitchStatement: no suggestion, matches exactly ---------------
		{ID: "empty-switch", Code: "switch (foo) {}"},
		{ID: "non-empty-switch", Code: "switch (foo) { case 1: break; }"},
		{ID: "switch-default-only", Code: "switch (foo) { default: break; }"},

		// ---- non-ASCII ------------------------------------------------------
		{ID: "non-ascii", Code: "if (caf\u00e9) {}"},

		// ---- severity -------------------------------------------------------
		{ID: "warn", Code: "switch (foo) {}",
			Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "if (foo) {}",
			Config: map[string]any{"rules": map[string]any{Name: "off"}}},
	})
}
