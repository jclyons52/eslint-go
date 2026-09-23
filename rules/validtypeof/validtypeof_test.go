package validtypeof

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// builtinEnv is the config ESLint's Linter always runs with: it resolves
// `env: { builtin: true }` implicitly (linter.js: `Object.assign({ builtin:
// true }, config.env, envInFile)`). The Go core only applies the envs a config
// names, so scope-sensitive cases (the `typeof x === undefined` branch, which
// needs the global `undefined` variable) must spell it out. It is a no-op for
// the oracle.
func builtinEnv() map[string]any {
	return map[string]any{"env": map[string]any{"builtin": true}}
}

// TestParity is expected to FAIL on its five `undefined`-branch cases until the
// core can emit rule suggestions.
//
// valid-typeof sets `meta.hasSuggestions: true` and attaches a `suggest` entry
// to the two reports on the `typeof x === undefined` branch (messageId
// invalidValue, or notString under requireStringLiterals). eslint.Message has
// no `suggestions` field and eslint.Report has no `Suggest` field, so those
// five cases differ from the oracle by exactly that one key and by nothing else
// (verified: 0 non-suggestion differences). Every other branch — literal and
// static-template values, the valid-type list, non-string siblings, the
// requireStringLiterals report — matches the oracle byte for byte.
//
// The corpus keeps those five cases on purpose: `isReferenceToGlobalVariable`
// is a whole branch of the rule, and dropping its cases to go green would leave
// it unverified. Making them pass needs a core change:
//
//	Message: add `Suggestions []Suggestion` (+ MarshalJSON emission, in
//	         ESLint's key order: after endLine/endColumn, before fix) and
//	Report:  add `Suggest []SuggestDescriptor`.
//
// Do not treat a green run of this file as parity until that lands.
func TestParity(t *testing.T) {
	reqString := []any{map[string]any{"requireStringLiterals": true}}

	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- doc examples ----------------------------------------------------
		{ID: "doc-incorrect-1", Code: "if (typeof foo === \"strnig\") {}"},
		{ID: "doc-incorrect-2", Code: "if (typeof foo == \"undefimed\") {}"},
		{ID: "doc-incorrect-3", Code: "if (typeof foo != \"nunber\") {}"},
		{ID: "doc-incorrect-4", Code: "if (typeof foo !== \"fucntion\") {}"},
		{ID: "doc-correct-1", Code: "if (typeof foo === \"string\") {}"},
		{ID: "doc-correct-2", Code: "if (typeof foo !== \"undefined\") {}"},

		// ---- every valid type string, both operand positions -----------------
		{ID: "valid-types-right", Code: "typeof x === \"symbol\"; typeof x === \"undefined\"; typeof x === \"object\"; typeof x === \"boolean\"; typeof x === \"number\"; typeof x === \"string\"; typeof x === \"function\"; typeof x === \"bigint\";"},
		{ID: "valid-types-left", Code: "\"symbol\" === typeof x; \"number\" == typeof x; \"bigint\" != typeof x; \"object\" !== typeof x;"},
		{ID: "invalid-types", Code: "typeof x === \"strnig\"; typeof x === \"\"; typeof x === \"Symbol\"; typeof x === \"Number\";"},

		// ---- all four operators ----------------------------------------------
		{ID: "operators", Code: "typeof x == \"nope\"; typeof x === \"nope\"; typeof x != \"nope\"; typeof x !== \"nope\";"},
		{ID: "non-comparison-operators", Code: "typeof x + \"string\"; typeof x - \"string\"; typeof x in obj; typeof x instanceof Object;"},
		{ID: "logical", Code: "typeof x && \"nope\"; typeof x || \"nope\";"},

		// ---- non-string literal siblings -------------------------------------
		{ID: "number-literal", Code: "typeof x === 5;"},
		{ID: "null-literal", Code: "typeof x === null;"},
		{ID: "boolean-literal", Code: "typeof x === true;"},
		{ID: "bigint-literal", Code: "typeof x === 1n;"},
		{ID: "regex-literal", Code: "typeof x === /a/;"},
		{ID: "array", Code: "typeof x === [];"},
		{ID: "object", Code: "typeof x === {};"},
		{ID: "call", Code: "typeof x === f();"},

		// ---- template literals ------------------------------------------------
		{ID: "template-valid", Code: "typeof x === `string`;"},
		{ID: "template-invalid", Code: "typeof x === `strnig`;"},
		{ID: "template-empty", Code: "typeof x === ``;"},
		{ID: "template-interpolated", Code: "typeof x === `str${y}ng`;"},
		{ID: "template-left", Code: "`number` === typeof x;"},
		{ID: "template-escaped", Code: "typeof x === `\\u0073tring`;"},
		{ID: "template-invalid-escape", Code: "typeof x === `\\unicode`;"},

		// ---- identifier siblings (default: silent) ---------------------------
		{ID: "identifier-sibling", Code: "typeof x === y;"},
		{ID: "identifier-sibling-neq", Code: "typeof x !== y;"},
		{ID: "member-sibling", Code: "typeof x === obj.undefined;"},
		{ID: "member-sibling-other", Code: "typeof x === obj.y;"},

		// ---- the global `undefined` branch (invalidValue + suggest) ----------
		{ID: "undefined-global", Code: "typeof x === undefined;", Config: builtinEnv()},
		{ID: "undefined-global-neq", Code: "typeof x !== undefined;", Config: builtinEnv()},
		{ID: "undefined-global-left", Code: "undefined === typeof x;", Config: builtinEnv()},
		{ID: "undefined-global-warn", Code: "typeof x === undefined;",
			Config: map[string]any{"rules": map[string]any{Name: "warn"}, "env": map[string]any{"builtin": true}}},
		{ID: "undefined-declared", Code: "var undefined = 1; typeof x === undefined;", Config: builtinEnv()},
		{ID: "undefined-shadowed-param", Code: "function f(undefined) { return typeof x === undefined; }", Config: builtinEnv()},
		{ID: "undefined-shadowed-block", Code: "{ let undefined = 1; typeof x === undefined; }", Config: builtinEnv()},
		{ID: "undefined-shadowed-nested", Code: "function f() { var undefined = 1; return typeof x === undefined; }", Config: builtinEnv()},
		{ID: "undefined-global-require-string", Code: "typeof x === undefined;", Options: reqString, Config: builtinEnv()},
		{ID: "undefined-declared-require-string", Code: "var undefined = 1; typeof x === undefined;", Options: reqString, Config: builtinEnv()},
		{ID: "undefined-non-comparison", Code: "typeof x + undefined;", Config: builtinEnv()},
		{ID: "undefined-in-switch", Code: "switch (typeof x) { case undefined: break; }", Config: builtinEnv()},

		// ---- requireStringLiterals -------------------------------------------
		{ID: "req-ident", Code: "typeof x === y;", Options: reqString},
		{ID: "req-ident-neq", Code: "typeof x !== y;", Options: reqString},
		{ID: "req-typeof-typeof", Code: "typeof x === typeof y;", Options: reqString},
		{ID: "req-typeof-literal", Code: "typeof x === \"string\";", Options: reqString},
		{ID: "req-typeof-invalid-literal", Code: "typeof x === \"strnig\";", Options: reqString},
		{ID: "req-typeof-template", Code: "typeof x === `string`;", Options: reqString},
		{ID: "req-member", Code: "typeof x === obj.y;", Options: reqString},
		{ID: "req-call", Code: "typeof x === f();", Options: reqString},
		{ID: "req-number", Code: "typeof x === 5;", Options: reqString},
		{ID: "req-null", Code: "typeof x === null;", Options: reqString},
		{ID: "req-array", Code: "typeof x === [];", Options: reqString},
		{ID: "req-conditional", Code: "typeof x === (y ? \"string\" : \"number\");", Options: reqString},
		{ID: "req-binary", Code: "typeof x === (y + \"\");", Options: reqString},
		{ID: "req-non-comparison", Code: "typeof x + y;", Options: reqString},
		{ID: "req-false", Code: "typeof x === y;", Options: []any{map[string]any{"requireStringLiterals": false}}},
		{ID: "req-empty-object", Code: "typeof x === y;", Options: []any{map[string]any{}}},

		// ---- nesting and parentheses -----------------------------------------
		{ID: "parens-sibling", Code: "typeof x === (\"strnig\");"},
		{ID: "parens-argument", Code: "typeof (x) === \"strnig\";"},
		{ID: "typeof-typeof", Code: "typeof typeof x === \"strnig\";"},
		{ID: "typeof-typeof-valid", Code: "typeof typeof x === \"string\";"},
		{ID: "chained", Code: "typeof x === typeof y === \"string\";"},
		{ID: "nested-call", Code: "f(typeof x === \"strnig\");"},
		{ID: "in-if", Code: "if (typeof x === \"strnig\") { } else if (typeof y !== \"nunber\") { }"},
		{ID: "conditional", Code: "const r = typeof x === \"strnig\" ? 1 : 2;"},
		{ID: "multiline", Code: "const r = typeof x\n    ===\n    \"strnig\";"},
		{ID: "comment", Code: "typeof x /* c */ === \"strnig\";"},

		// ---- non-typeof unary expressions ------------------------------------
		{ID: "unary-void", Code: "void x === \"strnig\";"},
		{ID: "unary-minus", Code: "-x === \"strnig\";"},
		{ID: "unary-delete", Code: "delete x.y === \"strnig\";"},
		{ID: "bare-typeof", Code: "typeof x;"},

		// ---- clean files ------------------------------------------------------
		{ID: "clean", Code: "if (typeof foo === \"string\") { }\nif (typeof bar !== \"undefined\") { }\nconst t = typeof baz;\n"},
		{ID: "clean-empty", Code: "\n"},

		// ---- non-ASCII --------------------------------------------------------
		{ID: "non-ascii", Code: "const é = typeof x === \"strnig\";"},
		{ID: "non-ascii-valid", Code: "const é = typeof x === \"string\";"},

		// ---- severity ---------------------------------------------------------
		{ID: "off", Code: "typeof x === \"strnig\";", Config: map[string]any{"rules": map[string]any{Name: "off"}}},
	})
}
