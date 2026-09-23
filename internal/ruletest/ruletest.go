// Package ruletest is the rule parity harness: it runs real ESLint (the
// eslint@8.57.0 install under oracle/) over a case list and compares the
// resulting LintMessages — and, for fix cases, the fixed output — with the Go
// port's, exactly.
//
// Every rule package in rules/ uses this: a rule is only "ported" when its
// parity test reports zero mismatches against the JS oracle.
package ruletest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	eslint "github.com/jclyons52/eslint-go"
)

// Case is one lint case handed to both implementations.
type Case struct {
	// ID names the case in failure output ("no-debugger-basic").
	ID string
	// Code is the source text.
	Code string
	// Options are the rule's options (config value becomes [2, ...Options]).
	Options []any
	// Config, when set, replaces the generated config object entirely (use for
	// globals/extra rules).
	Config map[string]any
	// SourceType defaults to "module" (acorn-go parses modules only).
	SourceType string
	// ECMAVersion defaults to 2022.
	ECMAVersion int
	// Fix makes the comparison use verifyAndFix: the Go port's fixed output and
	// final messages must both match the oracle's.
	Fix bool
}

type oracleCase struct {
	ID          string         `json:"id"`
	Code        string         `json:"code"`
	Config      map[string]any `json:"config"`
	ECMAVersion int            `json:"ecmaVersion"`
	SourceType  string         `json:"sourceType"`
	Fix         bool           `json:"fix"`
}

type oracleCaseResult struct {
	ID       string           `json:"id"`
	Messages []map[string]any `json:"messages"`
	Output   *string          `json:"output"`
	Fixed    *bool            `json:"fixed"`
}

type oracleOutput struct {
	Cases []oracleCaseResult `json:"cases"`
}

// Options configures the harness.
type Options struct {
	// RuleOptionsPosition is where rule options start in the config value
	// (default 1: [severity, opts...]).
	Severity any
	// ExtraConfig merges additional config keys (parserOptions, globals, ...)
	// into every generated case config.
	ExtraConfig map[string]any
}

// Compare lints every case with real ESLint and with linter, comparing messages
// (and fixed output for Fix cases). It logs one line per case and fails the test
// on any mismatch.
func Compare(t *testing.T, rule eslint.Rule, cases []Case) {
	t.Helper()
	CompareWith(t, rule, cases, Options{})
}

// CompareWith is Compare with harness options.
func CompareWith(t *testing.T, rule eslint.Rule, cases []Case, opts Options) {
	t.Helper()
	if len(cases) == 0 {
		t.Fatal("ruletest: no cases supplied")
	}

	severity := opts.Severity
	if severity == nil {
		severity = 2
	}

	payload := struct {
		Cases []oracleCase `json:"cases"`
	}{}
	for _, c := range cases {
		cfg := c.Config
		if cfg == nil {
			value := []any{severity}
			value = append(value, c.Options...)
			cfg = map[string]any{"rules": map[string]any{rule.ID: value}}
		} else if _, hasRules := cfg["rules"]; !hasRules {
			value := []any{severity}
			value = append(value, c.Options...)
			cfg["rules"] = map[string]any{rule.ID: value}
		}
		cfg = mergeConfig(cfg, opts.ExtraConfig)
		ev := c.ECMAVersion
		if ev == 0 {
			ev = 2022
		}
		st := c.SourceType
		if st == "" {
			st = "module"
		}
		payload.Cases = append(payload.Cases, oracleCase{
			ID: c.ID, Code: c.Code, Config: cfg, ECMAVersion: ev, SourceType: st, Fix: c.Fix,
		})
	}

	oracle := runOracle(t, payload)
	if len(oracle.Cases) != len(cases) {
		t.Fatalf("ruletest: oracle returned %d cases, want %d", len(oracle.Cases), len(cases))
	}
	if lc := len(oracle.Cases); lc > 0 && !hasCaseIDs(oracle) {
		t.Log("ruletest: oracle output had no per-case ids; matching by index")
	}

	linter := eslint.NewLinter(rule)
	mismatches := 0
	for i, oc := range oracle.Cases {
		c := cases[i]

		// Expected: the oracle messages as-is (ruleId is null for fatal errors).
		var want any
		b, _ := json.Marshal(oc.Messages)
		_ = json.Unmarshal(b, &want)

		cfg := configFromCase(rule, c, severity, opts)
		var got []eslint.Message
		var output string
		if c.Fix {
			result := linter.VerifyAndFix(c.Code, cfg, c.fileName())
			got = result.Messages
			output = result.Output
		} else {
			got = linter.Verify(c.Code, cfg, c.fileName())
			output = c.Code
		}
		gb, _ := json.Marshal(got)
		var gotAny any
		_ = json.Unmarshal(gb, &gotAny)

		if !reflect.DeepEqual(gotAny, want) {
			mismatches++
			t.Errorf("CASE %s: MESSAGE MISMATCH\n  oracle: %s\n  go    : %s", c.ID, pretty(want), string(gb))
			continue
		}
		if c.Fix {
			if oc.Output != nil && *oc.Output != output {
				mismatches++
				t.Errorf("CASE %s: FIX OUTPUT MISMATCH\n  oracle: %q\n  go    : %q", c.ID, *oc.Output, output)
				continue
			}
		}
		t.Logf("CASE %s: OK", c.ID)
	}
	if mismatches > 0 {
		t.Fatalf("PARITY FAIL: rule %s — %d/%d cases mismatched", rule.ID, mismatches, len(cases))
	}
	t.Logf("PARITY PASS: rule %s — %d cases, 0 mismatches", rule.ID, len(cases))
}

