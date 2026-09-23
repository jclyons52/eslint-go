package commadangle

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// Default: never.
		{ID: "default-object-trailing", Code: "var o = { a: 1, };", Fix: true},
		{ID: "default-array-trailing", Code: "var a = [1, 2, ];", Fix: true},
		{ID: "default-array-trailing-nospace", Code: "var a = [1, 2,];", Fix: true},
		{ID: "default-clean", Code: "var o = { a: 1 };"},
		{ID: "default-multiline-trailing", Code: "var o = {\n  a: 1,\n};", Fix: true},
		{ID: "default-function-trailing", Code: "function f(a,) {}", Fix: true},
		{ID: "default-call-trailing", Code: "f(a,);", Fix: true},
		{ID: "default-new-trailing", Code: "new F(a,);", Fix: true},
		{ID: "default-nested", Code: "var o = { a: [1, 2,], b: 3, };", Fix: true},

		// always.
		{ID: "always-object-missing", Code: "var o = { a: 1 };", Options: []any{"always"}, Fix: true},
		{ID: "always-array-missing", Code: "var a = [1, 2];", Options: []any{"always"}, Fix: true},
		{ID: "always-object-present-ok", Code: "var o = { a: 1, };", Options: []any{"always"}},
		{ID: "always-function", Code: "function f(a) {}", Options: []any{"always"}, Fix: true},
		{ID: "always-arrow", Code: "var g = (a) => a;", Options: []any{"always"}, Fix: true},
		{ID: "always-call", Code: "f(a);", Options: []any{"always"}, Fix: true},
		{ID: "always-new", Code: "new F(a);", Options: []any{"always"}, Fix: true},
		{ID: "always-multiline-object", Code: "var o = {\n  a: 1\n};", Options: []any{"always"}, Fix: true},
		{ID: "always-empty-array", Code: "var a = [];", Options: []any{"always"}},
		{ID: "always-empty-object", Code: "var o = {};", Options: []any{"always"}},
		{ID: "always-empty-call", Code: "f();", Options: []any{"always"}},
		{ID: "always-import", Code: "import { a, b } from 'm';", Options: []any{"always"}, Fix: true},
		{ID: "always-import-default-only", Code: "import a from 'm';", Options: []any{"always"}},
		{ID: "always-import-mixed", Code: "import a, { b } from 'm';", Options: []any{"always"}, Fix: true},
		{ID: "always-import-namespace", Code: "import * as ns from 'm';", Options: []any{"always"}},
		{ID: "always-export", Code: "export { a, b };", Options: []any{"always"}, Fix: true},
		{ID: "always-export-one", Code: "export { a };", Options: []any{"always"}, Fix: true},
		{ID: "always-export-default", Code: "export default 1;", Options: []any{"always"}},
		{ID: "always-object-pattern", Code: "var { a } = b;", Options: []any{"always"}, Fix: true},
		{ID: "always-array-pattern", Code: "var [a, b] = c;", Options: []any{"always"}, Fix: true},
		{ID: "always-rest-param", Code: "function f(...a) {}", Options: []any{"always"}},
		{ID: "always-array-rest", Code: "var [a, ...b] = c;", Options: []any{"always"}},
		{ID: "always-object-rest", Code: "var { a, ...b } = c;", Options: []any{"always"}},
		{ID: "always-nested", Code: "var o = { a: [1, 2], b: 3 };", Options: []any{"always"}, Fix: true},

		// always-multiline.
		{ID: "am-multiline-missing", Code: "var o = {\n  a: 1\n};", Options: []any{"always-multiline"}, Fix: true},
		{ID: "am-multiline-present-ok", Code: "var o = {\n  a: 1,\n};", Options: []any{"always-multiline"}},
		{ID: "am-single-present", Code: "var o = { a: 1, };", Options: []any{"always-multiline"}, Fix: true},
		{ID: "am-single-missing-ok", Code: "var o = { a: 1 };", Options: []any{"always-multiline"}},
		{ID: "am-call-multiline", Code: "f(\n  a,\n  b\n);", Options: []any{"always-multiline"}, Fix: true},
		{ID: "am-call-single", Code: "f(a, b,);", Options: []any{"always-multiline"}, Fix: true},
		{ID: "am-array-multiline", Code: "var a = [\n  1,\n  2\n];", Options: []any{"always-multiline"}, Fix: true},
		{ID: "am-comment-last-line", Code: "var a = [\n  1,\n  2 // c\n];", Options: []any{"always-multiline"}, Fix: true},
		{ID: "am-nested-single", Code: "var a = [1, 2];", Options: []any{"always-multiline"}},
		{ID: "am-new-multiline", Code: "new F(\n  a\n);", Options: []any{"always-multiline"}, Fix: true},
		{ID: "am-close-same-line", Code: "var a = [\n  1, 2];", Options: []any{"always-multiline"}, Fix: true},
		{ID: "am-import-multiline", Code: "import {\n  a,\n  b\n} from 'm';", Options: []any{"always-multiline"}, Fix: true},

		// only-multiline.
		{ID: "om-multiline-trailing-ok", Code: "var o = {\n  a: 1,\n};", Options: []any{"only-multiline"}},
		{ID: "om-multiline-missing-ok", Code: "var o = {\n  a: 1\n};", Options: []any{"only-multiline"}},
		{ID: "om-single-trailing", Code: "var o = { a: 1, };", Options: []any{"only-multiline"}, Fix: true},
		{ID: "om-single-clean", Code: "var o = { a: 1 };", Options: []any{"only-multiline"}},
		{ID: "om-array-multiline-trailing", Code: "var a = [\n  1,\n];", Options: []any{"only-multiline"}},
		{ID: "om-call-single-trailing", Code: "f(a,);", Options: []any{"only-multiline"}, Fix: true},

		// Object form options.
		{ID: "obj-arrays-always", Code: "var a = [1];\nvar o = { b: 2, };",
			Options: []any{map[string]any{"arrays": "always"}}, Fix: true},
		{ID: "obj-functions-ignore", Code: "f(a);", Options: []any{map[string]any{"functions": "ignore"}}},
		{ID: "obj-functions-ignore-objects", Code: "var o = { a: 1, };\nf(a);",
			Options: []any{map[string]any{"functions": "ignore"}}, Fix: true},
		{ID: "obj-exports-always", Code: "export { a };", Options: []any{map[string]any{"exports": "always"}}, Fix: true},
		{ID: "obj-imports-never", Code: "import { a, } from 'm';", Options: []any{map[string]any{"imports": "never"}}, Fix: true},
		{ID: "obj-mixed", Code: "var a = [1];\nvar o = { b: 2, };",
			Options: []any{map[string]any{"arrays": "always", "objects": "never"}}, Fix: true},
		{ID: "obj-ignore-all", Code: "var a = [1,];\nvar o = { b: 2, };\nf(c,);",
			Options: []any{map[string]any{"arrays": "ignore", "objects": "ignore", "functions": "ignore"}}},
		{ID: "obj-empty", Code: "var o = { a: 1, };", Options: []any{map[string]any{}}, Fix: true},

		// Comments.
		{ID: "comment-after-element", Code: "var a = [1, 2 /* c */];", Options: []any{"always"}, Fix: true},
		{ID: "comment-before-close", Code: "var a = [\n  1,\n  2\n  // c\n];", Options: []any{"always"}, Fix: true},
		{ID: "trailing-comma-with-comment", Code: "var a = [1, 2, /* c */];", Fix: true},

		// Holes.
		{ID: "array-hole", Code: "var a = [1,,];"},
		{ID: "array-hole-always", Code: "var a = [1,,];", Options: []any{"always"}},
		{ID: "array-leading-hole", Code: "var a = [,1];", Options: []any{"always"}, Fix: true},

		// Severity.
		{ID: "warn", Code: "var o = { a: 1, };", Config: map[string]any{
			"rules": map[string]any{Name: []any{1}},
		}, Fix: true},
		{ID: "off", Code: "var o = { a: 1, };", Config: map[string]any{
			"rules": map[string]any{Name: []any{0}},
		}},
	})
}

