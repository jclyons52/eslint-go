package noelsereturn

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		// ---- default (allowElseIf: true) ----
		{ID: "basic-block", Code: "function f() {\n  if (a) {\n    return 1;\n  } else {\n    return 2;\n  }\n}\n", Fix: true},
		{ID: "basic-no-block", Code: "function f() {\n  if (a) return 1;\n  else return 2;\n}\n", Fix: true},
		{ID: "one-line", Code: "function f() { if (a) { return 1; } else { return 2; } }\n", Fix: true},
		{ID: "clean-no-else", Code: "function f() {\n  if (a) {\n    return 1;\n  }\n  return 2;\n}\n"},
		{ID: "clean-no-return", Code: "function f() {\n  if (a) {\n    g();\n  } else {\n    h();\n  }\n}\n"},
		{ID: "clean-else-if-partial", Code: "function f() {\n  if (a) {\n    return 1;\n  } else if (b) {\n    return 2;\n  }\n}\n"},
		{ID: "clean-else-if-not-all-return", Code: "function f() {\n  if (a) {\n    return 1;\n  } else if (b) {\n    g();\n  } else {\n    return 3;\n  }\n}\n"},
		{ID: "warn-severity", Code: "function f() { if (a) { return 1; } else { return 2; } }\n", Config: map[string]any{"rules": map[string]any{Name: "warn"}}, Fix: true},

		// ---- if/else-if chains whose branches all return ----
		{ID: "chain-all-return", Code: "function f() {\n  if (a) {\n    return 1;\n  } else if (b) {\n    return 2;\n  } else {\n    return 3;\n  }\n}\n", Fix: true},
		{ID: "chain-no-blocks", Code: "function f() {\n  if (a) return 1;\n  else if (b) return 2;\n  else return 3;\n}\n", Fix: true},
		{ID: "chain-mixed", Code: "function f() {\n  if (a) return 1;\n  else if (b) {\n    return 2;\n  } else {\n    return 3;\n  }\n}\n", Fix: true},
		{ID: "return-not-last", Code: "function f() {\n  if (a) {\n    g();\n    return 1;\n  } else {\n    return 2;\n  }\n}\n", Fix: true},
		{ID: "nested-if-returns", Code: "function f() {\n  if (a) {\n    if (b) {\n      return 1;\n    } else {\n      return 2;\n    }\n  } else {\n    return 3;\n  }\n}\n", Fix: true},
		{ID: "nested-both-else", Code: "function f() {\n  if (a) {\n    if (b) {\n      return 1;\n    } else {\n      return 2;\n    }\n  } else {\n    if (c) {\n      return 3;\n    } else {\n      return 4;\n    }\n  }\n}\n", Fix: true},
		{ID: "consequent-if-no-else", Code: "function f() {\n  if (a) {\n    if (b) {\n      return 1;\n    }\n  } else {\n    return 2;\n  }\n}\n"},
		{ID: "empty-consequent-block", Code: "function f() {\n  if (a) {\n  } else {\n    return 2;\n  }\n}\n"},

		// ---- scopes where the fix must be withheld ----
		{ID: "collision-let-after", Code: "function f() {\n  if (a) {\n    return 1;\n  } else {\n    let x = 2;\n  }\n  let x = 1;\n}\n"},
		{ID: "collision-var-hoisted", Code: "function f() {\n  if (a) {\n    return 1;\n  } else {\n    let x = 2;\n  }\n  if (b) {\n    var x;\n  }\n}\n"},
		{ID: "collision-param", Code: "function f(x) {\n  if (a) {\n    return 1;\n  } else {\n    let x = 2;\n  }\n}\n"},
		{ID: "collision-through", Code: "function f() {\n  if (a) {\n    return 1;\n  } else {\n    let x = 2;\n  }\n  return x;\n}\n"},
		{ID: "collision-catch-var", Code: "function f() {\n  try {\n    g();\n  } catch (e) {\n    if (a) {\n      return 1;\n    } else {\n      let e = 2;\n    }\n  }\n}\n"},
		{ID: "safe-inner-block", Code: "function f() {\n  if (a) {\n    return 1;\n  } else {\n    let x = 2;\n    return x;\n  }\n}\n", Fix: true},
		{ID: "safe-nested-block", Code: "function f() {\n  {\n    if (a) {\n      return 1;\n    } else {\n      let y = 2;\n      return y;\n    }\n  }\n}\n", Fix: true},
		{ID: "collision-nested-block-name", Code: "function f() {\n  let y = 0;\n  {\n    if (a) {\n      return 1;\n    } else {\n      let y = 2;\n    }\n  }\n}\n"},

		// ---- ASI hazards ----
		{ID: "asi-else-starts-paren", Code: "function f() {\n  if (a) return 1\n  else (b)();\n}\n"},
		{ID: "asi-else-starts-bracket", Code: "function f() {\n  if (a) return 1\n  else [1].forEach(g);\n}\n"},
		{ID: "asi-else-starts-plus", Code: "function f() {\n  if (a) return 1\n  else +1;\n}\n"},
		{ID: "asi-else-starts-minus", Code: "function f() {\n  if (a) return 1\n  else -1;\n}\n"},
		{ID: "asi-else-starts-slash", Code: "function f() {\n  if (a) return 1\n  else /x/.test(b);\n}\n"},
		{ID: "asi-else-starts-backtick", Code: "function f() {\n  if (a) return 1\n  else `x`;\n}\n"},
		{ID: "asi-else-starts-ident", Code: "function f() {\n  if (a) return 1\n  else b();\n}\n", Fix: true},
		{ID: "asi-block-consequent", Code: "function f() {\n  if (a) {\n    return 1\n  } else (b)();\n}\n", Fix: true},
		{ID: "asi-next-token-same-line", Code: "function f() { if (a) { return 1; } else { return 2 } g(); }\n"},
		{ID: "asi-next-token-paren", Code: "function f() { if (a) { return 1; } else { return 2 }\n(f()); }\n"},
		{ID: "asi-next-token-brace", Code: "function f() { if (a) { return 1; } else { return 2 } }\n", Fix: true},
		{ID: "asi-no-semicolon-newline", Code: "function f() {\n  if (a) {\n    return 1;\n  } else {\n    return 2\n  }\n}\n", Fix: true},

		// ---- only-one-statement positions ----
		{ID: "position-arrow-body", Code: "const f = () => {\n  if (a) {\n    return 1;\n  } else {\n    return 2;\n  }\n};\n", Fix: true},
		{ID: "position-else-if-consequent", Code: "function f() {\n  if (a) {\n    return 1;\n  } else if (b) {\n    if (c) {\n      return 2;\n    } else {\n      return 3;\n    }\n  }\n}\n", Fix: true},
		{ID: "position-switch-case", Code: "function f() {\n  switch (x) {\n    case 1:\n      if (a) {\n        return 1;\n      } else {\n        return 2;\n      }\n  }\n}\n", Fix: true},
		{ID: "position-static-block", Code: "class C {\n  static {\n    if (a) {\n      return;\n    } else {\n      return;\n    }\n  }\n}\n"},
		{ID: "position-labelled", Code: "function f() {\n  loop: if (a) {\n    return 1;\n  } else {\n    return 2;\n  }\n}\n"},

		// ---- allowElseIf: false ----
		{ID: "no-elseif-block", Code: "function f() {\n  if (a) {\n    return 1;\n  } else if (b) {\n    return 2;\n  }\n}\n", Options: []any{map[string]any{"allowElseIf": false}}, Fix: true},
		{ID: "no-elseif-no-block", Code: "function f() {\n  if (a) return 1;\n  else if (b) return 2;\n}\n", Options: []any{map[string]any{"allowElseIf": false}}, Fix: true},
		{ID: "no-elseif-plain-else", Code: "function f() {\n  if (a) {\n    return 1;\n  } else {\n    return 2;\n  }\n}\n", Options: []any{map[string]any{"allowElseIf": false}}, Fix: true},
		{ID: "no-elseif-chain", Code: "function f() {\n  if (a) {\n    return 1;\n  } else if (b) {\n    return 2;\n  } else {\n    return 3;\n  }\n}\n", Options: []any{map[string]any{"allowElseIf": false}}, Fix: true},
		{ID: "no-elseif-not-returning", Code: "function f() {\n  if (a) {\n    return 1;\n  } else if (b) {\n    g();\n  }\n}\n", Options: []any{map[string]any{"allowElseIf": false}}},
		{ID: "allow-elseif-explicit-true", Code: "function f() {\n  if (a) {\n    return 1;\n  } else if (b) {\n    return 2;\n  }\n}\n", Options: []any{map[string]any{"allowElseIf": true}}},

		// ---- non-ASCII ----
		{ID: "nonascii-before", Code: "var s = \"h\u00e9llo\";\nfunction f() {\n  if (a) {\n    return 1;\n  } else {\n    return 2;\n  }\n}\n", Fix: true},
		{ID: "nonascii-inside", Code: "function f() {\n  if (a) {\n    return \"\u4f60\u597d\";\n  } else {\n    return \"\U0001f600\";\n  }\n}\n", Fix: true},
		{ID: "nonascii-let", Code: "function f() {\n  if (a) {\n    return 1;\n  } else {\n    let x = \"\u4f60\u597d\";\n  }\n  let x = 1;\n}\n"},
	})
}
