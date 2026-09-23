package eslint

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// config.go — eslintrc-style configuration: the subset of ESLint 8's config
// format the ported rule set and CLI need (rules, parserOptions, globals,
// settings, env). `extends`/`plugins`/`overrides` are resolved by the caller
// (the CLI flattens them) rather than here.

// Config is a flattened, resolved ESLint configuration.
type Config struct {
	// Rules maps rule id → configured value (severity string/number or
	// [severity, ...options]).
	Rules map[string]any
	// RuleOrder is the order rules appeared in the config file. ESLint
	// iterates rules in configuration order, which determines report order
	// for rules that fire on the same node — so it is preserved here.
	RuleOrder []string
	// ParserOptions is the parserOptions block.
	ParserOptions map[string]any
	// Globals maps global names to true (writable and/or explicitly allowed).
	Globals map[string]bool
	// Settings is config.settings.
	Settings map[string]any
	// Env maps environment names to true.
	Env map[string]bool
	// NoInlineConfig disables /* eslint */ inline configuration.
	NoInlineConfig bool
	// ReportUnusedDisableDirectives mirrors the same-named option.
	ReportUnusedDisableDirectives bool
}

// NewConfig returns an empty config.
func NewConfig() *Config {
	return &Config{
		Rules:         map[string]any{},
		ParserOptions: map[string]any{},
		Globals:       map[string]bool{},
		Settings:      map[string]any{},
		Env:           map[string]bool{},
	}
}

// NewConfigWithRules builds a config from rules and an explicit rule order
// (needed for deterministic report order in tests).
func NewConfigWithRules(rules map[string]any, order ...string) *Config {
	c := NewConfig()
	c.Rules = rules
	c.RuleOrder = order
	return c
}

// SourceType returns parserOptions.sourceType ("script" when unset).
func (c *Config) SourceType() string {
	if c != nil {
		if s, ok := c.ParserOptions["sourceType"].(string); ok && s != "" {
			return s
		}
	}
	return "script"
}

// ECMAVersion returns parserOptions.ecmaVersion as a number. Like ESLint, a
// missing ecmaVersion means ES5 and "latest" resolves to the version bundled
// with this ESLint generation (15 / ES2024 for eslint 8.57 + espree 9.6).
func (c *Config) ECMAVersion() int {
	if c == nil {
		return 5
	}
	switch v := c.ParserOptions["ecmaVersion"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if v == "latest" {
			return LatestECMAVersion
		}
	}
	return 5
}

// LatestECMAVersion is what "latest" resolves to: 15, i.e. ES2024, matching
// espree 9.6.1's latestEcmaVersion that eslint 8.57 uses.
const LatestECMAVersion = 15

// ECMAFeatures returns parserOptions.ecmaFeatures.
func (c *Config) ECMAFeatures() map[string]any {
	if c == nil {
		return nil
	}
	m, _ := c.ParserOptions["ecmaFeatures"].(map[string]any)
	return m
}

// EnabledRule is a configured rule that is switched on.
type EnabledRule struct {
	ID       string
	Severity int
	Options  []any
	// Known is false when the rule is configured but not registered with the
	// linter — ESLint reports "Definition for rule 'x' was not found." for
	// those, and so do we (see Linter.verify).
	Known bool
}

// EnabledRules resolves the configured rules against a rule registry, keeping
// ESLint's configuration order: rules listed in RuleOrder first (in that
// order), then any remaining rules in sorted order (a config built in Go
// without an explicit order still needs deterministic behaviour). Rules that
// are configured but not in the registry are returned with Known=false rather
// than dropped, so the linter can report them the way ESLint does.
func (c *Config) EnabledRules(registry map[string]Rule) []EnabledRule {
	var out []EnabledRule
	seen := map[string]bool{}
	add := func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		v, ok := c.Rules[id]
		if !ok {
			return
		}
		_, known := registry[id]
		sev, opts, enabled := ParseRuleConfig(v)
		if !enabled || sev == 0 {
			return
		}
		out = append(out, EnabledRule{ID: id, Severity: sev, Options: opts, Known: known})
	}
	for _, id := range c.RuleOrder {
		add(id)
	}
	ids := make([]string, 0, len(c.Rules))
	for id := range c.Rules {
		ids = append(ids, id)
	}
	sortStrings(ids)
	for _, id := range ids {
		add(id)
	}
	return out
}

