package notrailingspaces

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		{ID: "single-trailing", Code: "var a = 1; \nvar b = 2;\n", Fix: true},
		{ID: "tab-trailing", Code: "var a = 1;\t\n"},
		{ID: "multiple-trailing", Code: "var a = 1;   \nvar b = 2;  \n"},
		{ID: "blank-line", Code: "var a = 1;\n   \nvar b = 2;\n", Options: []any{map[string]any{"skipBlankLines": false}}, Fix: true},
		{ID: "skip-blank-lines", Code: "var a = 1;\n   \nvar b = 2;\n", Options: []any{map[string]any{"skipBlankLines": true}}, Fix: true},
		{ID: "clean", Code: "var a = 1;\nvar b = 2;\n"},
		{ID: "last-line-no-newline", Code: "var a = 1;  "},
		{ID: "trailing-in-template", Code: "const t = `line1\nline2  \nline3`;\n"},
		{ID: "comment-trailing", Code: "// comment   \nvar a = 1;\n"},
		{ID: "ignore-comments", Code: "// comment   \nvar a = 1;   \n", Options: []any{map[string]any{"ignoreComments": true}}, Fix: true},
		{ID: "block-comment", Code: "/* multi\n   line   */\nvar a = 1;\n"},
		{ID: "crlf", Code: "var a = 1;  \r\nvar b = 2;\r\n", Fix: true},
		{ID: "nbsp", Code: "var a = 1;\u00a0\n"},
		{ID: "indented-blank-skip", Code: "  \nvar a = 1;\n", Options: []any{map[string]any{"skipBlankLines": true}}},
		{ID: "nested-function", Code: "function f() {\n  return 1;  \n}\n", Fix: true},
	})
}

// TestNonASCIIParity covers the byte/code-unit boundary: reported columns and
// fix ranges in ESLint are UTF-16 code-unit indices, while this port works in
// byte offsets internally (see units.go). Non-ASCII text is where the two
// spaces disagree, so every rule corpus should carry a few such cases.
func TestNonASCIIParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		{ID: "nbsp-trailing", Code: "var a = 1;\u00a0\n", Fix: true},
		{ID: "cjk-trailing", Code: "var s = \"\u4f60\u597d\";  \n", Fix: true},
		{ID: "emoji-trailing", Code: "var s = \"\U0001f600\";   \nvar b = 2;\n", Fix: true},
		{ID: "emoji-then-trailing", Code: "var s = \"\U0001f600\";\nvar b = 2; \n", Fix: true},
		{ID: "ideographic-space", Code: "var a = 1;\u3000\n", Fix: true},
		{ID: "accented", Code: "var caf\u00e9 = 1; \n", Fix: true},
		{ID: "nonascii-clean", Code: "var s = \"\u4f60\u597d\";\n"},
	})
}
