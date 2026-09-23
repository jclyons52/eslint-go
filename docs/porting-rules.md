# Porting an ESLint rule to Go

This is the convention + API reference for rule packages under `rules/`. Every
rule is verified the same way: **its output must be identical to real ESLint's**
over the JS oracle (`oracle/lint_driver.js` runs the vendored eslint@8.57.0).
A rule is only ported when its parity test prints `PARITY PASS` with zero
mismatches.

## Layout

```
rules/<ruleid-without-dashes>/<name>.go        e.g. rules/nodebugger/nodebugger.go
rules/<ruleid-without-dashes>/<name>_test.go   e.g. rules/nodebugger/nodebugger_test.go
```

The package exports `Rule` (an `eslint.Rule`) and usually `Name` (the rule id).
Rule packages never touch the core (`eslint-go/*.go`) or each other's
directory.

```go
package nodebugger

import eslint "github.com/jclyons52/eslint-go"

const Name = "no-debugger"

var Rule = eslint.Rule{
    ID: Name,
    Meta: eslint.RuleMeta{
        Type:     "problem",            // problem | suggestion | layout
        Docs:     eslint.RuleDocs{Description: "…", Recommended: true, URL: "…"},
        Fixable:  "",                   // "" | "code" | "whitespace"
        Messages: map[string]string{"unexpected": "Unexpected 'debugger' statement."},
        Schema:   []any{},              // copy the original schema (documentation only)
    },
    Create: func(ctx *eslint.Context) map[string]func(eslint.Node) {
        return map[string]func(eslint.Node){
            "DebuggerStatement": func(node eslint.Node) { /* ctx.Report(...) */ },
            "ObjectExpression:exit": func(node eslint.Node) { /* leave handler */ },
        }
    },
}
```

`Create` returns exactly what the JS rule returns from `create()`: a map from
node type (with optional `:exit` suffix) to handler. Report order inside one
rule follows handler invocation order, which follows traversal order.

## The parity harness

```go
func TestParity(t *testing.T) {
    ruletest.Compare(t, Rule, []ruletest.Case{
        {ID: "basic", Code: "debugger;"},
        {ID: "warn", Code: "debugger;", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
        {ID: "with-options", Code: "…", Options: []any{map[string]any{"allow": []any{"warn"}}}},
        {ID: "fix-applied", Code: "var a=1;\n", Fix: true},   // compares verifyAndFix output too
        {ID: "globals", Code: "foo();", Config: map[string]any{"globals": map[string]any{"foo": "readonly"}}},
    })
}
```

- `Options` become the config value `[2, ...Options]`.
- `Config` replaces the whole config object (add `rules` yourself if you need
  extra keys); `globals`, `parserOptions`, `settings` and `env` are supported.
- `Fix: true` compares the fixed text *and* the post-fix messages — use it for
  any rule with `Fixable` set.
- `SourceType` defaults to `"module"` (the Go parser is module-only), and
  `ECMAVersion` to 2022. Don't write script-mode-only cases (`with`, legacy
  octal literals) — those are a documented parser divergence.
- The oracle needs `node` and `oracle/node_modules`; the harness `t.Skip`s
  cleanly when they are missing (never treat a skip as a pass).

**Corpus expectations.** A rule's cases should cover: the rule's default
behaviour, each option combination the rule supports, both severities, nested
scopes/blocks, a clean file that reports nothing, edge tokens (comments,
trailing commas, template literals), and at least one non-ASCII case for
anything token/layout related (see *Units* below). Copy realistic cases from the
rule's own doc examples and its `tests/lib/rules/<rule>.js` in the ESLint source
if it is available.

## The rule API

`*eslint.Context`:

| Go | JS equivalent |
|---|---|
| `ctx.ID`, `ctx.Severity`, `ctx.Options` | `context.id/severity/options` |
| `ctx.SourceCode` / `ctx.GetSourceCode()` | `context.sourceCode` |
| `ctx.Filename`, `ctx.PhysicalFilename` | `context.getFilename()` |
| `ctx.Settings`, `ctx.ParserOptions` | `context.settings`, `context.parserOptions` |
| `ctx.Option(i)`, `ctx.OptionString(i, def)`, `ctx.OptionBool(i, def)`, `ctx.OptionInt(i, def)`, `ctx.OptionMap(i)` | `context.options[i]` |
| `ctx.OptionMapValue(i, "key", def)` | `context.options[i].key` |
| `ctx.Report(eslint.Report{…})` | `context.report({…})` |
| `ctx.GetScope()` / `ctx.ScopeOf(node)` | `context.getScope()` / `sourceCode.getScope(node)` |
| `ctx.Ancestors()` | `context.getAncestors()` |
| `ctx.DeclaredVariables(node)` | `context.getDeclaredVariables(node)` |
| `ctx.MarkVariableAsUsed("name")` | `context.markVariableAsUsed("name")` |

`eslint.Report` mirrors the report descriptor:

```go
eslint.Report{
    Node:      node,                          // reported node
    Loc:       eslint.Loc(eslint.GetNode(node, "key")),  // optional loc override (LocOf / StartLoc helpers)
    MessageID: "unexpected",                  // or Message: "literal"
    Data:      map[string]any{"name": name},  // {{name}} interpolation
    Fix:       func(f *eslint.Fixer) *eslint.Fix { return f.RemoveRange(r) },
}
```

`eslint.SourceCode`:

