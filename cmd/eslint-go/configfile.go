package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// configfile.go — config loading for the CLI: an eslintrc-style JSON file per
// directory (cascading to the root, exactly as ESLint 8 resolves them), with
// `extends: "eslint:recommended"` and relative-path extends.
//
// Unsupported config keys (plugins, shareable configs from npm, .js configs)
// are reported as errors rather than ignored, so a config that would behave
// differently never silently lints with the wrong rules.

// configFileNames are the eslintrc file names the CLI looks for, in order.
var configFileNames = []string{".eslintrc.json", ".eslintrc"}

// loadConfigForFile resolves the effective config for a file: every
// .eslintrc.json from the file's directory upwards, merged root-first (closer
// files win), stopping at a config with root:true or at the filesystem root.
func loadConfigForFile(file string, rules map[string]eslint.Rule) (*eslint.Config, error) {
	dir := filepath.Dir(file)
	var chain []string
	for {
		for _, name := range configFileNames {
			candidate := filepath.Join(dir, name)
			if _, err := os.Stat(candidate); err == nil {
				chain = append(chain, candidate)
				break
			}
		}
		if len(chain) > 0 {
			// Stop once a config declares itself the root.
			if isRootConfig(chain[len(chain)-1]) {
				break
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	merged := eslint.NewConfig()
	// Apply from the outermost config inwards so nearer files win.
	for i := len(chain) - 1; i >= 0; i-- {
		cfg, err := loadConfigFile(chain[i], rules)
		if err != nil {
			return nil, err
		}
		mergeConfig(merged, cfg)
	}
	return merged, nil
}

func isRootConfig(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var raw struct {
		Root bool `json:"root"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	return raw.Root
}

// loadConfigFile reads one config file, resolving `extends`.
func loadConfigFile(path string, rules map[string]eslint.Rule) (*eslint.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := eslint.ParseConfigJSON(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	var raw struct {
		Extends []string `json:"extends"`
	}
	if err := json.Unmarshal(data, &raw); err == nil {
		for _, ext := range raw.Extends {
			base, err := resolveExtends(ext, path, rules)
			if err != nil {
				return nil, err
			}
			if base != nil {
				inherited := eslint.NewConfig()
				mergeConfig(inherited, base)
				mergeConfig(inherited, cfg) // the file's own settings win
				cfg = inherited
			}
		}
	}
	return cfg, nil
}

// resolveExtends handles "eslint:recommended" and relative-path configs.
//
// `eslint:recommended` here enables the recommended rules this port implements
// (see rules.All); without this filter every unported rule would produce a
// "Definition for rule … was not found." problem — accurate to ESLint's
// behaviour for a missing rule, but not what the user asked for.
func resolveExtends(ext, fromPath string, rules map[string]eslint.Rule) (*eslint.Config, error) {
	switch {
	case ext == "eslint:recommended":
		cfg := eslint.NewConfig()
		var order []string
		for id, sev := range eslint.RecommendedRules {
			if _, ok := rules[id]; !ok {
				continue
			}
			cfg.Rules[id] = sev
			order = append(order, id)
		}
		sortStrings(order)
		cfg.RuleOrder = order
		return cfg, nil
	case strings.HasPrefix(ext, "./") || strings.HasPrefix(ext, "../"):
		base := filepath.Join(filepath.Dir(fromPath), ext)
		if _, err := os.Stat(base); err != nil {
			base += ".json"
			if _, err2 := os.Stat(base); err2 != nil {
				return nil, fmt.Errorf("%s: cannot resolve extends %q", fromPath, ext)
			}
		}
		return loadConfigFile(base, rules)
	case ext == "eslint:all":
		return nil, fmt.Errorf("%s: extends %q is not supported (the port implements a subset of the built-in rules)", fromPath, ext)
	default:
		return nil, fmt.Errorf("%s: extends %q is not supported (only \"eslint:recommended\" and relative paths)", fromPath, ext)
	}
}

// mergeConfig merges src into dst: rules and globals accumulate (src wins),
// parserOptions/settings merge key by key, env unions.
//
// Rule order matters (it decides report order for rules firing on the same node
// and the order of eslint's usedDeprecatedRules), so it is preserved: rules
// already staged keep their position, src's own configured order follows, and
// anything left over is appended in sorted order rather than in Go's random map
// order.
func mergeConfig(dst, src *eslint.Config) {
	if dst.Rules == nil {
		dst.Rules = map[string]any{}
	}
	staged := map[string]bool{}
	for _, id := range dst.RuleOrder {
		staged[id] = true
	}
	ordered := append([]string{}, src.RuleOrder...)
	orderedSeen := map[string]bool{}
	for _, id := range ordered {
		orderedSeen[id] = true
	}
	rest := make([]string, 0, len(src.Rules))
	for id := range src.Rules {
		if !orderedSeen[id] {
			rest = append(rest, id)
		}
	}
	sortStrings(rest)
	ordered = append(ordered, rest...)

	for id, v := range src.Rules {
		dst.Rules[id] = v
	}
	for _, id := range ordered {
		if _, exists := src.Rules[id]; !exists {
			continue
		}
		if !staged[id] {
			dst.RuleOrder = append(dst.RuleOrder, id)
			staged[id] = true
		}
	}
	if dst.ParserOptions == nil {
		dst.ParserOptions = map[string]any{}
	}
	for k, v := range src.ParserOptions {
		dst.ParserOptions[k] = v
	}
	if dst.Globals == nil {
		dst.Globals = map[string]bool{}
	}
	for k, v := range src.Globals {
		dst.Globals[k] = v
	}
	if dst.Settings == nil {
		dst.Settings = map[string]any{}
	}
	for k, v := range src.Settings {
		dst.Settings[k] = v
	}
	if dst.Env == nil {
		dst.Env = map[string]bool{}
	}
	for k, v := range src.Env {
		dst.Env[k] = v
	}
	if src.NoInlineConfig {
		dst.NoInlineConfig = true
	}
	if src.ReportUnusedDisableDirectives {
		dst.ReportUnusedDisableDirectives = true
	}
}

// ignorePatternsFromConfig reads ignorePatterns out of a config file chain.
func ignorePatternsFromConfig(file string) []string {
	var out []string
	dir := filepath.Dir(file)
	for {
		for _, name := range configFileNames {
			path := filepath.Join(dir, name)
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var raw struct {
				IgnorePatterns []string `json:"ignorePatterns"`
			}
			if err := json.Unmarshal(data, &raw); err == nil {
				out = append(out, raw.IgnorePatterns...)
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return out
		}
		dir = parent
	}
}

// readIgnoreFile reads an .eslintignore file (one pattern per line, # comments).
func readIgnoreFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
