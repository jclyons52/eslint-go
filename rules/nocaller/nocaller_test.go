package nocaller

import (
	"reflect"
	"testing"

	eslint "github.com/jclyons52/eslint-go"
	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// TestParity is expected to FAIL until acorn-go stops treating `arguments` as
// a reserved word in identifier *reference* position.
//
// The rule itself is ported in full and every case below reproduces the
// oracle's message exactly (ruleId, severity, message, messageId, nodeType,
// line, column, endLine, endColumn — verified programmatically: 0 of the 26
// failing cases differ for any reason other than the parse error). What the
// Go side cannot reproduce is the *parse*: no-caller's only trigger is
// `arguments.callee` / `arguments.caller`, and acorn-go raises
//
//	Parsing error: The keyword 'arguments' is reserved
//
// for every occurrence of the identifier `arguments`. The parser is
// module-only (comments.go: `inModule: true, strict: true`), so this fires on
// all module code, while real acorn parses `arguments.callee` as a plain
// member access (the oracle reports the rule violation, the Go side reports a
// fatal parse error instead).
//
// The divergence is in acorn-go, not in this package: Parser.parseExprAtom
// (expression.go) parses the identifier of an expression atom with
// `p.parseIdent(false)`, which runs checkUnreserved; real acorn calls
// `this.parseIdent(this.options.allowReserved !== "never")` there, i.e.
// liberal=true, which skips the check. checkUnreserved must only run in
// binding positions (acorn's reservedWordsStrictBind = the strict reserved
// words plus `eval` and `arguments`). The same gap makes no-unused-vars'
// `arguments` cases fail (2 of its 5 mismatches).
//
// Do not treat a green run of this file as parity until that lands.
func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- the two property names the rule flags --------------------------
		{ID: "callee", Code: "function f() { return arguments.callee; }"},
		{ID: "caller", Code: "function f() { return arguments.caller; }"},
		{ID: "both", Code: "function f() { return arguments.callee + arguments.caller; }"},
		{ID: "several", Code: "function f() { return arguments.callee && arguments.caller; }"},

		// ---- other non-computed property names (no report) ------------------
		{ID: "other-property", Code: "function f() { return arguments.length; }"},
		{ID: "similar-name", Code: "function f() { return arguments.callex; }"},
		{ID: "prefix-name", Code: "function f() { return arguments.callee2; }"},
		{ID: "suffix-name", Code: "function f() { return arguments.xcallee; }"},
		{ID: "capitalised", Code: "function f() { return arguments.Callee; }"},

		// ---- computed members are never flagged ------------------------------
		{ID: "computed-string", Code: "function f() { return arguments[\"callee\"]; }"},
		{ID: "computed-caller", Code: "function f() { return arguments[\"caller\"]; }"},
		{ID: "computed-identifier", Code: "function f() { const p = \"callee\"; return arguments[p]; }"},

		// ---- the object must be the identifier `arguments` -------------------
		{ID: "other-object", Code: "function f() { return foo.callee; }"},
		{ID: "this-object", Code: "function f() { return this.callee; }"},
		{ID: "call-object", Code: "function f() { return getArgs().callee; }"},
		{ID: "member-object", Code: "function f() { return arguments.obj.callee; }"},
		{ID: "parenthesised-object", Code: "function f() { return (arguments).callee; }"},

		// ---- nesting and positions -------------------------------------------
		{ID: "nested-function", Code: "function a() { function b() { return arguments.callee; } }"},
		{ID: "arrow-function", Code: "const f = () => arguments.caller;"},
		{ID: "class-method", Code: "class C { m() { return arguments.callee; } }"},
		{ID: "object-method", Code: "const o = { m() { return arguments.caller; } };"},
		{ID: "call-expression", Code: "function f() { return arguments.callee(); }"},
		{ID: "assignment-target", Code: "function f() { arguments.callee = 1; }"},
		{ID: "member-chain", Code: "function f() { return arguments.callee.name; }"},

		// ---- clean files ------------------------------------------------------
		{ID: "clean", Code: "function f() { return arguments.length + foo.callee; }"},
		{ID: "clean-empty", Code: "\n"},

		// ---- comments and non-ASCII ------------------------------------------
		{ID: "comment", Code: "function f() { return arguments /* c */.callee; }"},
		{ID: "non-ascii", Code: "function f() { return arguments.callee; // é\n}"},

		// ---- severity ---------------------------------------------------------
		{ID: "warn", Code: "function f() { return arguments.callee; }", Config: map[string]any{
			"rules": map[string]any{Name: "warn"},
		}},
		{ID: "off", Code: "function f() { return arguments.callee; }", Config: map[string]any{
			"rules": map[string]any{Name: "off"},
		}},
	})
}

// TestMeta checks the ported meta block against eslint@8.57.0's
// lib/rules/no-caller.js: type, docs, fixable (absent there, "" here),
// messages and schema.
func TestMeta(t *testing.T) {
	want := eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Disallow the use of `arguments.caller` or `arguments.callee`",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/no-caller",
		},
		Messages: map[string]string{
			"unexpected": "Avoid arguments.{{prop}}.",
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
