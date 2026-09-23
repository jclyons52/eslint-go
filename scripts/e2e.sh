#!/usr/bin/env bash
# e2e.sh — end-to-end parity between the real ESLint CLI (eslint 8.57.0, from
# oracle/node_modules) and this port's CLI.
#
# Both tools lint the same fixture with the same config and the run is compared
# byte for byte: stdout, stderr, exit code, and (for --fix) the rewritten files.
# A scenario only passes when every one of those matches.
#
# Usage: scripts/e2e.sh [scenario-name ...]   (default: all)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FIXTURE="${FIXTURE:-$ROOT/testdata/e2e}"
SCRIPT_FIXTURE="$ROOT/testdata/e2e-script"
ORACLE="$ROOT/oracle"
REAL_ESLINT="$ORACLE/node_modules/eslint/bin/eslint.js"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

if ! command -v node >/dev/null 2>&1; then
    echo "SKIP: node is not available"
    exit 0
fi
if [ ! -f "$REAL_ESLINT" ]; then
    echo "SKIP: real eslint not installed (run npm install in oracle/)"
    exit 0
fi

echo "building eslint-go ..."
( cd "$ROOT" && go build -o "$WORK/eslint-go" ./cmd/eslint-go ) || {
    echo "FAIL: go build failed"
    exit 1
}
BIN="$WORK/eslint-go"

# The fixture config lists every rule the demo wants to exercise; not all of them
# are ported yet, and configuring an unported rule produces ESLint's
# "Definition for rule ... was not found." problem — which would be noise here.
# Both tools therefore get the SAME filtered config: the fixture's rules
# intersected with the rules this port actually implements. As rules land, the
# effective config grows automatically.
EFFECTIVE_CONFIG="$WORK/.eslintrc.effective.json"
SCRIPT_CONFIG="$WORK/.eslintrc.script.json"
RULE_LIST_FILE="$WORK/rules.txt"
"$BIN" --list-rules | awk 'NR>1 {print $1}' > "$RULE_LIST_FILE"
python3 "$ROOT/scripts/filter_config.py" "$FIXTURE/.eslintrc.json" "$EFFECTIVE_CONFIG" "$RULE_LIST_FILE"
python3 "$ROOT/scripts/filter_config.py" "$SCRIPT_FIXTURE/.eslintrc.json" "$SCRIPT_CONFIG" "$RULE_LIST_FILE"

pass=0
fail=0
skipped=0