// CompareCorpus is the raw form: cases plus pre-built configs, no rule wrapper.
// Use it when a file-level check needs several rules at once.
func CompareCorpus(t *testing.T, rules []eslint.Rule, cases []Case) {
	t.Helper()
	payload := struct {
		Cases []oracleCase `json:"cases"`
	}{}
	for _, c := range cases {
		ev := c.ECMAVersion
		if ev == 0 {
			ev = 2022
		}
		st := c.SourceType
		if st == "" {
			st = "module"
		}
		payload.Cases = append(payload.Cases, oracleCase{
			ID: c.ID, Code: c.Code, Config: c.Config, ECMAVersion: ev, SourceType: st, Fix: c.Fix,
		})
	}
	oracle := runOracle(t, payload)
	linter := eslint.NewLinter(rules...)
	mismatches := 0
	for i, oc := range oracle.Cases {
		c := cases[i]
		var want any
		b, _ := json.Marshal(oc.Messages)
		_ = json.Unmarshal(b, &want)
		cfg, err := eslint.ParseConfigJSON(mustJSON(t, c.Config))
		if err != nil {
			t.Fatalf("case %s: bad config: %v", c.ID, err)
		}
		got := linter.Verify(c.Code, cfg, c.fileName())
		gb, _ := json.Marshal(got)
		var gotAny any
		_ = json.Unmarshal(gb, &gotAny)
		if !reflect.DeepEqual(gotAny, want) {
			mismatches++
			t.Errorf("CASE %s: MISMATCH\n  oracle: %s\n  go    : %s", c.ID, pretty(want), string(gb))
			continue
		}
		t.Logf("CASE %s: OK", c.ID)
	}
	if mismatches > 0 {
		t.Fatalf("PARITY FAIL: %d/%d cases mismatched", mismatches, len(cases))
	}
	t.Logf("PARITY PASS: %d cases, 0 mismatches", len(cases))
}

func (c Case) fileName() string {
	if c.ID == "" {
		return "case.js"
	}
	return c.ID + ".js"
}

func configFromCase(rule eslint.Rule, c Case, severity any, opts Options) *eslint.Config {
	cfg := c.Config
	if cfg == nil {
		value := []any{severity}
		value = append(value, c.Options...)
		cfg = map[string]any{"rules": map[string]any{rule.ID: value}}
	}
	cfg = mergeConfig(cfg, opts.ExtraConfig)
	parsed, err := eslint.ParseConfigJSON(mustJSONConfig(cfg))
	if err != nil {
		panic("ruletest: invalid generated config: " + err.Error())
	}
	return parsed
}

func mergeConfig(cfg, extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range cfg {
		out[k] = v
	}
	for k, v := range extra {
		if k == "rules" {
			merged := map[string]any{}
			if existing, ok := out["rules"].(map[string]any); ok {
				for rk, rv := range existing {
					merged[rk] = rv
				}
			}
			for rk, rv := range v.(map[string]any) {
				merged[rk] = rv
			}
			out["rules"] = merged
			continue
		}
		out[k] = v
	}
	return out
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func mustJSONConfig(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func hasCaseIDs(o oracleOutput) bool {
	for _, c := range o.Cases {
		if c.ID == "" {
			return false
		}
	}
	return true
}

// runOracle shells out to the real ESLint via oracle/lint_driver.js.
func runOracle(t *testing.T, payload any) oracleOutput {
	t.Helper()
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available; skipping JS-oracle parity")
	}
	oracleDir := findOracleDir(t)
	driver := filepath.Join(oracleDir, "lint_driver.js")
	if _, err := os.Stat(driver); err != nil {
		t.Skipf("oracle driver missing (%s); skipping parity", driver)
	}
	if _, err := os.Stat(filepath.Join(oracleDir, "node_modules", "eslint")); err != nil {
		t.Skip("eslint oracle deps not installed (run npm install in oracle/); skipping parity")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal corpus: %v", err)
	}
	tmp, err := os.CreateTemp(t.TempDir(), "corpus-*.json")
	if err != nil {
		t.Fatalf("temp corpus: %v", err)
	}
	if _, err := tmp.Write(body); err != nil {
		t.Fatalf("write corpus: %v", err)
	}
	tmp.Close()

	cmd := exec.Command(nodeBin, driver, tmp.Name())
	cmd.Dir = oracleDir
	cmd.Env = append(os.Environ(),
		"ESLINT_ORACLE="+filepath.Join(oracleDir, "node_modules", "eslint"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("oracle run failed: %v\n%s", err, out)
	}
	var res oracleOutput
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("unmarshal oracle output: %v\n%s", err, out)
	}
	return res
}

// findOracleDir locates the module's oracle/ directory by walking up from the
// test's working directory to go.mod.
func findOracleDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "oracle")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("ruletest: could not locate module root (go.mod) from " + wd)
		}
		dir = parent
	}
}

func pretty(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "<unprintable>"
	}
	return string(b)
}
