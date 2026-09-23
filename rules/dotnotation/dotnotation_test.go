package dotnotation

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- default (allowKeywords: true) ----
		{ID: "basic", Code: "var x = foo[\"bar\"];\n", Fix: true},
		{ID: "member-after", Code: "var x = foo[\"bar\"].baz;\n", Fix: true},
		{ID: "call-after", Code: "foo[\"bar\"]();\n", Fix: true},
		{ID: "plus-after", Code: "var x = foo[\"bar\"]+1;\n", Fix: true},
		{ID: "template-after", Code: "foo[\"bar\"]`x`;\n", Fix: true},
		{ID: "in-after", Code: "var x = foo[\"bar\"]in obj;\n", Fix: true},
		{ID: "instanceof-after", Code: "var x = foo[\"bar\"]instanceof Bar;\n", Fix: true},
		{ID: "nested-computed", Code: "foo[\"bar\"][\"baz\"];\n", Fix: true},
		{ID: "clean-dot", Code: "foo.bar;\n"},
		{ID: "clean-number-index", Code: "foo[0];\n"},
		{ID: "keyword-computed-default", Code: "foo[\"catch\"];\n", Fix: true},
		{ID: "keyword-dot-default", Code: "foo.catch;\n"},
		{ID: "space-in-key", Code: "foo[\"bar baz\"];\n"},
		{ID: "dot-in-key", Code: "foo[\"a.b\"];\n"},
		{ID: "digit-key", Code: "foo[\"1\"];\n"},
		{ID: "empty-key", Code: "foo[\"\"];\n"},
		{ID: "dollar-underscore", Code: "foo[\"$\"]; bar[\"_\"];\n", Fix: true},
		{ID: "escaped-key", Code: "foo[\"\\u0062ar\"];\n", Fix: true},
		{ID: "true-literal", Code: "foo[true];\n", Fix: true},
		{ID: "null-literal", Code: "foo[null];\n", Fix: true},
		{ID: "string-number-literal", Code: "foo[1];\n"},
		{ID: "regex-literal", Code: "foo[/x/];\n"},
		{ID: "template-literal", Code: "foo[`bar`];\n", Fix: true},
		{ID: "template-literal-invalid", Code: "foo[`bar baz`];\n"},
		{ID: "template-then-in", Code: "var x = foo[`bar`]in obj;\n", Fix: true},
		{ID: "optional-chain", Code: "foo?.[\"bar\"];\n", Fix: true},
		{ID: "optional-chain-member-after", Code: "foo?.[\"bar\"].baz;\n", Fix: true},
		{ID: "optional-chain-in-after", Code: "var x = foo?.[\"bar\"]in obj;\n", Fix: true},
		{ID: "numeric-object", Code: "(1)[\"toString\"];\n", Fix: true},
		{ID: "comment-in-brackets", Code: "foo[/* c */\"bar\"];\n"},
		{ID: "comment-in-brackets-line", Code: "foo[// c\n\"bar\"];\n"},
		{ID: "proto-key", Code: "obj[\"__proto__\"];\n", Fix: true},
		{ID: "warn-severity", Code: "var x = foo[\"bar\"];\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}, Fix: true},
		{ID: "computed-expr", Code: "foo[bar];\n"},
		{ID: "assignment-target", Code: "foo[\"bar\"] = 1;\n", Fix: true},
		{ID: "multiline-object", Code: "var x = ({\n  a: 1\n})[\"a\"];\n", Fix: true},

		// ---- allowKeywords: false ----
		{ID: "no-keywords-computed", Code: "foo[\"catch\"];\n", Options: []any{map[string]any{"allowKeywords": false}}},
		{ID: "no-keywords-dot", Code: "foo.catch;\n", Options: []any{map[string]any{"allowKeywords": false}}, Fix: true},
		{ID: "no-keywords-in", Code: "foo.in;\n", Options: []any{map[string]any{"allowKeywords": false}}, Fix: true},
		{ID: "no-keywords-identifier", Code: "foo.bar;\n", Options: []any{map[string]any{"allowKeywords": false}}},
		{ID: "no-keywords-nonkeyword-string", Code: "foo[\"bar\"];\n", Options: []any{map[string]any{"allowKeywords": false}}, Fix: true},
		{ID: "no-keywords-true-string", Code: "foo[\"true\"];\n", Options: []any{map[string]any{"allowKeywords": false}}},
		{ID: "no-keywords-comment-between", Code: "foo./* c */catch;\n", Options: []any{map[string]any{"allowKeywords": false}}},
		{ID: "no-keywords-optional", Code: "foo?.catch;\n", Options: []any{map[string]any{"allowKeywords": false}}, Fix: true},
		{ID: "no-keywords-member-chain", Code: "foo.catch.bar;\n", Options: []any{map[string]any{"allowKeywords": false}}, Fix: true},
		{ID: "no-keywords-delete", Code: "delete foo.delete;\n", Options: []any{map[string]any{"allowKeywords": false}}, Fix: true},

		// ---- allowPattern ----
		{ID: "pattern-underscore", Code: "foo[\"_bar\"];\n", Options: []any{map[string]any{"allowPattern": "^_"}}},
		{ID: "pattern-underscore-other", Code: "foo[\"bar\"];\n", Options: []any{map[string]any{"allowPattern": "^_"}}, Fix: true},
		{ID: "pattern-unanchored", Code: "foo[\"bar_baz\"];\n", Options: []any{map[string]any{"allowPattern": "_"}}},
		{ID: "pattern-lowercase-only", Code: "foo[\"Bar\"];\n", Options: []any{map[string]any{"allowPattern": "^[a-z]+$"}}, Fix: true},
		{ID: "pattern-with-keywords", Code: "foo[\"_catch\"];\n", Options: []any{map[string]any{"allowKeywords": false, "allowPattern": "^_"}}},
		{ID: "pattern-template", Code: "foo[`_bar`];\n", Options: []any{map[string]any{"allowPattern": "^_"}}},

		// ---- non-ASCII (fix ranges are UTF-16 code units) ----
		{ID: "nonascii-before", Code: "var s = \"h\u00e9llo\"; foo[\"bar\"];\n", Fix: true},
		{ID: "nonascii-emoji", Code: "var s = \"\U0001f600\"; foo[\"bar\"].baz;\n", Fix: true},
		{ID: "nonascii-keyword", Code: "var s = \"\u4f60\u597d\"; foo.catch;\n", Options: []any{map[string]any{"allowKeywords": false}}, Fix: true},
		{ID: "nonascii-in-brackets", Code: "foo[\"bar\"]; var t = \"\u4f60\u597d\";\n", Fix: true},
	})
}
