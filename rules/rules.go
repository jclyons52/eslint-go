// Package rules is the registry of ported ESLint rules: it wires every rule
// package in this module into one list for the linter and the CLI.
//
// Each rule lives in its own package (rules/<name>/), is validated against the
// real ESLint JS oracle by its own parity test, and is registered here. This
// file is the only place that needs updating when a rule lands.
package rules

import (
	"sort"

	eslint "github.com/jclyons52/eslint-go"

	"github.com/jclyons52/eslint-go/rules/nodebugger"
	"github.com/jclyons52/eslint-go/rules/nodupekeys"
	"github.com/jclyons52/eslint-go/rules/notrailingspaces"
	"github.com/jclyons52/eslint-go/rules/noundef"
)

// All returns every ported rule, sorted by rule id.
func All() []eslint.Rule {
	all := []eslint.Rule{
		nodebugger.Rule,
		nodupekeys.Rule,
		notrailingspaces.Rule,
		noundef.Rule,
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all
}

// Map returns the ported rules keyed by rule id.
func Map() map[string]eslint.Rule {
	m := map[string]eslint.Rule{}
	for _, r := range All() {
		m[r.ID] = r
	}
	return m
}

// IDs returns the ported rule ids, sorted.
func IDs() []string {
	ids := make([]string, 0, len(Map()))
	for id := range Map() {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
