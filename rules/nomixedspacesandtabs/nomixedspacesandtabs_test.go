package nomixedspacesandtabs

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Clean indentation, both styles.
		{ID: "clean-spaces", Code: "if (a) {\n    b();\n}\n"},
		{ID: "clean-tabs", Code: "if (a) {\n\tb();\n}\n"},
		{ID: "clean-flat", Code: "var a = 1;\nvar b = 2;\n"},
		{ID: "clean-blank-lines", Code: "var a = 1;\n\nvar b = 2;\n"},

		// The default pattern: a maximal run of one whitespace character
		// followed by the other one.
		{ID: "space-then-tab", Code: "if (a) {\n \tb();\n}\n"},
		{ID: "tab-then-space", Code: "if (a) {\n\t b();\n}\n"},
		{ID: "top-level-space-then-tab", Code: "var a = 1;\n \tvar b = 2;\n"},
		{ID: "top-level-tab-then-space", Code: "var a = 1;\n\t var b = 2;\n"},
		{ID: "deep-tabs-then-space", Code: "if (a) {\n\t\t  b();\n}\n"},
		{ID: "deep-spaces-then-tab", Code: "if (a) {\n    \tb();\n}\n"},
		{ID: "mixed-only-line", Code: " \t\n"},
		{ID: "mixed-blank-line", Code: "var a = 1;\n \t\nvar b = 2;\n"},
		{ID: "leading-blank-then-mixed", Code: "\n \tvar a = 1;\n"},
		{ID: "crlf", Code: "var a = 1;\r\n 	var b = 2;\r\n"},
		{ID: "line-separator-2028", Code: "var a = 1;\u2028 	var b = 2;\u2028"},
		{ID: "paragraph-separator-2029", Code: "var a = 1;\u2029	 var b = 2;\u2029"},
		{ID: "several-mixed-lines", Code: "if (a) {\n 	b();\n	 c();\n}\n"},

		// smart-tabs: tabs for indentation, spaces for alignment.
		{ID: "smart-tabs-tab-space-code", Code: "if (a) {\n\t b();\n}\n",
			Options: []any{"smart-tabs"}},
		{ID: "smart-tabs-tab-space-tab", Code: "if (a) {\n\t \tb();\n}\n",
			Options: []any{"smart-tabs"}},
		{ID: "smart-tabs-space-tab", Code: "if (a) {\n  \tb();\n}\n",
			Options: []any{"smart-tabs"}},
		{ID: "smart-tabs-space-tab-space", Code: "if (a) {\n \t b();\n}\n",
			Options: []any{"smart-tabs"}},
		{ID: "smart-tabs-deep", Code: "if (a) {\n\t  \tb();\n}\n",
			Options: []any{"smart-tabs"}},
		{ID: "smart-tabs-clean-tabs", Code: "if (a) {\n\t\tb();\n}\n",
			Options: []any{"smart-tabs"}},
		{ID: "smart-tabs-clean-spaces", Code: "if (a) {\n  b();\n}\n",
			Options: []any{"smart-tabs"}},
		{ID: "smart-tabs-default-would-fire", Code: "if (a) {\n\t b();\n}\n",
			Options: []any{"smart-tabs"}, Config: map[string]any{"rules": map[string]any{Name: []any{2, "smart-tabs"}}}},

		// The legacy `true` spelling is smart-tabs; `false` is the default.
		{ID: "option-true", Code: "if (a) {\n\t \tb();\n}\n", Options: []any{true}},
		{ID: "option-true-clean", Code: "if (a) {\n\t b();\n}\n", Options: []any{true}},
		{ID: "option-false", Code: "if (a) {\n\t \tb();\n}\n", Options: []any{false}},
		{ID: "option-false-tab-space", Code: "if (a) {\n\t b();\n}\n", Options: []any{false}},

		// Comment lines are ignored (all but the comment's first line).
		{ID: "block-comment-lines", Code: "/*\n \tcomment\n*/\nvar a = 1;\n"},
		{ID: "block-comment-mid-line", Code: "var a = 1; /*\n \tc\n*/\n"},
		{ID: "line-comment-not-ignored", Code: "var a = 1; // c\n \tvar b = 2;\n"},
		{ID: "block-comment-first-line", Code: "/* \tc */\nvar a = 1;\n"},

		// Literal and template contents are ignored.
		{ID: "multiline-string", Code: "var s = \"a\n \tb\";\n"},
		{ID: "multiline-template", Code: "var t = `a\n \tb`;\n"},
		{ID: "multiline-template-expression", Code: "var t = `a\n \t${x}\nb`;\n"},
		{ID: "multiline-string-with-code", Code: "var s = \"a\n \tb\";\n \tvar c = 1;\n"},

		// Severity.
		{ID: "warn", Code: "var a = 1;\n \tvar b = 2;\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "off", Code: "var a = 1;\n \tvar b = 2;\n", Config: map[string]any{"rules": map[string]any{Name: "off"}}},

		// Non-ASCII: the reported line/column pair is derived through
		// getIndexFromLoc, so byte offsets and code units must agree here.
		{ID: "nonascii-before", Code: "var caf\u00e9 = 1;\n \tvar b = 2;\n"},
		{ID: "nonascii-same-line", Code: "var s = \"\u4f60\u597d\";\n \tvar b = 2;\n"},
		{ID: "nonascii-emoji-before", Code: "var s = \"\U0001f600\";\n\t var b = 2;\n"},
		{ID: "nonascii-in-comment", Code: "/* \u4f60\u597d\n \t*/ \nvar a = 1;\n \tvar b = 2;\n"},
		{ID: "nonascii-clean", Code: "var caf\u00e9 = 1;\nvar b = 2;\n"},
	})
}
