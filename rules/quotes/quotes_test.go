package quotes

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Default: double.
		{ID: "default-single", Code: "var a = 'x';", Fix: true},
		{ID: "default-ok", Code: "var a = \"x\";"},
		{ID: "default-single-escaped", Code: "var a = 'it\\'s';", Fix: true},
		{ID: "default-directive-ok", Code: "\"use strict\";"},
		{ID: "default-directive-single", Code: "'use strict';", Fix: true},
		{ID: "default-directive-in-function", Code: "function f() { 'use strict'; }", Fix: true},
		{ID: "default-non-string-literals", Code: "var a = 1; var b = true; var c = null; var r = /x/g;"},
		{ID: "default-bigint", Code: "var a = 1n;"},
		{ID: "default-nested", Code: "var o = { a: 'x', b: { c: 'y' } };", Fix: true},
		{ID: "default-mixed", Code: "var a = 'x';\nvar b = \"y\";\nvar c = 'z';", Fix: true},
		{ID: "default-warn", Code: "var a = 'x';", Config: map[string]any{
			"rules": map[string]any{Name: []any{1}},
		}, Fix: true},

		// single.
		{ID: "single-double", Code: "var a = \"x\";", Options: []any{"single"}, Fix: true},
		{ID: "single-ok", Code: "var a = 'x';", Options: []any{"single"}},
		{ID: "single-escape", Code: "var a = \"it's\";", Options: []any{"single"}, Fix: true},
		{ID: "single-unescape", Code: "var a = \"a\\\"b\";", Options: []any{"single"}, Fix: true},
		{ID: "single-property-key", Code: "var o = { \"k\": 1 };", Options: []any{"single"}, Fix: true},
		{ID: "single-import-source", Code: "import x from \"m\";", Options: []any{"single"}, Fix: true},
		{ID: "single-export-star", Code: "export * from \"m\";", Options: []any{"single"}, Fix: true},
		{ID: "single-call-args", Code: "f(\"a\", \"b\");", Options: []any{"single"}, Fix: true},
		{ID: "single-array", Code: "var a = [\"x\", \"y\"];", Options: []any{"single"}, Fix: true},
		{ID: "single-computed-key", Code: "var o = { [\"k\"]: 1 };", Options: []any{"single"}, Fix: true},

		// backtick.
		{ID: "backtick-double", Code: "var a = \"x\";", Options: []any{"backtick"}, Fix: true},
		{ID: "backtick-single", Code: "var a = 'x';", Options: []any{"backtick"}, Fix: true},
		{ID: "backtick-property-key-ok", Code: "var o = { \"k\": 1 };", Options: []any{"backtick"}},
		{ID: "backtick-computed-key", Code: "var o = { [\"k\"]: 1 };", Options: []any{"backtick"}, Fix: true},
		{ID: "backtick-directive-ok", Code: "\"use strict\";", Options: []any{"backtick"}},
		{ID: "backtick-directive-single-ok", Code: "'use strict';", Options: []any{"backtick"}},
		{ID: "backtick-directive-parenthesised", Code: "(\"use strict\");", Options: []any{"backtick"}, Fix: true},
		{ID: "backtick-directive-after-nondirective", Code: "var a = 1;\n\"not a directive\";", Options: []any{"backtick"}, Fix: true},
		{ID: "backtick-method-key-ok", Code: "class C { \"m\"() {} }", Options: []any{"backtick"}},
		{ID: "backtick-class-field-ok", Code: "class C { \"f\" = 1; }", Options: []any{"backtick"}},
		{ID: "backtick-class-field-computed", Code: "class C { [\"f\"] = 1; }", Options: []any{"backtick"}, Fix: true},
		{ID: "backtick-import-source-ok", Code: "import x from \"m\";", Options: []any{"backtick"}},
		{ID: "backtick-import-string-name-ok", Code: "import { \"a\" as b } from \"m\";", Options: []any{"backtick"}},
		{ID: "backtick-export-string-name-ok", Code: "var a = 1;\nexport { a as \"b\" };", Options: []any{"backtick"}},
		{ID: "backtick-export-all-as-ok", Code: "export * as \"ns\" from \"m\";", Options: []any{"backtick"}},
		{ID: "backtick-export-source", Code: "export * from \"m\";", Options: []any{"backtick"}},
		{ID: "backtick-template-ok", Code: "var a = `x`;", Options: []any{"backtick"}},
		{ID: "backtick-template-feature-ok", Code: "var a = `x${y}`;", Options: []any{"backtick"}},
		{ID: "backtick-escape-quote", Code: "var a = \"a'b\";", Options: []any{"backtick"}, Fix: true},
		{ID: "backtick-escape-backtick", Code: "var a = \"a`b\";", Options: []any{"backtick"}, Fix: true},
		{ID: "backtick-dollar-brace", Code: "var a = \"a${b}\";", Options: []any{"backtick"}, Fix: true},

		// avoid-escape.
		{ID: "avoid-escape-ok", Code: "var a = \"it's\";", Options: []any{"single", map[string]any{"avoidEscape": true}}},
		{ID: "avoid-escape-double-ok", Code: "var a = 'say \"hi\"';", Options: []any{"double", map[string]any{"avoidEscape": true}}},
		{ID: "avoid-escape-report", Code: "var a = \"x\";", Options: []any{"single", map[string]any{"avoidEscape": true}}, Fix: true},
		{ID: "avoid-escape-string-form", Code: "var a = \"it's\";", Options: []any{"single", "avoid-escape"}},
		{ID: "avoid-escape-string-form-report", Code: "var a = \"x\";", Options: []any{"single", "avoid-escape"}, Fix: true},
		{ID: "avoid-escape-backtick-ok", Code: "var a = \"`\";", Options: []any{"backtick", map[string]any{"avoidEscape": true}}},
		{ID: "avoid-escape-backtick-report", Code: "var a = \"x\";", Options: []any{"backtick", map[string]any{"avoidEscape": true}}, Fix: true},
		{ID: "avoid-escape-not-set", Code: "var a = \"it's\";", Options: []any{"single"}, Fix: true},

		// allowTemplateLiterals.
		{ID: "atl-ok", Code: "var a = `x`;", Options: []any{"double", map[string]any{"allowTemplateLiterals": true}}},
		{ID: "atl-interpolation", Code: "var a = `x${y}`;", Options: []any{"double", map[string]any{"allowTemplateLiterals": true}}},
		{ID: "atl-false", Code: "var a = `x`;", Options: []any{"double", map[string]any{"allowTemplateLiterals": false}}, Fix: true},

		// Template literals without features.
		{ID: "template-double", Code: "var a = `x`;", Fix: true},
		{ID: "template-single", Code: "var a = `x`;", Options: []any{"single"}, Fix: true},
		{ID: "template-interpolation-ok", Code: "var a = `x${y}`;"},
		{ID: "template-multiline-ok", Code: "var a = `x\ny`;"},
		{ID: "template-escaped-newline", Code: "var a = `x\\ny`;", Fix: true},
		{ID: "template-even-backslashes", Code: "var a = `x\\\\\ny`;"},
		{ID: "template-escaped-dollar", Code: "var a = `a\\${b}`;", Fix: true},
		{ID: "template-nested-quotes", Code: "var a = `\"x\"`;", Fix: true},
		{ID: "string-both-quotes", Code: "var a = \"a'b\\\"c\";", Options: []any{"single"}, Fix: true},
		{ID: "template-as-directive", Code: "`use strict`;"},
		{ID: "template-parenthesised", Code: "(`use strict`);", Fix: true},
		{ID: "template-tagged-ok", Code: "var a = tag`x`;"},
		{ID: "template-in-function-directive", Code: "function f() { `x`; }"},
		{ID: "template-multiple", Code: "var a = `x`;\nvar b = `y`;", Fix: true},

		// Realistic nested scopes.
		{ID: "nested-function", Code: "function f() {\n  return 'x';\n}\n", Fix: true},
		{ID: "class-body", Code: "class C {\n  m() { return \"x\"; }\n}\n", Options: []any{"single"}, Fix: true},
	})
}

// TestNonASCIIParity covers the byte/code-unit boundary: reported columns and
// fix ranges are compared in ESLint's UTF-16 code-unit space (see units.go).
func TestNonASCIIParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		{ID: "cjk-single", Code: "var a = '\u4f60\u597d';", Fix: true},
		{ID: "cjk-accented-single-option", Code: "var s = \"caf\u00e9\";", Options: []any{"single"}, Fix: true},
		{ID: "cjk-emoji-single", Code: "var a = '\U0001f600';", Fix: true},
		{ID: "cjk-emoji-template", Code: "var a = `\U0001f600`;", Fix: true},
		{ID: "cjk-property-key", Code: "var o = { \"\u4f60\u597d\": 1 };", Options: []any{"single"}, Fix: true},
		{ID: "cjk-clean", Code: "var s = \"\u4f60\u597d\";"},
		{ID: "cjk-before-quote", Code: "var s = \"\u4f60\u597d\"; var b = 'x';", Fix: true},
		{ID: "cjk-backtick", Code: "var s = \"caf\u00e9\";", Options: []any{"backtick"}, Fix: true},
	})
}
