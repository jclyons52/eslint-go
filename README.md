# eslint-go

A Go port of the **ESLint 8.57.0 Linter pipeline**, validated by a JS-oracle
parity test against the real `Linter().verify()`. This is the goal layer of
the ESLint-in-Go effort: it runs real ESLint rules and produces byte-identical
lint output, on top of the already-ported parser chain
(`espree-go` → `acorn-go` → `estraverse-go` → `eslint-scope-go`).

## Status — Linter core + starter rules

This is the **first Linter milestone**: a faithful, minimal `Linter.verify`
that reproduces the observable contract of real ESLint for the supported
rules — traversal, rule dispatch, config severity/options normalization, and
the report-translator's precise message object. It currently runs:

- `no-debugger` — pure traversal + report
- `no-dupe-keys` — exercises `data` interpolation, `:exit` selectors,
  `node.parent`, get/set dedup, and static property-name resolution

Not yet wired (documented next milestones): rule fixes/suggestions
(`rule-fixer`), the token-store (`getTokenBefore/After`), scope integration
into SourceCode (`getScope`, via `eslint-scope-go`), and inline
disable-directives / config-comment parsing.

## Files

- `linter.go` — `Linter`, config severity/options normalization, rule dispatch
- `source_code.go` — `SourceCode`, the traverser, `node.parent` attachment
- `report.go` — report-translator (message object), `interpolate`
- `rules.go` — rule context + the implemented rules
- `ast_utils.go` — `getStaticPropertyName` / `getStaticStringValue`
- `visitor_keys.go` — the canonical ESTree visitor-keys table
- `original/` — vendored real ESLint 8.57 source (for reference)
- `oracle/` — Node env (`eslint@8.57.0` + `espree`) as the parity oracle

## Parity

`parity_test.go` runs the oracle over a 19-case corpus and compares the full
`verify()` message output (ruleId, severity, message, line, column, nodeType,
messageId, endLine, endColumn) between the Go port and real ESLint.

```
PARITY PASS: 19 cases, 0 mismatches
```

## Verify

```
go build ./... && go vet ./... && gofmt -l . && go test ./...
```

The parity test skips cleanly when `node` or the oracle deps aren't installed
(`npm install` in `oracle/` first).