// TestECMAVersionParity covers normalizeOptions' `ecmaVersion < 2017` branch:
// with a string option, `functions` becomes "ignore" before ES2017. The
// ecmaVersion has to travel in parserOptions (the harness only sends it to the
// oracle through the case's ECMAVersion field).
func TestECMAVersionParity(t *testing.T) {
	withEv := func(ev float64) map[string]any {
		return map[string]any{
			"parserOptions": map[string]any{"ecmaVersion": ev},
			"rules":         map[string]any{Name: []any{2, "always"}},
		}
	}
	ruletest.Compare(t, Rule, []ruletest.Case{
		{ID: "ev-2015-functions-ignored", Code: "function f(a) {}\nf(b);\nnew F(c);", Config: withEv(2015), Fix: true},
		{ID: "ev-2015-other-slots", Code: "var o = { a: 1 };\nvar a = [1];", Config: withEv(2015), Fix: true},
		{ID: "ev-2017-functions-apply", Code: "function f(a) {}", Config: withEv(2017), Fix: true},
		{ID: "ev-2022-functions-apply", Code: "f(a);", Config: withEv(2022), Fix: true},
		{ID: "ev-2016-arrow-ignored", Code: "var g = (a) => a;", Config: withEv(2016), Fix: true},
		{ID: "ev-2015-object-form", Code: "function f(a) {}",
			Config: map[string]any{
				"parserOptions": map[string]any{"ecmaVersion": 2015},
				"rules":         map[string]any{Name: []any{2, map[string]any{"functions": "always"}}},
			}, Fix: true},
	})
}

// TestNonASCIIParity covers the byte/code-unit boundary in the merged fix
// ranges this rule emits (see units.go).
func TestNonASCIIParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		{ID: "cjk-object", Code: "var o = { \"\u4f60\u597d\": 1 };", Options: []any{"always"}, Fix: true},
		{ID: "cjk-array", Code: "var a = [\"\u4f60\u597d\"];", Options: []any{"always"}, Fix: true},
		{ID: "cjk-array-trailing", Code: "var a = [\"\u4f60\u597d\",];", Fix: true},
		{ID: "emoji-call", Code: "f(\"\U0001f600\");", Options: []any{"always"}, Fix: true},
		{ID: "emoji-multiline", Code: "var o = {\n  a: \"\U0001f600\"\n};", Options: []any{"always-multiline"}, Fix: true},
		{ID: "accented-call", Code: "f(caf\u00e9);", Options: []any{"always"}, Fix: true},
		{ID: "accented-clean", Code: "var caf\u00e9 = 1;"},
	})
}
