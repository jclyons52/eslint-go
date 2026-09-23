package eslint

import (
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// globals.go — configured globals and environments. ESLint augments the global
// scope after analysis with the names from `globals` (and from the environments
// enabled by `env`), then re-resolves the references that were left in
// globalScope.through. Without this step every reference to a configured global
// would be reported by no-undef.
//
// Ported from linter.js's addDeclaredGlobals.

// applyConfiguredGlobals defines configured globals in the global scope and
// re-resolves the references that name them.
func applyConfiguredGlobals(sm *eslintscope.ScopeManager, cfg *Config) {
	if sm == nil || len(sm.Scopes) == 0 {
		return
	}
	globalScope := sm.Scopes[0]

	names := map[string]string{} // name → "writable" | "readonly"
	for env := range cfg.Env {
		for name, value := range EnvGlobals(env) {
			names[name] = value
		}
	}
	for name, writable := range cfg.Globals {
		value := "readonly"
		if writable {
			value = "writable"
		}
		names[name] = value
	}

	for name, value := range names {
		if value == "off" {
			continue
		}
		variable := globalScope.Set[name]
		if variable == nil {
			variable = &eslintscope.Variable{Name: name, Scope: globalScope}
			globalScope.Variables = append(globalScope.Variables, variable)
			globalScope.Set[name] = variable
		}
		variable.Writeable = value == "writable"
	}

	// Re-resolve references that the configuration just defined.
	remaining := globalScope.Through[:0]
	for _, ref := range globalScope.Through {
		name := GetString(ref.Identifier, "name")
		variable := globalScope.Set[name]
		if variable != nil {
			ref.Resolved = variable
			variable.References = append(variable.References, ref)
			continue
		}
		remaining = append(remaining, ref)
	}
	globalScope.Through = remaining
}
