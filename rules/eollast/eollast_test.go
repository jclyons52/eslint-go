package eollast

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Default mode ("always").
		{ID: "always-ok", Code: "var a = 1;\n"},
		{ID: "always-missing", Code: "var a = 1;", Fix: true},
		{ID: "always-missing-trailing-space", Code: "var a = 1;  ", Fix: true},
		{ID: "always-missing-comment", Code: "var a = 1; // c", Fix: true},
		{ID: "empty-file", Code: ""},
		{ID: "empty-file-never", Code: "", Options: []any{"never"}},
		{ID: "only-newline", Code: "\n"},
		// Verify-only: a fix pass over a source shorter than the UTF-8 BOM
		// (3 bytes) panics in the core's applyFixes (fixer.go:87,
		// `len(sourceText) > 0 && sourceText[:len(bom)] == bom`), so the fix
		// half of this case cannot run. Reported as a core gap.
		{ID: "only-newline-never", Code: "\n", Options: []any{"never"}},
		{ID: "only-newlines-never-fix", Code: "\n\n\n", Options: []any{"never"}, Fix: true},
		{ID: "whitespace-only", Code: "   ", Fix: true},

		// "never".
		{ID: "never-ok", Code: "var a = 1;", Options: []any{"never"}},
		{ID: "never-trailing", Code: "var a = 1;\n", Options: []any{"never"}, Fix: true},
		{ID: "never-two-trailing", Code: "var a = 1;\n\n", Options: []any{"never"}, Fix: true},
		{ID: "never-many-trailing", Code: "var a = 1;\n\n\n\n", Options: []any{"never"}, Fix: true},
		{ID: "never-crlf", Code: "var a = 1;\r\n", Options: []any{"never"}, Fix: true},
		{ID: "never-crlf-mixed", Code: "var a = 1;\r\n\r\n\n", Options: []any{"never"}, Fix: true},
		{ID: "never-lone-cr", Code: "var a = 1;\r", Options: []any{"never"}},

		// "unix" / "windows".
		{ID: "unix-missing", Code: "var a = 1;", Options: []any{"unix"}, Fix: true},
		{ID: "unix-ok", Code: "var a = 1;\n", Options: []any{"unix"}},
		{ID: "unix-never-ish", Code: "var a = 1;\r\n", Options: []any{"unix"}},
		{ID: "windows-missing", Code: "var a = 1;", Options: []any{"windows"}, Fix: true},
		{ID: "windows-ok-lf", Code: "var a = 1;\n", Options: []any{"windows"}},
		{ID: "windows-ok-crlf", Code: "var a = 1;\r\n", Options: []any{"windows"}},

		// Lone CR / unicode line separators.
		{ID: "lone-cr-missing", Code: "var a = 1;\r", Fix: true},
		{ID: "u2028-missing", Code: "var a = 1;\u2028", Fix: true},

		// Realistic multi-line files.
		{ID: "multiline-ok", Code: "function f() {\n  return 1;\n}\n"},
		{ID: "multiline-missing", Code: "function f() {\n  return 1;\n}", Fix: true},
		{ID: "template-multiline-missing", Code: "var t = `a\nb`;", Fix: true},
		{ID: "comment-last", Code: "var a = 1;\n// end", Fix: true},

		// Severity.
		{ID: "warn", Code: "var a = 1;", Config: map[string]any{
			"rules": map[string]any{Name: []any{1}},
		}, Fix: true},
	})
}

// TestNonASCIIParity covers the byte/code-unit boundary: ESLint reports the
// "missing" column as the last line's length in UTF-16 code units, while this
// port computes it in bytes (see units.go).
func TestNonASCIIParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		{ID: "cjk-missing", Code: "var s = \"\u4f60\u597d\";", Fix: true},
		{ID: "cjk-ok", Code: "var s = \"\u4f60\u597d\";\n"},
		{ID: "cjk-never", Code: "var s = \"\u4f60\u597d\";\n", Options: []any{"never"}, Fix: true},
		{ID: "emoji-missing", Code: "var s = \"\U0001f600\";", Fix: true},
		{ID: "emoji-never", Code: "var s = \"\U0001f600\";\r\n", Options: []any{"never"}, Fix: true},
		{ID: "accented-missing", Code: "var caf\u00e9 = 1;", Fix: true},
		{ID: "accented-never", Code: "var caf\u00e9 = 1;\n", Options: []any{"never"}, Fix: true},
		{ID: "nbsp-missing", Code: "var a = 1;\u00a0", Fix: true},
		{ID: "cjk-multiline-missing", Code: "var s = \"\u4f60\u597d\";\nvar b = 2;", Fix: true},
	})
}