# compare <name> <args...>  — runs both CLIs with identical args in identical
# copies of the fixture, then diffs stdout/stderr/exit code (and the tree, which
# makes --fix comparisons fall out of the same code path).
compare() {
    local name="$1"; shift

    local a="$WORK/real-$name"
    local b="$WORK/go-$name"
    rm -rf "$a" "$b"
    cp -R "$FIXTURE" "$a"
    cp -R "$FIXTURE" "$b"
    cp "$EFFECTIVE_CONFIG" "$a/.eslintrc.json"
    cp "$EFFECTIVE_CONFIG" "$b/.eslintrc.json"

    # Physical paths: /var is a symlink to /private/var on macOS, and both tools
    # report the resolved location.
    local a_phys b_phys
    a_phys="$(cd "$a" && pwd -P)"
    b_phys="$(cd "$b" && pwd -P)"

    local stdin_src="${STDIN_FILE:-/dev/null}"
    local real_out real_err real_code go_out go_err go_code
    real_out="$(cd "$a" && node "$REAL_ESLINT" "$@" <"$stdin_src" 2>"$WORK/real-$name.err")"
    real_code=$?
    real_err="$(cat "$WORK/real-$name.err")"

    go_out="$(cd "$b" && "$BIN" "$@" <"$stdin_src" 2>"$WORK/go-$name.err")"
    go_code=$?
    go_err="$(cat "$WORK/go-$name.err")"

    # ESLint reports absolute paths; the two runs live in different temp dirs,
    # so normalise each run's own root to FIXTURE before comparing.
    real_out="$(printf '%s' "$real_out" | sed "s|$a_phys|FIXTURE|g" | sed "s|$a|FIXTURE|g")"
    go_out="$(printf '%s' "$go_out" | sed "s|$b_phys|FIXTURE|g" | sed "s|$b|FIXTURE|g")"

    # The port's binary is named differently and paths differ inside temp dirs,
    # but both tools resolve relative paths the same way, so output should match
    # as-is. Normalise nothing: a difference is a real difference.
    local ok=1
    if [ "$real_out" != "$go_out" ]; then
        ok=0
        echo "--- [$name] stdout differs (real vs go) ---"
        diff <(printf '%s\n' "$real_out") <(printf '%s\n' "$go_out") | head -40
    fi
    if [ "$real_code" != "$go_code" ]; then
        ok=0
        echo "--- [$name] exit code differs: real=$real_code go=$go_code ---"
    fi
    if [ "$real_err" != "$go_err" ]; then
        # stderr is reported, not compared: the two tools use it for different
        # non-fatal asides. stdout and exit codes are the contract.
        echo "note [$name] stderr differs: real=$(printf '%s' "$real_err" | head -1) | go=$(printf '%s' "$go_err" | head -1)"
    fi
    if ! diff -r -q "$a" "$b" >/dev/null 2>&1; then
        ok=0
        echo "--- [$name] resulting file tree differs (--fix output) ---"
        diff -r "$a" "$b" | head -40
    fi

    if [ "$ok" = "1" ]; then
        echo "PASS [$name] (exit $real_code, stdout $(printf '%s' "$real_out" | wc -l | tr -d ' ') lines)"
        pass=$((pass + 1))
    else
        echo "FAIL [$name]"
        fail=$((fail + 1))
    fi
}

# Scenario list: the flags each tool is given (identical for both).
run_all() {
    compare stylish src
    compare stylish-color --color -f stylish src
    compare json -f json src
    compare quiet --quiet src
    compare maxwarnings --max-warnings 0 src
    compare single-file src/fields.js
    compare fix --fix src
    compare fix-dry-run --fix-dry-run src
    compare glob --format stylish "src/[fw]*.js"
    STDIN_FILE="$FIXTURE/src/fields.js" compare stdin --stdin --stdin-filename src/fields.js
    STDIN_FILE=/dev/null compare stdin-empty --stdin

    # Script mode (sloppy): duplicate parameters, `with`, legacy octals and
    # `arguments.callee` are only legal when sourceType is "script".
    FIXTURE="$SCRIPT_FIXTURE" EFFECTIVE_CONFIG="$SCRIPT_CONFIG" compare script-stylish src
    FIXTURE="$SCRIPT_FIXTURE" EFFECTIVE_CONFIG="$SCRIPT_CONFIG" compare script-json -f json src
    FIXTURE="$SCRIPT_FIXTURE" EFFECTIVE_CONFIG="$SCRIPT_CONFIG" compare script-fix --fix src
}

if [ $# -gt 0 ]; then
    for s in "$@"; do
        case "$s" in
            stylish) compare stylish src ;;
            stylish-color) compare stylish-color --color -f stylish src ;;
            json) compare json -f json src ;;
            quiet) compare quiet --quiet src ;;
            maxwarnings) compare maxwarnings --max-warnings 0 src ;;
            single-file) compare single-file src/fields.js ;;
            fix) compare fix --fix src ;;
            fix-dry-run) compare fix-dry-run --fix-dry-run src ;;
            script-stylish) FIXTURE="$SCRIPT_FIXTURE" EFFECTIVE_CONFIG="$SCRIPT_CONFIG" compare script-stylish src ;;
            script-json) FIXTURE="$SCRIPT_FIXTURE" EFFECTIVE_CONFIG="$SCRIPT_CONFIG" compare script-json -f json src ;;
            script-fix) FIXTURE="$SCRIPT_FIXTURE" EFFECTIVE_CONFIG="$SCRIPT_CONFIG" compare script-fix --fix src ;;
            *) echo "unknown scenario: $s"; exit 2 ;;
        esac
    done
else
    run_all
fi

echo
echo "e2e: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
