package nodebugger

import (
	"testing"

	eslint "github.com/jclyons52/eslint-go"
	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		{ID: "basic", Code: "debugger;"},
		{ID: "in-function", Code: "function f() { debugger; }"},
		{ID: "in-block", Code: "if (a) { debugger; }"},
		{ID: "in-loop", Code: "for (;;) { debugger; }"},
		{ID: "multiple", Code: "debugger;\ndebugger;\ndebugger;"},
		{ID: "nested-functions", Code: "function a() { function b() { debugger; } }"},
		{ID: "arrow", Code: "const f = () => { debugger; };"},
		{ID: "class-method", Code: "class C { m() { debugger; } }"},
		{ID: "clean", Code: "const x = 1;"},
		{ID: "warn-severity", Code: "debugger;", Config: map[string]any{
			"rules": map[string]any{Name: "warn"},
		}},
		{ID: "off-severity", Code: "debugger;", Config: map[string]any{
			"rules": map[string]any{Name: "off"},
		}},
		{ID: "numeric-severity", Code: "debugger;", Config: map[string]any{
			"rules": map[string]any{Name: 2},
		}},
		{ID: "array-severity", Code: "debugger;", Config: map[string]any{
			"rules": map[string]any{Name: []any{"error"}},
		}},
		{ID: "after-comment", Code: "// note\ndebugger; // trailing\n"},
		{ID: "fatal-parse-error", Code: "debugger;\nvar a = ;"},
		{ID: "label", Code: "outer: { debugger; break outer; }"},
	})
	_ = eslint.Node(nil)
}
