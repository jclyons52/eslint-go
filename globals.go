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

// applyConfiguredGlobals defines the globals a run has in the global scope and
// re-resolves the references that name them.
//
// This mirrors ESLint's layering exactly (eslintrc mode):
//
//	SourceCode.applyLanguageOptions:  built-ins for the ecmaVersion
//	                                 (+ globals.commonjs for sourceType commonjs)
//	                                 then the config's globals on top
//	linter.js resolveGlobals:         enabled environments' globals, then the
//	                                 provided globals
//
// so a configured global (or an environment's) wins over the built-in, and a
// linter that skipped the ecmaVersion layer would report Array/NaN/Promise as
// undefined.
func applyConfiguredGlobals(sm *eslintscope.ScopeManager, cfg *Config) {
	if sm == nil || len(sm.Scopes) == 0 {
		return
	}
	globalScope := sm.Scopes[0]

	names := map[string]string{} // name → "writable" | "readonly"
	// The ES5 builtin set is the base in eslintrc mode — not the per-ecmaVersion
	// table (see BuiltinGlobals).
	for name, access := range BuiltinGlobals {
		names[name] = access
	}
	if cfg.SourceType() == "commonjs" {
		for name, access := range CommonJSGlobals {
			names[name] = access
		}
	}
	for env := range cfg.Env {
		for name, access := range EnvGlobals(env) {
			names[name] = access
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

// ecmaVersionKey maps an ecmaVersion number to the key ESLint's conf/globals.js
// uses: 3 and 5 stay as they are, 6-14 become the year form (6 → es2015,
// 15 → es2024) and a year stays itself.
func ecmaVersionKey(version int) string {
	switch {
	case version == 3:
		return "es3"
	case version == 5 || version < 3:
		return "es5"
	case version < 2015:
		return "es" + itoa(version+2009)
	default:
		return "es" + itoa(version)
	}
}
