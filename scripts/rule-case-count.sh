#!/usr/bin/env bash
# rule-case-count.sh — cross-check the published oracle-case total against what
# the suite actually executes.
#
# The README and the registry state a case count ("46 rules, 2432 oracle
# cases"). Those numbers are derived statically from the test files, so they can
# drift from reality: a case left commented out as a note about a known gap is
# counted by a naive text search but never runs. This script runs every rule
# package, sums the "PARITY PASS: rule X — N cases" lines the harness prints, and
# compares that with the declared total. They must agree.
#
# Usage: scripts/rule-case-count.sh
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

packages=$(ls rules/*/*_test.go | xargs -n1 dirname | xargs -n1 basename | sort -u)
results=$(mktemp)
trap 'rm -f "$results"' EXIT

for pkg in $packages; do
    # A package may run more than one parity test function (e.g. the rule and
    # its --fix behaviour), so keep every parity line, not just the last.
    go test -count=1 -v "./rules/$pkg/" 2>&1 | grep -E 'PARITY (PASS|FAIL)' | sed "s/^/$pkg :: /" >> "$results"
done

python3 - "$results" <<'PY'
import re, subprocess, sys

lines = [l.rstrip("\n") for l in open(sys.argv[1]) if l.strip()]
seen = set()
executed = 0
failures = []
for line in lines:
    pkg = line.split(" :: ", 1)[0]
    seen.add(pkg)
    if "PARITY FAIL" in line:
        failures.append(line)
        continue
    m = re.search(r"(\d+) cases, 0 mismatches", line)
    if m:
        executed += int(m.group(1))
    else:
        failures.append("unreadable parity line: " + line)

declared_out = subprocess.run(
    [sys.executable, "scripts/gen_rule_registry.py"], capture_output=True, text=True
)
m = re.search(r"(\d+) rules?, (\d+) oracle cases", declared_out.stdout + declared_out.stderr)
declared = int(m.group(2)) if m else None

print()
print("rule packages run:    %d" % len(seen))
print("cases executed:       %d" % executed)
print("cases declared:       %s" % ("<unavailable>" if declared is None else declared))
print("rule parity failures: %d" % len(failures))
for f in failures[:10]:
    print("  " + f)

if failures:
    print("RESULT: FAIL (%d rule packages did not report a clean parity result)" % len(failures))
    sys.exit(1)
if declared is not None and executed != declared:
    print("RESULT: FAIL (declared %d != executed %d — a commented-out or unused case"
          " literal is being counted, or a declared case is not running)" % (declared, executed))
    sys.exit(1)
print("RESULT: PASS (every declared case ran and matched)")
PY