| Go | JS |
|---|---|
| `Text()`, `AST()`, `Tokens()`, `Comments()`, `AllTokens()`, `Lines()` | same |
| `GetText(node)`, `GetTextRange([2]int)`, `GetTextBetween(a, b)` | `getText`, `text.slice` |
| `GetFirstToken(node, opts…)`, `GetLastToken`, `GetTokenBefore`, `GetTokenAfter` | same |
| `GetFirstTokens(node, n, opts…)`, `GetLastTokens`, `GetTokens`, `GetTokensBetween(a, b, opts…)` | same |
| `GetCommentsBefore/After/Inside(node)`, `GetAllComments()` | same |
| `GetNodeByRangeIndex(i)` | same |
| `GetIndexFromLoc(line, col)`, `GetLocFromIndex(off)` | same |
| `IsSpaceBetween(a, b)`, `IsSpaceBetweenTokens(a, b)` | `isSpaceBetween` |
| `Scope(node)`, `DeclaredVariables(node)`, `MarkVariableAsUsed(name, node)` | `getScope`, `getDeclaredVariables` |

Token query options are `eslint.TokenOpt{IncludeComments bool, Skip int, Filter func(Node) bool}`
(the JS `{includeComments, skip, filter}` object).

Node/token helpers (all exported, in the core package):

- `eslint.NodeType(n)`, `Get/GetNode/GetNodes/GetString/GetStringOK/GetBool/GetFloat/GetInt/GetMap`,
  `Range(n)`, `MustRange(n)`, `Start(n)`, `End(n)`, `Loc(n)`, `LocStart(n)`, `LocEnd(n)`,
  `LocOf(...)`, `StartLoc(...)`, `Parent(n)`, `AncestorsOf(n)`, `SameNode(a,b)`,
  `IsNode(x)`
- tokens: `IsCommentToken`, `IsPunctuatorToken(n, value)`, `IsCommaToken`,
  `IsSemicolonToken`, `IsOpeningBraceToken`/`IsClosingBraceToken`,
  `IsOpeningParenToken`/`IsClosingParenToken`, `IsOpeningBracketToken`/`IsClosingBracketToken`,
  `IsDotToken`, `IsStarToken`, `IsColonToken`, `IsArrowToken`, `IsNotToken`,
  `IsKeywordToken`, `IsIdentifierToken`, `IsStringToken`, `IsNumericToken`,
  `TokenValue(n)`, `TokenEndsStatement(n)`, and the `Token*`/`Comment*` type constants
- ast utils: `IsNullLiteral`, `IsStringLiteral`, `IsEmptyStringLiteral`,
  `GetStaticStringValue`, `GetStaticPropertyName` (returns `(string, bool)` —
  `bool=false` is JS `null`), `IsParenthesized`, `IsFunction`, `IsArrowFunction`,
  `IsLoop`, `IsLexicalDeclaration`, `SkipChainExpression`, `GetUpperFunction`,
  `GetInnermostScope`, `GetVariableByName`, `ModuleScope`, `IsDecimalInteger`,
  `IsDecimalIntegerNumericToken`, `IsDirectiveComment`, `GetUpperFunctionOrProgram`
- fixes: `eslint.Fixer` (`InsertTextAfter/Before`, `…Range`, `ReplaceText`,
  `ReplaceTextRange`, `Remove`, `RemoveRange`) and `eslint.NewFixTracker(fixer, sourceCode)`
  (`RetainRange`, `RetainEnclosingFunction`, `RetainSurroundingTokens`,
  `ReplaceTextRange`, `Remove`) — used where the JS rule uses `FixTracker`

If a helper is missing, write it **privately in your rule package** and mention
it in your final report; do not edit the core (concurrent ports would conflict).

## Pitfalls

1. **Byte vs UTF-16 code unit (units.go).** The Go port works in byte offsets
   internally; the *emitted* message (columns and fix ranges) is converted to
   ESLint's UTF-16 code-unit space automatically. So: compute offsets in bytes
   exactly as the JS computes them in code units, and do not hand-convert. For
   line-based arithmetic use `len(lines[i])` (bytes) — it corresponds to the JS
   `lines[i].length` once the emitted values are converted.
2. **Go regexp ≠ JS regexp.** `\uXXXX` is invalid in Go regexps — use `\x{XXXX}`.
   No lookbehind, no backreferences, no `u`-flag semantics. Rewrite the pattern
   rather than fighting it.
3. **Map iteration is random in Go.** Never put two handlers for the same node
   type in one map (impossible anyway) and never rely on map order for
   reporting; report order must follow code, not maps.
4. **`nil` vs empty.** A JS `name === null` check is `_, ok := …; !ok` in Go —
   `GetStaticPropertyName` returns `(name string, ok bool)`.
5. **Comments are not tokens** unless you ask for them
   (`TokenOpt{IncludeComments: true}`).
6. **Suggestions are out of scope.** If a rule's `meta.hasSuggestions` is true
   (or the oracle output contains a `suggestions` array), stop and report it —
   the message shape is not implemented.
7. **Code-path analysis rules are out of scope** (`no-unreachable`,
   `consistent-return`, `no-fallthrough`, `getter-return`, `constructor-super`).
8. **Don't weaken the corpus.** If a case mismatches, the rule (or your port of
   it) is wrong — narrowing the case list to make it green is not an option. If
   you conclude the *core* is wrong, say so explicitly in your report with the
   failing case and stop.

## Verification before you finish

```sh
gofmt -l rules/<pkg>            # must print nothing
go vet ./rules/<pkg>/
go test ./rules/<pkg>/          # must print PARITY PASS: rule <id> — N cases, 0 mismatches
```

Report: package path, rule id, number of cases, the exact PASS line, anything
skipped and why, and any core gap you hit.
