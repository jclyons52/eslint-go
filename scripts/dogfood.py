#!/usr/bin/env python3
"""Differential dogfooding: lint real npm code with eslint-go and real ESLint.

The rule corpora are hand-written, so they cannot contain the shapes shipped
packages actually use. This harness points both CLIs at the JavaScript in
oracle/node_modules (real, unmodified packages), with a config enabling every
implemented rule, and compares the JSON output file by file.

Findings are attributed, not just counted:
  crash           the port died on a file the oracle handled
  nested-config   the file sits under a package that ships its own .eslintrc*,
                  which ESLint cascades into and this CLI does not (documented)
  inline-directive the file carries /* eslint ... */, /* globals */ (documented)
  divergent       anything else: a genuine bug until proven otherwise

Usage: python3 scripts/dogfood.py [--limit N] [--quiet]
"""
import argparse
import collections
import json
import pathlib
import re
import subprocess
import sys
import tempfile

ROOT = pathlib.Path(__file__).resolve().parent.parent
BIN = ""  # set in main(): the freshly built CLI
ORACLE_PKGS = ROOT / "oracle" / "node_modules"
REAL_ESLINT = ORACLE_PKGS / "eslint" / "bin" / "eslint.js"
BATCH = 50
FIELDS = ("ruleId", "severity", "message", "line", "column", "endLine", "endColumn",
          "messageId", "fatal", "nodeType")
DIRECTIVE = re.compile(r"/\*\s*eslint|//\s*eslint|/\*\s*globals?\b")


def ported_rules():
    out = subprocess.run([BIN, "--list-rules"], capture_output=True, text=True).stdout
    return [line.split()[0] for line in out.splitlines()[1:] if line.strip()]


def corpus(limit):
    files = []
    for p in sorted(ORACLE_PKGS.rglob("*.js")):
        rel = p.relative_to(ORACLE_PKGS)
        if "node_modules" in rel.parts[1:]:  # skip nested copies
            continue
        try:
            size = p.stat().st_size
        except OSError:
            continue
        if 0 < size <= 400_000:
            files.append(str(p))
    return files[:limit] if limit else files


def run(cmd, config, files):
    return subprocess.run(cmd + ["--no-ignore", "-f", "json", "-c", str(config)] + files,
                          capture_output=True, text=True, timeout=600)


def as_json(stdout):
    try:
        data = json.loads(stdout)
    except Exception:
        return None
    return {str(pathlib.Path(e["filePath"]).resolve()): e for e in data}


def nested_config(path):
    for d in [path.parent, *path.parents]:
        if ORACLE_PKGS not in d.parents and d != ORACLE_PKGS:
            break
        for c in d.glob(".eslintrc*"):
            if c.is_file():
                return True
    return False


def main():
    global BIN
    ap = argparse.ArgumentParser()
    ap.add_argument("--limit", type=int, default=0, help="only lint the first N files")
    ap.add_argument("--quiet", action="store_true")
    args = ap.parse_args()

    if not REAL_ESLINT.exists():
        print("SKIP: the npm oracle is not installed (run npm install in oracle/)")
        return 0

    work = pathlib.Path(tempfile.mkdtemp(prefix="dogfood-"))
    BIN = str(work / "eslint-go")
    build = subprocess.run(["go", "build", "-o", BIN, "./cmd/eslint-go"], cwd=ROOT,
                           capture_output=True, text=True)
    if build.returncode != 0:
        print(build.stderr)
        return 1

    rules = ported_rules()
    config = work / ".eslintrc.json"
    config.write_text(json.dumps({
        "root": True,
        "parserOptions": {"ecmaVersion": 2022, "sourceType": "script"},
        "env": {"node": True, "es2022": True, "browser": True},
        "rules": {r: "error" for r in rules},
    }, indent=2))
    files = corpus(args.limit)
    print(f"config: {len(rules)} implemented rules | corpus: {len(files)} real npm files")

    stats = collections.Counter(files=len(files))
    findings = []
    for i in range(0, len(files), BATCH):
        batch = files[i:i + BATCH]
        real, ours = run(["node", str(REAL_ESLINT)], config, batch), run([BIN], config, batch)
        rp, op = as_json(real.stdout), as_json(ours.stdout)
        if op is None:
            # A batch that is not JSON means the port died: bisect to the file.
            for f in batch:
                one = run([BIN], config, [f])
                if as_json(one.stdout) is None:
                    findings.append({"kind": "crash", "file": f,
                                     "reason": next((l for l in one.stderr.splitlines() if l.strip()), "")[:200]})
                    stats["crash"] += 1
            continue
        if rp is None:
            print("note: the real CLI did not produce JSON for a batch; skipping it")
            continue
        for path, ren in rp.items():
            oen = op.get(path)
            if oen is None:
                continue
            stats["compared"] += 1
            rm = [{f: m.get(f) for f in FIELDS} for m in ren.get("messages", [])]
            om = [{f: m.get(f) for f in FIELDS} for m in oen.get("messages", [])]
            if rm == om:
                stats["identical"] += 1
                continue
            stats["divergent"] += 1
            p = pathlib.Path(path)
            if nested_config(p):
                stats["nested-config"] += 1
            elif DIRECTIVE.search(p.read_text(errors="replace")):
                stats["inline-directive"] += 1
            else:
                stats["GENUINE"] += 1
                findings.append({"kind": "divergent", "file": path,
                                 "only_real": [m for m in rm if m not in om][:5],
                                 "only_go": [m for m in om if m not in rm][:5]})

    identical = stats["identical"]
    compared = stats["compared"] or 1
    print(f"\nidentical : {identical}/{compared} ({100 * identical / compared:.1f}%)")
    print(f"crashes   : {stats['crash']}")
    print(f"divergent : {stats['divergent']}  "
          f"[nested-config {stats['nested-config']}, inline-directive {stats['inline-directive']}, "
          f"GENUINE {stats['GENUINE']}]")
    if not args.quiet and findings:
        print("\nfindings needing investigation:")
        for f in findings[:15]:
            if f["kind"] == "crash":
                print(f"  CRASH {pathlib.Path(f['file']).name}: {f['reason']}")
            else:
                print(f"  {pathlib.Path(f['file']).name}: "
                      f"real-only {[(m['ruleId'], m['line'], m['column']) for m in f['only_real']]} "
                      f"go-only {[(m['ruleId'], m['line'], m['column']) for m in f['only_go']]}")
    return 1 if (stats["crash"] or stats["GENUINE"]) else 0


if __name__ == "__main__":
    sys.exit(main())
