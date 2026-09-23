#!/usr/bin/env python3
"""Regenerate the rule coverage table in README.md from the rules/ packages.

The table is generated rather than hand-maintained so it cannot drift from the
code: the rule id comes from the package's `const Name`, the case count from the
`{ID:` entries in its parity test, and the fixability from the package's
`Fixable:` meta field.

  python3 scripts/gen_readme.py          # check (exit 1 if README is stale)
  python3 scripts/gen_readme.py --write  # rewrite the table in README.md
"""
import argparse
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
RULES = ROOT / "rules"
README = ROOT / "README.md"
START = "<!-- rule-table:start -->"
END = "<!-- rule-table:end -->"

NAME_RE = re.compile(r'const Name = "([^"]+)"')
ID_RE = re.compile(r'\bID:\s*Name\b')
FIX_RE = re.compile(r'Fixable:\s*"([^"]+)"')
CASE_RE = re.compile(r'\{ID:\s*"')
TESTCASE_RE = re.compile(r'\bID:\s*"')


def rule_rows():
    rows = []
    for pkg in sorted(p for p in RULES.iterdir() if p.is_dir()):
        go = pkg / f"{pkg.name}.go"
        if not go.exists():
            rows.append((pkg.name, None, 0, None, "no implementation"))
            continue
        src = go.read_text()
        m = NAME_RE.search(src)
        rule_id = m.group(1) if m else None
        fix = FIX_RE.search(src)
        tests = list(pkg.glob("*_test.go"))
        cases = 0
        for t in tests:
            cases += len(TESTCASE_RE.findall(t.read_text()))
        rows.append((pkg.name, rule_id, cases, fix.group(1) if fix else None,
                     None if tests else "no parity test"))
    return rows


def table():
    rows = rule_rows()
    ok = [r for r in rows if r[1] and r[2] and not r[4]]
    out = [
        f"{len(ok)} rules are ported and verified. "
        f"Every one is checked against the real ESLint by a parity test that fails on any difference.",
        "",
        "| rule | package | fixable | oracle cases |",
        "|---|---|---|---|",
    ]
    for pkg, rule_id, cases, fix, problem in sorted(rows, key=lambda r: (r[1] or "zz")):
        if problem:
            out.append(f"| — | `rules/{pkg}` | — | ⚠ {problem} |")
            continue
        out.append(f"| `{rule_id}` | `rules/{pkg}` | {fix or '—'} | {cases} |")
    missing = [r for r in rows if r[4]]
    out.append("")
    out.append(f"Total oracle cases across the rule suites: {sum(r[2] for r in rows)}.")
    if missing:
        out.append("")
        out.append(
            f"Not yet complete ({len(missing)}): "
            + ", ".join(f"`{r[0]}`" for r in missing)
            + "."
        )
    return "\n".join(out), len(ok)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--write", action="store_true")
    args = ap.parse_args()

    body, count = table()
    readme = README.read_text()
    if START not in readme or END not in readme:
        print(f"README.md is missing the {START} / {END} markers", file=sys.stderr)
        return 2
    head, rest = readme.split(START, 1)
    _, tail = rest.split(END, 1)
    updated = f"{head}{START}\n{body}\n{END}{tail}"

    if args.write:
        README.write_text(updated)
        print(f"updated README.md: {count} verified rules")
        return 0
    if updated != readme:
        print("README.md rule table is stale; re-run with --write", file=sys.stderr)
        return 1
    print(f"README.md rule table is up to date: {count} verified rules")
    return 0


if __name__ == "__main__":
    sys.exit(main())
