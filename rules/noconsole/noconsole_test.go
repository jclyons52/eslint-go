package noconsole

import (
	"reflect"
	"testing"

	eslint "github.com/jclyons52/eslint-go"
	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// TestParity is expected to FAIL until the core can emit rule suggestions.
//
// The rule is ported in full and every case below reproduces the oracle's
// message exactly — the `Program:exit` + `getVariableByName(scope, "console")`
// shape, the `scope.through` fallback, the shadowed-variable guard, the
// `isMemberAccessExceptAllowed` test and the `allow` option all match. What it
// cannot reproduce is the oracle's `suggestions` array: no-console sets
// `meta.hasSuggestions: true` and, when the reported MemberExpression is the
// callee of an ExpressionStatement in a statement list and removing it cannot
// break ASI (`canProvideSuggestions`), passes a one-entry `removeConsole`
// suggestion. eslint.Message has no `suggestions` field and eslint.Report has
// no `Suggest` field (docs/porting-rules.md pitfall 6 — "the message shape is
// not implemented"). The 14 affected cases below therefore differ from the
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
	allowLog := []any{map[string]any{"allow": []any{"log"}}}
	allowErrorWarn := []any{map[string]any{"allow": []any{"error", "warn"}}}
	allowEmptyObject := []any{map[string]any{}}

	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- the basic member accesses ----------------------------------------
		{ID: "log-call", Code: "console.log(\"hello\");"},
		{ID: "error-call", Code: "console.error(\"e\");"},
		{ID: "several-calls", Code: "console.log(1); console.error(2);"},
		{ID: "multi-args", Code: "console.log(a, b, c);"},

		// ---- not a call: a member access all the same --------------------------
		{ID: "bare-member", Code: "console.log;"},
		{ID: "assigned", Code: "var x = console.log;"},
		{ID: "as-argument", Code: "foo(console.log);"},
		{ID: "new-member", Code: "new console.log();"},
		{ID: "in-object", Code: "({ log: console.log });"},

		// ---- computed access ----------------------------------------------------
		{ID: "computed-string", Code: "console[\"log\"](\"x\");"},
		{ID: "computed-identifier", Code: "console[method]();"},

		// ---- the allow option ----------------------------------------------------
		{ID: "allow-log", Code: "console.log(1);", Options: allowLog},
		{ID: "allow-log-other-reported", Code: "console.error(1);", Options: allowLog},
		{ID: "allow-two", Code: "console.error(1); console.warn(2); console.log(3);", Options: allowErrorWarn},
		{ID: "allow-computed", Code: "console[\"log\"](1);", Options: allowLog},
		{ID: "allow-identifier-not-allowed", Code: "console[method](1);", Options: allowLog},
		{ID: "allow-empty-object", Code: "console.log(1);", Options: allowEmptyObject},

		// ---- a declared `console` shadows the global -----------------------------
		{ID: "shadowed-var", Code: "var console = { log() {} }; console.log(1);"},
		{ID: "shadowed-param", Code: "function f(console) { console.log(1); }"},
		{ID: "shadowed-block", Code: "{ let console = 1; console.log(); }"},
		{ID: "shadowed-nested", Code: "function f() { var console = {}; console.log(); }"},

		// ---- references that are not member accesses ------------------------------
		{ID: "bare-console", Code: "console;"},
		{ID: "typeof-console", Code: "typeof console;"},
		{ID: "call-console", Code: "console();"},

		// ---- statement positions (the suggestion branch) ---------------------------
		{ID: "if-body", Code: "if (a) console.log(b);"},
		{ID: "block-body", Code: "if (a) { console.log(b); }"},
		{ID: "asi-hazard", Code: "foo()\nconsole.log();\n[1, 2, 3].forEach(f)"},

		// ---- clean files ------------------------------------------------------------
		{ID: "clean", Code: "const x = 1;\nfoo();\n"},
		{ID: "clean-empty", Code: "\n"},

		// ---- comments and non-ASCII --------------------------------------------------
		{ID: "comment", Code: "console /* c */.log(1);"},
		{ID: "non-ascii", Code: "console.log(\"é\");"},

		// ---- severity -----------------------------------------------------------------
		{ID: "warn", Code: "console.log(1);", Config: map[string]any{
			"rules": map[string]any{Name: "warn"},
		}},
		{ID: "off", Code: "console.log(1);", Config: map[string]any{
			"rules": map[string]any{Name: "off"},
		}},
	})
}

// TestMeta checks the ported meta block against eslint@8.57.0's
// lib/rules/no-console.js: type, docs, fixable (absent there, "" — the rule is
// suggestion-only), messages and schema. The original also sets
// `hasSuggestions: true`; eslint.RuleMeta has no such field (the core cannot
// emit suggestions — see TestParity).
func TestMeta(t *testing.T) {
	want := eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow the use of `console`",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-console",
		},
		Messages: map[string]string{
			"unexpected":    "Unexpected console statement.",
			"removeConsole": "Remove the console.{{ propertyName }}().",
		},
		Schema: []any{
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"allow": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
						"minItems":    1,
						"uniqueItems": true,
					},
				},
				"additionalProperties": false,
			},
		},
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
