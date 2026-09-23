# eslint-go

A Go implementation of **ESLint 8.57.0** — the parser, the scope analysis, the
rule engine and the CLI — where every layer is validated against the real thing
rather than reimplemented from the docs.

The headline claim is testable: lint the same files with the same config and
`eslint-go --format stylish src` produces **the same bytes** as
`eslint --format stylish src` — colours, columns, summary, exit code, and the
files rewritten by `--fix`.

```console
$ go build -o eslint-go ./cmd/eslint-go
$ ./eslint-go src
/app/src/fields.js
   3:19  warning  Trailing spaces not allowed      no-trailing-spaces
   5:5   error    Duplicate key 'mode'             no-dupe-keys
  12:5   error    Duplicate key 'level'            no-dupe-keys
  16:5   error    Unexpected 'debugger' statement  no-debugger
  17:12  error    'missingGlobal' is not defined   no-undef

✖ 5 problems (4 errors, 1 warning)
  0 errors and 1 warning potentially fixable with the `--fix` option.
```

Nothing here shells out to Node. `eslint-go` parses the text in-process with
[espree-go](https://github.com/jclyons52/espree-go), analyzes scopes with
[eslint-scope-go](https://github.com/jclyons52/eslint-scope-go), runs ported
rules over one traversal, and applies fixes with ESLint's own 10-pass loop.

## Rule coverage

<!-- rule-table:start -->
46 rules are ported and verified. Every one is checked against the real ESLint by a parity test that fails on any difference.

| rule | package | fixable | oracle cases |
|---|---|---|---|
| `comma-dangle` | `rules/commadangle` | code | 83 |
| `curly` | `rules/curly` | code | 56 |
| `dot-notation` | `rules/dotnotation` | code | 63 |
| `eol-last` | `rules/eollast` | whitespace | 39 |
| `eqeqeq` | `rules/eqeqeq` | code | 87 |
| `no-alert` | `rules/noalert` | — | 47 |
| `no-array-constructor` | `rules/noarrayconstructor` | — | 33 |
| `no-caller` | `rules/nocaller` | — | 30 |
| `no-cond-assign` | `rules/nocondassign` | — | 38 |
| `no-console` | `rules/noconsole` | — | 33 |
| `no-debugger` | `rules/nodebugger` | — | 16 |
| `no-dupe-args` | `rules/nodupeargs` | — | 30 |
| `no-dupe-keys` | `rules/nodupekeys` | — | 17 |
| `no-duplicate-case` | `rules/noduplicatecase` | — | 49 |
| `no-else-return` | `rules/noelsereturn` | code | 52 |
| `no-empty` | `rules/noempty` | — | 28 |
| `no-eval` | `rules/noeval` | — | 91 |
| `no-extra-boolean-cast` | `rules/noextrabooleancast` | code | 58 |
| `no-extra-semi` | `rules/noextrasemi` | code | 62 |
| `no-floating-decimal` | `rules/nofloatingdecimal` | code | 32 |
| `no-func-assign` | `rules/nofuncassign` | — | 33 |
| `no-lonely-if` | `rules/nolonelyif` | code | 40 |
| `no-mixed-spaces-and-tabs` | `rules/nomixedspacesandtabs` | — | 44 |
| `no-multi-spaces` | `rules/nomultispaces` | whitespace | 42 |
| `no-multiple-empty-lines` | `rules/nomultipleemptylines` | whitespace | 44 |
| `no-new-object` | `rules/nonewobject` | — | 33 |
| `no-new-wrappers` | `rules/nonewwrappers` | — | 31 |
| `no-redeclare` | `rules/noredeclare` | — | 41 |
| `no-self-compare` | `rules/noselfcompare` | — | 63 |
| `no-shadow` | `rules/noshadow` | — | 53 |
| `no-sparse-arrays` | `rules/nosparsearrays` | — | 22 |
| `no-throw-literal` | `rules/nothrowliteral` | — | 44 |
| `no-trailing-spaces` | `rules/notrailingspaces` | whitespace | 22 |
| `no-undef` | `rules/noundef` | — | 24 |
| `no-unsafe-negation` | `rules/nounsafenegation` | — | 52 |
| `no-unused-vars` | `rules/nounusedvars` | — | 196 |
| `no-var` | `rules/novar` | code | 68 |
| `no-whitespace-before-property` | `rules/nowhitespacebeforeproperty` | whitespace | 45 |
| `prefer-const` | `rules/preferconst` | code | 54 |
| `quotes` | `rules/quotes` | code | 77 |
| `semi` | `rules/semi` | code | 130 |
| `semi-spacing` | `rules/semispacing` | whitespace | 69 |
| `space-infix-ops` | `rules/spaceinfixops` | whitespace | 55 |
| `use-isnan` | `rules/useisnan` | — | 57 |
| `valid-typeof` | `rules/validtypeof` | — | 78 |
| `yoda` | `rules/yoda` | code | 76 |

Total oracle cases across the rule suites: 2437.
<!-- rule-table:end -->

Rules are added as independent packages under `rules/<name>/`; each one ships a
parity test that runs the real ESLint over a corpus and diffs the resulting
message objects (and the `--fix` output where the rule is fixable).
`docs/porting-rules.md` is the API reference a rule port follows.

## How it is verified

There are four gates, and all of them compare against the npm original running
in `oracle/` (`npm install` there first):

| gate | what it compares | command |
|---|---|---|
| end-to-end CLI | stdout, exit code and `--fix` file contents vs `eslint` | `scripts/e2e.sh` |
| rules | per-rule message JSON + fixed text vs `Linter.verify`/`verifyAndFix` | `go test ./rules/...` |
| parser + scope | full ASTs / ScopeManager serializations vs `espree` / `eslint-scope` | in the sibling repos |
| whole suite | build, vet, fmt, tests | `go build ./... && go vet ./... && gofmt -l . && go test ./...` |

`scripts/e2e.sh` builds the CLI and runs both tools over
`testdata/e2e/` (duplicate keys, an undeclared global, a syntax error,
non-ASCII text, trailing whitespace) and `testdata/e2e-script/` (sloppy-mode
script: duplicate parameters, `with`, legacy octals) across fourteen scenarios —
stylish, coloured stylish, JSON, `--quiet`, `--max-warnings`, a single file, a
glob, stdin, `--fix`, `--fix-dry-run` — and requires identical results in every
one:

```
PASS [stylish] (exit 1, stdout 19 lines)
PASS [script-stylish] (exit 1, stdout 8 lines)
...
e2e: 14 passed, 0 failed
```

Parity tests skip cleanly when `node` or `oracle/node_modules` is missing, so
the suite still runs on a machine without the oracle — a skip is never a pass.

## Layout

```
eslint.go        package doc + layer map
linter.go        Linter.verify / VerifyAndFix: parse → scope → rules → sort → fix
config.go        eslintrc config (rules, parserOptions, globals, settings, env)
parse.go         espree-go integration, parse-error normalisation
traverser.go     the keyed DFS that dispatches node events to rules
source_code.go   SourceCode: text, AST, tokens, comments, lines, scopes
token_store.go   the token query API (getFirstToken/getTokenBefore/skip/filter)
units.go         the byte ↔ UTF-16 code-unit boundary (see below)
scope.go         eslint-scope-go integration, getScope/markVariableAsUsed
rule.go          the rule authoring API (Context, Report)
report.go        report-translator: createProblem, interpolate, LintMessage
fixer.go         rule-fixer + source-code-fixer (applyFixes)
fix_tracker.go   FixTracker (retainRange / retainSurroundingTokens)
ast_utils.go     the Ported rules/utils/ast-utils subset
builtin.go       generated: eslint:recommended, deprecated rules, replacements
formatters/      stylish, json, compact, unix + text-table + a chalk v4 subset
rules/           one package per ported rule (+ its parity test)
cmd/eslint-go/   the CLI: flags, config cascade, glob expansion, --fix
internal/ruletest/ the JS-oracle harness every rule test uses
```

## Implementation notes worth knowing

**Byte offsets vs UTF-16 code units.** ESLint's positions are UTF-16 code-unit
indices; this port works in byte offsets internally (acorn-go reports bytes, so
ranges stay mutually consistent and slicing a Go string by a range is what
actually produces fixed text). The two spaces agree on ASCII and diverge on any
non-ASCII line, so the conversion happens at exactly one boundary — when a
message is emitted (`units.go`). Reported columns and fix ranges are therefore
ESLint's, and `--fix` output is identical.

**chalk's per-line re-open.** ESLint's coloured output repeats the style's close
and open around every newline (`stringEncaseCRLFWithFirstIndex`). Replicating
that is what makes the coloured `stylish` output byte-identical rather than
merely close.

**Unknown rules are reported, not ignored.** Configuring a rule that this port
does not implement produces the same problem ESLint produces for a missing rule
(`Definition for rule 'x' was not found.`, or the "was removed and replaced by"
variant from `conf/replacements.json`), at the same 1:1 position — so a config
that references unported rules is visibly incomplete instead of silently quieter.

## Known divergences

These are deliberate and documented rather than hidden. Everything else aims at
byte equality.

- **`sourceType` is parsed as a module.** The underlying `acorn-go` parser is
  module-only, so script-only syntax (`with`, legacy octal literals, duplicate
  parameters in sloppy mode) fails to parse where ESLint with
  `sourceType: "script"` accepts it. Already-strict code parses identically.
- **Newer syntax may parse.** `espree-go` parses at acorn's *latest* grammar,
  while ESLint 8.57 caps at ES2024 (ecmaVersion 15, which is what this port uses
  for scope analysis). A file using post-ES2024 syntax can therefore parse here
  and fail there.
- **Inline directives are not implemented.** `/* eslint-disable */`,
  `/* eslint rule: "error" */`, `/* global x */` and `/* exported x */` comments
  do not affect a run, so files that rely on them differ from ESLint.
- **Code-path analysis rules are out of scope** (`no-unreachable`,
  `consistent-return`, `no-fallthrough`, `getter-return`, `constructor-super`).
- **`parserOptions.ecmaVersion` is not enforced.** The port parses at the newest
  syntax level, so a configured older version's restrictions (e.g. `ecmaVersion:
  5` rejecting `let`) are not applied. `sourceType` *is* honoured: `module`
  parses as strict module code and `script`/`commonjs` as sloppy script code.
- **`extends`** supports `"eslint:recommended"` (filtered to implemented rules)
  and relative paths; shareable configs from npm and plugins are not resolved.
- **Formatters**: `stylish`, `json`, `compact` and `unix` only.
- **CLI flags** implemented: `-c/--config`, `--no-eslintrc`, `-f/--format`,
  `-o/--output-file`, `--ext`, `--ignore-pattern`, `--no-ignore`,
  `--global`, `--env`, `--fix`, `--fix-dry-run`, `--quiet`, `--color`,
  `--no-color`, `--stdin`, `--stdin-filename`, `--max-warnings`, `--list-rules`.
  Not implemented: `--cache`, `--fix-type`, `--print-config`, `--rulesdir`,
  `--report-unused-disable-directives`, plugins, `--init`.

## Scope notes that are not divergences

Two behaviours look like gaps but are what ESLint 8.57 itself does, and were
reproduced deliberately after measuring the reference implementation:

- **Only the ES5 builtin globals are defined.** Despite
  `parserOptions.ecmaVersion: 2022`, ESLint's eslintrc path defines exactly
  `conf/globals.js`'s `es5` set: probing all 38 `es5` names and all 28 later
  names through 8.57 shows `Array`/`JSON`/`escape` defined and `Map`, `Promise`,
  `globalThis`, `WeakRef` *not* defined. `env`/`globals` then layer on top.
- **`suggestions` are emitted, never applied.** Suggestion entries appear in
  `--format json` output (with descendants, `data` and their fix), and `--fix`
  ignores them — matching ESLint, where `--fix-type suggestion` is required.

## Why bother

Because "we ported ESLint to Go" is only interesting if it is *the same
linter*: same rules, same messages, same fixes. The oracle harnesses are how
that claim stays honest — they are the reason this repository can state exact
case counts instead of describing intent.

## Credits and licence

The behaviour, message strings and rule semantics are ESLint's, by its authors
and contributors (MIT). This repository is a Go translation of that behaviour;
`LICENSE` carries the upstream licence text and `NOTICE` explains the
relationship. The npm packages themselves are never vendored into git — they are
installed under `oracle/` as the test-time reference.

The ported rule packages derive from `eslint/lib/rules/*.js` (MIT), and the
supporting layers from `espree`, `acorn`, `estraverse`, `esquery` and
`eslint-scope` (BSD-2-Clause / BSD-3-Clause / MIT — see the sibling repos).
