package nomultispaces

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Doc examples.
		{ID: "variable-declarator", Code: "var a  = 1;\n", Fix: true},
		{ID: "before-value", Code: "var a =  1;\n", Fix: true},
		{ID: "binary-expression", Code: "a  +  b;\n", Fix: true},
		{ID: "logical-expression", Code: "if (a  &&  b) {}\n", Fix: true},
		{ID: "call-args", Code: "f(1,  2);\n", Fix: true},
		{ID: "array-elements", Code: "[1,  2];\n", Fix: true},
		{ID: "function-params", Code: "function f(a,  b) {}\n", Fix: true},
		{ID: "multi-declarator", Code: "let a = 1,  b = 2;\n", Fix: true},

		// Default exception: a Property node's own whitespace is allowed.
		{ID: "property-default", Code: "var a = { b:  1 };\n"},
		{ID: "property-separator", Code: "var a = { b: 1,  c: 2 };\n", Fix: true},
		{ID: "property-disabled", Code: "var a = { b:  1 };\n",
			Options: []any{map[string]any{"exceptions": map[string]any{"Property": false}}}, Fix: true},

		// Other exception types.
		{ID: "exception-var-declarator", Code: "var a  =  1;\n",
			Options: []any{map[string]any{"exceptions": map[string]any{"VariableDeclarator": true}}}},
		{ID: "exception-call-expression", Code: "foo(bar,  baz);\n",
			Options: []any{map[string]any{"exceptions": map[string]any{"CallExpression": true}}}},
		{ID: "exception-call-off", Code: "foo(bar,  baz);\n",
			Options: []any{map[string]any{"exceptions": map[string]any{"Property": false}}}, Fix: true},

		// Comments as the right token.
		{ID: "line-comment", Code: "var a = 1;  // comment\n", Fix: true},
		{ID: "block-comment", Code: "var a = 1;  /* comment */\n", Fix: true},
		{ID: "long-line-comment", Code: "var a = 1;  // this is a very long comment\n", Fix: true},
		{ID: "exact-12-chars", Code: "var a = 1;  //123456789012\n", Fix: true},
		{ID: "13-chars", Code: "var a = 1;  //1234567890123\n", Fix: true},
		{ID: "multiline-block-comment", Code: "var a = 1;  /* line1\nline2 */\n", Fix: true},
		{ID: "comment-inside-expression", Code: "f(a,  /* c */ b);\n", Fix: true},

		// ignoreEOLComments.
		{ID: "eol-comment-ignored", Code: "var a = 1;  // comment\n",
			Options: []any{map[string]any{"ignoreEOLComments": true}}},
		{ID: "eol-block-ignored", Code: "var a = 1;  /* comment */\n",
			Options: []any{map[string]any{"ignoreEOLComments": true}}},
		{ID: "eol-not-last", Code: "var a = 1;  /* c */ var b = 2;\n",
			Options: []any{map[string]any{"ignoreEOLComments": true}}, Fix: true},
		{ID: "eol-last-token", Code: "var a = 1;  /* c */\n",
			Options: []any{map[string]any{"ignoreEOLComments": true}}},
		{ID: "eol-with-multiline-comment", Code: "var a = 1;  /* c\n   d */\n",
			Options: []any{map[string]any{"ignoreEOLComments": true}}},
		{ID: "eol-off-by-default", Code: "var a = 1;  // comment\n", Fix: true},

		// Whitespace shapes that are not "multiple spaces".
		{ID: "tab-then-space", Code: "var a =\t 1;\n"},
		{ID: "space-space-tab", Code: "var a =  \t1;\n", Fix: true},
		{ID: "leading-indent", Code: "  var a = 1;\n"},
		{ID: "trailing-eol", Code: "var a = 1;  \nvar b = 2;\n"},
		{ID: "across-lines", Code: "var a = 1;  \nvar b = 2;\n"},
		{ID: "newline-between", Code: "var a =\n  1;\n"},
		{ID: "single-space", Code: "var a = 1;\n"},
		{ID: "clean", Code: "var a = 1;\nvar b = a + 1;\nf(a, b);\nvar o = { k: 1 };\n"},

		// Severity.
		{ID: "warn", Code: "var a  = 1;\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "var a  = 1;\n", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Non-ASCII: the fix range and reported columns are UTF-16 code units.
		{ID: "nonascii-declarator", Code: "var caf\u00e9  = 1;\n", Fix: true},
		{ID: "nonascii-two-statements", Code: "var s = \"\u4f60\u597d\";  var t = 1;\n", Fix: true},
		{ID: "nonascii-emoji", Code: "var s = \"\U0001f600\";  var t = 1;\n", Fix: true},
		{ID: "nonascii-comment", Code: "var s = \"\u4f60\u597d\";  // \u4f60\u597d\u4f60\u597d\u4f60\u597d\u4f60\u597d\n", Fix: true},
		{ID: "nonascii-clean", Code: "var s = \"\u4f60\u597d\";\n"},
	})
}
