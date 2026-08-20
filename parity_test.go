package eslint

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

type linterCase struct {
	ID          string
	Code        string
	Config      map[string]any
	ECMAVersion int
	SourceType  string
}

func linterCorpus() []linterCase {
	debugger := map[string]any{"rules": map[string]any{"no-debugger": "error"}}
	dupe := map[string]any{"rules": map[string]any{"no-dupe-keys": "error"}}
	both := map[string]any{"rules": map[string]any{"no-debugger": "error", "no-dupe-keys": "error"}}
	return []linterCase{
		{ID: "debugger-basic", Code: "debugger;", Config: debugger},
		{ID: "debugger-in-fn", Code: "function f() { debugger; }", Config: debugger},
		{ID: "debugger-multiple", Code: "debugger;\ndebugger;", Config: debugger},
		{ID: "debugger-warn", Code: "debugger;", Config: map[string]any{"rules": map[string]any{"no-debugger": "warn"}}},
		{ID: "debugger-off", Code: "debugger;", Config: map[string]any{"rules": map[string]any{"no-debugger": "off"}}},
		{ID: "debugger-numeric", Code: "debugger;", Config: map[string]any{"rules": map[string]any{"no-debugger": 2}}},
		{ID: "debugger-array", Code: "debugger;", Config: map[string]any{"rules": map[string]any{"no-debugger": []any{"error"}}}},
		{ID: "debugger-in-block", Code: "if (a) { debugger; }", Config: debugger},
		{ID: "dupe-keys-basic", Code: "var o = { a: 1, a: 2 };", Config: dupe},
		{ID: "dupe-keys-string", Code: "var o = { 'b': 1, b: 2 };", Config: dupe},
		{ID: "dupe-keys-getset", Code: "var o = { get x() {}, set x(v) {} };", Config: dupe},
		{ID: "dupe-keys-getget", Code: "var o = { get x() {}, get x() {} };", Config: dupe},
		{ID: "dupe-keys-computed", Code: "var a = 'k'; var o = { [a]: 1, [a]: 2 };", Config: dupe},
		{ID: "dupe-keys-num", Code: "var o = { 1: 'a', 1: 'b' };", Config: dupe},
		{ID: "dupe-keys-nested", Code: "var o = { a: { x: 1, x: 2 }, a: 3 };", Config: dupe},
		{ID: "dupe-keys-destructure", Code: "function f({ a }) { return a; }", Config: dupe},
		{ID: "dupe-keys-method-dup", Code: "var o = { foo() {}, foo() {} };", Config: dupe},
		{ID: "both-rules", Code: "debugger;\nvar o = { a: 1, a: 2 };", Config: both},
		{ID: "clean-code", Code: "var x = 1; function f(){ return x; }", Config: both},
	}
}

func TestLinterParity(t *testing.T) {
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available; skipping oracle parity")
	}
	wd, _ := os.Getwd()
	driver := filepath.Join(wd, "oracle", "driver.js")
	if _, err := os.Stat(driver); err != nil {
		t.Skip("oracle driver not installed; skipping")
	}

	cases := linterCorpus()
	type jcase struct {
		ID          string         `json:"id"`
		Code        string         `json:"code"`
		Config      map[string]any `json:"config"`
		ECMAVersion int            `json:"ecmaVersion"`
		SourceType  string         `json:"sourceType,omitempty"`
	}
	jc := make([]jcase, 0, len(cases))
	for _, c := range cases {
		ev := c.ECMAVersion
		if ev == 0 {
			ev = 2022
		}
		jc = append(jc, jcase{ID: c.ID, Code: c.Code, Config: c.Config, ECMAVersion: ev, SourceType: c.SourceType})
	}
	corpusBytes, _ := json.Marshal(map[string]any{"cases": jc})
	tmp, err := os.CreateTemp(t.TempDir(), "corpus-*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmp.Write(corpusBytes)
	tmp.Close()

	out, runErr := exec.Command(nodeBin, driver, tmp.Name()).CombinedOutput()
	if runErr != nil {
		t.Fatalf("oracle run failed: %v\n%s", runErr, out)
	}

	var res struct {
		Cases []struct {
			ID       string           `json:"id"`
			Tree     map[string]any   `json:"tree"`
			Messages []map[string]any `json:"messages"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("unmarshal oracle: %v", err)
	}
	if len(res.Cases) != len(cases) {
		t.Fatalf("oracle returned %d cases, expected %d", len(res.Cases), len(cases))
	}

	l := &Linter{}
	failures := 0
	for i, rc := range res.Cases {
		got := l.VerifyParsed(cases[i].Code, Node(rc.Tree), cases[i].Config)
		gotBytes, _ := json.Marshal(got)
		var gotAny any
		json.Unmarshal(gotBytes, &gotAny)

		var expBytes []byte
		expBytes, _ = json.Marshal(rc.Messages)
		var expAny any
		json.Unmarshal(expBytes, &expAny)

		if !reflect.DeepEqual(gotAny, expAny) {
			failures++
			t.Errorf("CASE %s: MISMATCH\n--- oracle ---\n%s\n--- go ---\n%s", rc.ID, indentJSON(expAny), string(gotBytes))
		} else {
			t.Logf("CASE %s: OK (%s)", rc.ID, string(gotBytes))
		}
	}
	if failures > 0 {
		t.Fatalf("parity: %d/%d cases mismatched", failures, len(cases))
	}
	t.Logf("PARITY PASS: %d cases, 0 mismatches", len(cases))
}

func indentJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
