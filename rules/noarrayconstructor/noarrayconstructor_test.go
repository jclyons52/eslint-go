package noarrayconstructor

import (
	"reflect"
	"testing"

	eslint "github.com/jclyons52/eslint-go"
	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// builtinEnv is the environment ESLint's Linter always resolves implicitly
// (`Object.assign({ builtin: true }, config.env, envInFile)` in linter.js).
// `Array` is one of its predefined globals, and the rule only reports when
// `getVariableByName(scope, "Array")` finds a variable with no declarations —
// which is exactly the predefined-global shape. The Go core applies only the
// envs a config names, so the corpus spells `builtin` out (a no-op for the
// oracle).
func builtinEnv() map[string]any {
	return map[string]any{"env": map[string]any{"builtin": true}}
}

// TestParity is expected to FAIL until the core can emit rule suggestions.
//
// The rule is ported in full and every reporting case below reproduces the
// oracle's message exactly — the `Array` callee check, the single
// non-spread-argument exception, the predefined-global test
// (`getVariableByName(scope, "Array")` with an empty `identifiers` list),
// messageId, node and loc. What it cannot reproduce is the oracle's
// `suggestions` array: no-array-constructor sets `meta.hasSuggestions: true`
// and passes a one-entry `suggest` list on *every* report, while
// eslint.Message has no `suggestions` field and eslint.Report has no `Suggest`
// field (docs/porting-rules.md pitfall 6 — "the message shape is not
// implemented"). The 16 reporting cases below therefore differ from the
// oracle by exactly one key, `suggestions`, and by nothing else (verified
// programmatically: stripping `suggestions` from the oracle output makes all
// 33 cases equal).
//
// Making them pass needs a core change:
//
//	Message: add `Suggestions []Suggestion` (+ MarshalJSON emission, in
//	         ESLint's key order: after endLine/endColumn, before fix) and
//	Report:  add `Suggest []SuggestDescriptor`.
//
// Do not treat a green run of this file as parity until that lands.
func TestParity(t *testing.T) {
	ruletest.CompareWith(t, Rule, []ruletest.Case{
		// ---- doc examples ----------------------------------------------------
		{ID: "doc-new-no-args", Code: "new Array();"},
		{ID: "doc-new-args", Code: "new Array(1, 2, 3);"},
		{ID: "call-no-args", Code: "Array();"},
		{ID: "call-args", Code: "Array(1, 2, 3);"},
		{ID: "correct-literal", Code: "const a = [1, 2, 3];"},

		// ---- the single-argument exception ------------------------------------
		{ID: "new-single-number", Code: "new Array(5);"},
		{ID: "call-single-number", Code: "Array(5);"},
		{ID: "new-single-string", Code: "new Array(\"5\");"},
		{ID: "new-single-spread", Code: "new Array(...args);"},
		{ID: "call-single-spread", Code: "Array(...args);"},
		{ID: "new-two-args-one-spread", Code: "new Array(1, ...args);"},

		// ---- argument-count and callee shapes ----------------------------------
		{ID: "new-no-parens", Code: "new Array;"},
		{ID: "new-two-args", Code: "new Array(1, 2);"},
		{ID: "new-arg-call", Code: "new Array(f());"},
		{ID: "member-callee", Code: "foo.Array(1, 2);"},
		{ID: "window-callee", Code: "window.Array(1, 2);"},
		{ID: "call-method", Code: "Array.call(null, 1, 2);"},
		{ID: "new-member", Code: "new foo.Array(1, 2);"},

		// ---- shadowing: `Array` is not the predefined global --------------------
		{ID: "shadowed-var", Code: "var Array = 1; new Array(1, 2);"},
		{ID: "shadowed-let-block", Code: "{ let Array = 1; new Array(1, 2); }"},
		{ID: "shadowed-param", Code: "function f(Array) { return new Array(1, 2); }"},
		{ID: "shadowed-function", Code: "function Array() {} Array(1, 2);"},
		{ID: "shadowed-import", Code: "import Array from \"mod\"; new Array(1, 2);"},

		// ---- nesting -------------------------------------------------------------
		{ID: "nested-inner-only", Code: "new Array(new Array(1, 2));"},
		{ID: "nested-both", Code: "new Array(1, 2, new Array(3, 4));"},
		{ID: "member-of-result", Code: "new Array(1, 2).length;"},

		// ---- clean files ----------------------------------------------------------
		{ID: "clean", Code: "const a = [];\nconst b = [1];\n"},
		{ID: "clean-empty", Code: "\n"},

		// ---- comments, ASI and non-ASCII ------------------------------------------
		{ID: "comment", Code: "new /* c */ Array(1, 2);"},
		{ID: "asi-line", Code: "let a = 1\nnew Array(1, 2)"},
		{ID: "non-ascii", Code: "const é = 1; new Array(1, 2);"},

		// ---- severity --------------------------------------------------------------
		{ID: "warn", Code: "new Array(1, 2);", Config: map[string]any{
			"rules": map[string]any{Name: "warn"}, "env": map[string]any{"builtin": true},
		}},
		{ID: "off", Code: "new Array(1, 2);", Config: map[string]any{
			"rules": map[string]any{Name: "off"}, "env": map[string]any{"builtin": true},
		}},
	}, ruletest.Options{ExtraConfig: builtinEnv()})
}

// TestMeta checks the ported meta block against eslint@8.57.0's
// lib/rules/no-array-constructor.js: type, docs, fixable (absent there, "" —
// the rule is suggestion-only), messages and schema. The original also sets
// `hasSuggestions: true`; eslint.RuleMeta has no such field (the core cannot
// emit suggestions — see TestParity).
func TestMeta(t *testing.T) {
	want := eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow `Array` constructors",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-array-constructor",
		},
		Messages: map[string]string{
			"preferLiteral":            "The array literal notation [] is preferable.",
			"useLiteral":               "Replace with an array literal.",
			"useLiteralAfterSemicolon": "Replace with an array literal, add preceding semicolon.",
		},
		Schema: []any{},
	}
	if Rule.Meta.Type != want.Type {
		t.Errorf("Type = %q, want %q", Rule.Meta.Type, want.Type)
	}
	if Rule.Meta.Fixable != "" {
		t.Errorf("Fixable = %q, want \"\" (the original declares no fixable)", Rule.Meta.Fixable)
	}
	if !reflect.DeepEqual(Rule.Meta.Docs, want.Docs) {
		t.Errorf("Docs = %+v, want %+v", Rule.Meta.Docs, want.Docs)
	}
	if !reflect.DeepEqual(Rule.Meta.Messages, want.Messages) {
		t.Errorf("Messages = %+v, want %+v", Rule.Meta.Messages, want.Messages)
	}
	if !reflect.DeepEqual(Rule.Meta.Schema, want.Schema) {
		t.Errorf("Schema = %#v, want %#v", Rule.Meta.Schema, want.Schema)
	}
}
