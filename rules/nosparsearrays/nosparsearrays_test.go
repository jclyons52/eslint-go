package nosparsearrays

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func severity(level any, opts ...any) map[string]any {
	value := []any{level}
	value = append(value, opts...)
	return map[string]any{"rules": map[string]any{Name: value}}
}

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Default behaviour: no holes.
		{ID: "empty", Code: "var a = [];"},
		{ID: "filled", Code: "var a = [1, 2, 3];"},
		{ID: "trailing-comma", Code: "var a = [1, 2,];"},
		{ID: "new-array", Code: "var a = new Array(3);"},
		{ID: "undefined-elements", Code: "var a = [undefined, undefined];"},
		{ID: "array-pattern", Code: "var [, a] = [1, 2];"},
		{ID: "nested-clean", Code: "var a = [[1, 2], [3, 4]];"},

		// Holes.
		{ID: "middle-hole", Code: "var a = [1,,2];"},
		{ID: "leading-hole", Code: "var a = [,1];"},
		{ID: "trailing-hole", Code: "var a = [1,,];"},
		{ID: "only-hole", Code: "var a = [,];"},
		{ID: "double-hole", Code: "var a = [,,];"},
		{ID: "triple-hole", Code: "var a = [1,,,2];"},
		{ID: "nested-hole", Code: "var a = [[1,,2]];"},
		{ID: "comment-before-hole", Code: "var a = [/* x */, 1];"},
		{ID: "spread-and-hole", Code: "var a = [...x, , 1];"},
		{ID: "call-argument", Code: "f([1,,2]);"},
		{ID: "multiline", Code: "var a = [\n    1,\n    ,\n    2\n];"},
		{ID: "template-elements", Code: "var a = [`a`,, `b`];"},
		{ID: "non-ascii", Code: "var a = [\"café\",, \"日本語\"];"},

		// Severities.
		{ID: "warn-severity", Code: "var a = [1,,2];", Config: severity("warn")},
		{ID: "off-severity", Code: "var a = [1,,2];", Config: severity("off")},
	})
}
