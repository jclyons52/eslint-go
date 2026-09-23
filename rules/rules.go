// Package rules is the registry of ported ESLint rules: it wires every verified
// rule package in this module into one list for the linter and the CLI.
//
// Each rule lives in its own package (rules/<name>/), is validated against the
// real ESLint JS oracle by its own parity test, and is registered by
// scripts/gen_rule_registry.py — which only registers rules whose parity test
// carries at least one case, so nothing runs unverified. Adding a rule means
// adding its package and re-running that script.
package rules

import (
	"sort"

	eslint "github.com/jclyons52/eslint-go"
)

// All returns every registered rule, sorted by rule id.
func All() []eslint.Rule {
	all := append([]eslint.Rule{}, registry...)
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all
}

// Map returns the registered rules keyed by rule id.
func Map() map[string]eslint.Rule {
	m := make(map[string]eslint.Rule, len(registry))
	for _, r := range registry {
		m[r.ID] = r
	}
	return m
}

// IDs returns the registered rule ids, sorted.
func IDs() []string {
	ids := make([]string, 0, len(registry))
	for _, r := range registry {
		ids = append(ids, r.ID)
	}
	sort.Strings(ids)
	return ids
}

// Unverified returns rule packages that exist but have no parity test yet, so
// they are deliberately not registered.
func Unverified() []string {
	out := append([]string{}, unverified...)
	sort.Strings(out)
	return out
}
