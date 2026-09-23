package nomultipleemptylines

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Defaults (max 2).
		{ID: "default-ok", Code: "var a = 1;\n\nvar b = 2;\n"},
		{ID: "default-two-blank-ok", Code: "var a = 1;\n\n\nvar b = 2;\n"},
		{ID: "default-three-blank", Code: "var a = 1;\n\n\n\nvar b = 2;\n", Fix: true},
		{ID: "default-four-blank", Code: "var a = 1;\n\n\n\n\nvar b = 2;\n", Fix: true},
		{ID: "clean", Code: "var a = 1;\nvar b = 2;\n"},
		{ID: "empty-file", Code: ""},
		{ID: "only-newline", Code: "\n"},
		{ID: "only-two-newlines", Code: "\n\n"},
		{ID: "only-three-newlines", Code: "\n\n\n"},

		// max.
		{ID: "max-1", Code: "var a = 1;\n\n\nvar b = 2;\n", Options: []any{map[string]any{"max": 1}}, Fix: true},
		{ID: "max-0", Code: "var a = 1;\n\nvar b = 2;\n", Options: []any{map[string]any{"max": 0}}, Fix: true},
		{ID: "max-0-ok", Code: "var a = 1;\nvar b = 2;\n", Options: []any{map[string]any{"max": 0}}},
		{ID: "max-3-ok", Code: "var a = 1;\n\n\n\nvar b = 2;\n", Options: []any{map[string]any{"max": 3}}},

		// maxEOF.
		{ID: "maxeof-0", Code: "var a = 1;\n\n\n", Options: []any{map[string]any{"max": 2, "maxEOF": 0}}, Fix: true},
		{ID: "maxeof-1", Code: "var a = 1;\n\n\n\n", Options: []any{map[string]any{"max": 2, "maxEOF": 1}}, Fix: true},
		{ID: "maxeof-2-ok", Code: "var a = 1;\n\n\n", Options: []any{map[string]any{"max": 2, "maxEOF": 2}}},
		{ID: "maxeof-trailing-blanks", Code: "var a = 1;\n\n\n\n", Fix: true},

		// maxBOF.
		{ID: "maxbof-0", Code: "\n\nvar a = 1;\n", Options: []any{map[string]any{"max": 2, "maxBOF": 0}}, Fix: true},
		{ID: "maxbof-0-only-blanks", Code: "\n\n\n\n", Options: []any{map[string]any{"max": 2, "maxBOF": 0}}, Fix: true},
		{ID: "maxbof-1", Code: "\n\n\nvar a = 1;\n", Options: []any{map[string]any{"max": 2, "maxBOF": 1}}, Fix: true},
		{ID: "maxbof-1-ok", Code: "\nvar a = 1;\n", Options: []any{map[string]any{"max": 2, "maxBOF": 1}}},
		{ID: "maxbof-2-ok", Code: "\n\nvar a = 1;\n"},

		// Blank lines that are not literally empty.
		{ID: "whitespace-lines", Code: "var a = 1;\n   \n   \n   \nvar b = 2;\n", Fix: true},
		{ID: "tab-lines", Code: "var a = 1;\n\t\n\t\n\t\nvar b = 2;\n", Fix: true},
		{ID: "nbsp-lines", Code: "var a = 1;\n\u00a0\n\u00a0\n\u00a0\nvar b = 2;\n", Fix: true},
		{ID: "ideographic-space-lines", Code: "var a = 1;\n\u3000\n\u3000\n\u3000\nvar b = 2;\n", Fix: true},

		// Template literals: blank lines inside them are semantic.
		{ID: "template-blanks-ok", Code: "var t = `a\n\n\nb`;\n"},
		{ID: "template-two-blanks-ok", Code: "var t = `a\n\nb`;\n"},
		{ID: "template-interpolation-blanks", Code: "var t = `a${x}\n\n\nb${y}`;\n"},
		{ID: "template-then-blanks", Code: "var t = `a\n\n\nb`;\n\n\n\nvar c = 3;\n", Fix: true},
		{ID: "template-trailing-blank-inside", Code: "var t = `a\n\n\n`;\n"},

		// Comments / realistic files.
		{ID: "comments-separate", Code: "var a = 1;\n\n// c\n\nvar b = 2;\n"},
		{ID: "comment-blank-run", Code: "var a = 1;\n\n\n\n// c\nvar b = 2;\n", Fix: true},
		{ID: "nested-function", Code: "function f() {\n  var a = 1;\n\n\n\n  return a;\n}\n", Fix: true},

		// CRLF.
		{ID: "crlf", Code: "var a = 1;\r\n\r\n\r\n\r\nvar b = 2;\r\n", Fix: true},
		{ID: "crlf-ok", Code: "var a = 1;\r\n\r\nvar b = 2;\r\n"},

		// Severity.
		{ID: "warn", Code: "var a = 1;\n\n\n\nvar b = 2;\n", Config: map[string]any{
			"rules": map[string]any{Name: []any{1, map[string]any{"max": 2}}},
		}, Fix: true},
	})
}

// TestNonASCIIParity covers the byte/code-unit boundary for the removal ranges
// and the reported columns (see units.go).
func TestNonASCIIParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		{ID: "cjk-blanks", Code: "var s = \"\u4f60\u597d\";\n\n\n\nvar b = 2;\n", Fix: true},
		{ID: "cjk-bof", Code: "\n\n\nvar s = \"\u4f60\u597d\";\n", Options: []any{map[string]any{"max": 2, "maxBOF": 0}}, Fix: true},
		{ID: "cjk-eof", Code: "var s = \"\u4f60\u597d\";\n\n\n\n", Options: []any{map[string]any{"max": 2, "maxEOF": 0}}, Fix: true},
		{ID: "emoji-blanks", Code: "var s = \"\U0001f600\";\n\n\n\nvar b = 2;\n", Fix: true},
		{ID: "accented-blanks", Code: "var caf\u00e9 = 1;\n\n\n\nvar b = 2;\n", Fix: true},
		{ID: "cjk-template-blanks", Code: "var t = `\u4f60\u597d\n\n\n\u4e16\u754c`;\n"},
		{ID: "cjk-clean", Code: "var s = \"\u4f60\u597d\";\nvar b = 2;\n"},
	})
}