// ParseRuleConfig mirrors ESLint's legacy severity/options normalization:
// "off"/"warn"/"error", 0/1/2, true/false, or [severity, ...options].
func ParseRuleConfig(v any) (severity int, options []any, enabled bool) {
	switch t := v.(type) {
	case string:
		switch t {
		case "off":
			return 0, nil, false
		case "warn":
			return 1, nil, true
		case "error":
			return 2, nil, true
		}
		return 0, nil, false
	case float64:
		return int(t), nil, int(t) != 0
	case int:
		return t, nil, t != 0
	case int64:
		return int(t), nil, int(t) != 0
	case bool:
		if t {
			return 2, nil, true
		}
		return 0, nil, false
	case []any:
		if len(t) == 0 {
			return 0, nil, false
		}
		sev, _, _ := ParseRuleConfig(t[0])
		return sev, t[1:], sev != 0
	default:
		return 0, nil, false
	}
}

// ParseConfigJSON parses an eslintrc-style JSON config, preserving rule order.
func ParseConfigJSON(data []byte) (*Config, error) {
	cfg := NewConfig()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("config must be a JSON object")
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := keyTok.(string)
		switch key {
		case "rules":
			rules, order, err := decodeOrderedObject(dec)
			if err != nil {
				return nil, err
			}
			cfg.Rules = rules
			cfg.RuleOrder = order
		case "parserOptions":
			m := map[string]any{}
			if err := dec.Decode(&m); err != nil {
				return nil, err
			}
			cfg.ParserOptions = normalizeNumbers(m)
		case "globals":
			var m map[string]any
			if err := dec.Decode(&m); err != nil {
				return nil, err
			}
			for k, v := range m {
				switch t := v.(type) {
				case bool:
					cfg.Globals[k] = t
				case string:
					cfg.Globals[k] = t == "writable" || t == "writeable" || t == "readonly" || t == "readable"
				default:
					cfg.Globals[k] = true
				}
			}
		case "settings":
			m := map[string]any{}
			if err := dec.Decode(&m); err != nil {
				return nil, err
			}
			cfg.Settings = normalizeNumbers(m)
		case "env":
			var m map[string]any
			if err := dec.Decode(&m); err != nil {
				return nil, err
			}
			for k, v := range m {
				if b, ok := v.(bool); !ok || b {
					cfg.Env[k] = true
				}
			}
		case "noInlineConfig":
			var b bool
			if err := dec.Decode(&b); err != nil {
				return nil, err
			}
			cfg.NoInlineConfig = b
		case "root", "extends", "plugins", "overrides", "ignorePatterns", "parser",
			"reportUnusedDisableDirectives":
			var skip any
			if err := dec.Decode(&skip); err != nil {
				return nil, err
			}
			if key == "reportUnusedDisableDirectives" {
				switch t := skip.(type) {
				case bool:
					cfg.ReportUnusedDisableDirectives = t
				case string:
					cfg.ReportUnusedDisableDirectives = t == "error" || t == "warn"
				}
			}
		default:
			var skip any
			if err := dec.Decode(&skip); err != nil {
				return nil, err
			}
		}
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	cfg.Rules = normalizeNumbersMap(cfg.Rules)
	return cfg, nil
}

// decodeOrderedObject decodes a JSON object preserving key order.
func decodeOrderedObject(dec *json.Decoder) (map[string]any, []string, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, nil, fmt.Errorf("expected an object")
	}
	m := map[string]any{}
	var order []string
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		key, _ := keyTok.(string)
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, nil, err
		}
		m[key] = v
		order = append(order, key)
	}
	if _, err := dec.Token(); err != nil {
		return nil, nil, err
	}
	return m, order, nil
}

// normalizeNumbers converts json.Number values (decoder.UseNumber) to numbers
// rules and severity resolution expect, recursively.
func normalizeNumbers(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return m
	}
	return normalizeNumbersMap(m)
}

func normalizeNumbersMap(m map[string]any) map[string]any {
	for k, v := range m {
		m[k] = normalizeValue(v)
	}
	return m
}

func normalizeValue(v any) any {
	switch t := v.(type) {
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return float64(i)
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	case map[string]any:
		return normalizeNumbersMap(t)
	case []any:
		for i := range t {
			t[i] = normalizeValue(t[i])
		}
		return t
	}
	return v
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
