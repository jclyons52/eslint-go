#!/usr/bin/env python3
"""Write the effective e2e config: the fixture's rules ∩ the ported rules.

  filter_config.py <fixture config> <output config> <file with one rule id per line>

The end-to-end parity run must hand both tools the same config, and a config
referencing an unported rule would make ESLint emit "Definition for rule … was
not found." problems that the port also emits (which is correct behaviour, but
noise for a parity run). Filtering keeps the fixture honest: it lists what the
demo wants to exercise and this script narrows it to what is actually ported, so
the effective config grows as rules land.
"""
import json
import sys


def main():
    if len(sys.argv) != 4:
        print(__doc__.strip(), file=sys.stderr)
        return 2
    src, dst, rule_list = sys.argv[1:]
    with open(rule_list) as handle:
        available = {line.strip() for line in handle if line.strip()}

    with open(src) as handle:
        cfg = json.load(handle)

    configured = cfg.get("rules", {})
    kept = {k: v for k, v in configured.items() if k in available}
    skipped = sorted(set(configured) - set(kept))
    cfg["rules"] = kept
    with open(dst, "w") as handle:
        json.dump(cfg, handle, indent=2)
        handle.write("\n")

    print(
        f"effective config: {len(kept)} rules"
        + (f" ({len(skipped)} not ported yet: {', '.join(skipped)})" if skipped else "")
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
