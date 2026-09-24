package eslint

import (
	"regexp"
	"sort"
	"strings"
)

// shebangPattern is upstream's /^#!([^\r\n]+)/u from lib/shared/ast-utils.js.
// The capture group excludes only CR/LF (not U+2028/U+2029), and `^` without the
// m flag anchors to the start of the file, so only a real shebang — after BOM
// stripping — is rewritten.
var shebangPattern = regexp.MustCompile("^#!([^\r\n]+)")

// linter.go — Linter.verify / verifyAndFix: the file-level pipeline.
// Parse (espree-go) → SourceCode (+ eslint-scope) → run enabled rules over one
// traversal → sort problems by position (as ESLint does) → optionally apply
// fixes in up to 10 passes.

// Linter runs rules over source text.
type Linter struct {
	rules map[string]Rule
}

// NewLinter returns a linter with the given rules registered.
func NewLinter(rules ...Rule) *Linter {
	l := &Linter{rules: map[string]Rule{}}
	l.DefineRules(rules...)
	return l
}

// DefineRule registers a rule.
func (l *Linter) DefineRule(r Rule) {
	if r.ID == "" {
		panic("eslint: DefineRule called with an empty rule ID")
	}
	if r.Create == nil {
		panic("eslint: rule " + r.ID + " has no Create function")
	}
	l.rules[r.ID] = r
}

// DefineRules registers several rules.
func (l *Linter) DefineRules(rules ...Rule) {
	for _, r := range rules {
		l.DefineRule(r)
	}
}

// Rule looks up a registered rule.
func (l *Linter) Rule(id string) (Rule, bool) {
	r, ok := l.rules[id]
	return r, ok
}

// RuleIDs returns the registered rule IDs in sorted order.
func (l *Linter) RuleIDs() []string {
	ids := make([]string, 0, len(l.rules))
	for id := range l.rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// MaxAutofixPasses is ESLint's MAX_AUTOFIX_PASSES.
const MaxAutofixPasses = 10

// Verify lints text and returns the reported problems.
func (l *Linter) Verify(text string, cfg *Config, filename string) []Message {
	messages, _ := l.verify(text, cfg, filename)
	return messages
}

// VerifyAndFix lints text and applies fixes, repeating until no fixes remain
// or 10 passes have run — the exact loop ESLint's Linter.verifyAndFix uses.
func (l *Linter) VerifyAndFix(text string, cfg *Config, filename string) *FixResult {
	currentText := text
	fixed := false
	passNumber := 0
	var messages []Message
	var fixedResult *FixResult

	for {
		passNumber++
		messages, _ = l.verify(currentText, cfg, filename)
		ptrs := messagePointers(messages)
		wasFixed, remaining, output := applyFixes(currentText, ptrs, nil)
		fixedResult = &FixResult{
			Fixed:    wasFixed,
			Output:   output,
			Messages: derefMessages(remaining),
		}
		// "stop if there are any syntax errors. 'fixedResult.output' is a empty string."
		if len(messages) == 1 && messages[0].Fatal {
			break
		}
		fixed = fixed || wasFixed
		currentText = output
		if !wasFixed || passNumber >= MaxAutofixPasses {
			break
		}
	}

	// If the last pass applied fixes, lint again so the reported messages match
	// the final text.
	if fixedResult.Fixed {
		messages, _ = l.verify(currentText, cfg, filename)
		fixedResult.Messages = messages
	}
	fixedResult.Fixed = fixed
	fixedResult.Output = currentText
	return fixedResult
}

// FixResult mirrors ESLint's verifyAndFix return value.
type FixResult struct {
	Fixed    bool
	Output   string
	Messages []Message
}

// LintResult is one file's lint outcome (the object formatters receive).
type LintResult struct {
	FilePath string
	Messages []Message
	Source   string
	Output   string
	// Fixed is true when the source was modified by --fix.
	Fixed bool

	ErrorCount          int
	WarningCount        int
	FixableErrorCount   int
	FixableWarningCount int
	// UsedDeprecatedRules lists the deprecated rules this file's config
	// enabled (eslint reports them per result).
	UsedDeprecatedRules []DeprecatedRuleInfo
}

// DeprecatedRuleInfo is one deprecated rule used by a configuration
// (eslint's DeprecatedRuleInfo).
type DeprecatedRuleInfo struct {
	RuleID     string
	ReplacedBy []string
}

// NewLintResult builds a result and computes its counts, mirroring ESLint's
// result-object construction (fixable counts include any message carrying a
// fix, regardless of whether --fix was used).
func NewLintResult(filePath string, messages []Message) *LintResult {
	r := &LintResult{FilePath: filePath, Messages: messages}
	for _, m := range messages {
		if m.Fatal || m.Severity == 2 {
			r.ErrorCount++
		} else {
			r.WarningCount++
		}
		if m.Fix != nil {
			if m.Severity == 2 {
				r.FixableErrorCount++
			} else {
				r.FixableWarningCount++
			}
		}
	}
	return r
}

// FatalErrorCount returns the number of fatal (parse) errors, which ESLint's
// result objects report separately from errorCount.
func (r *LintResult) FatalErrorCount() int {
	n := 0
	for _, m := range r.Messages {
		if m.Fatal {
			n++
		}
	}
	return n
}

// Lint verifies source text and returns a fully populated LintResult.
func (l *Linter) Lint(text string, cfg *Config, filename string) *LintResult {
	messages, _ := l.verify(text, cfg, filename)
	r := NewLintResult(filename, messages)
	r.Source = text
	return r
}

// VerifyResult is Lint on already-read text, with output set after fixing.
func (l *Linter) LintAndFix(text string, cfg *Config, filename string) *LintResult {
	fr := l.VerifyAndFix(text, cfg, filename)
	r := NewLintResult(filename, fr.Messages)
	r.Source = text
	r.Output = fr.Output
	r.Fixed = fr.Fixed
	return r
}

// listenerEntry is one bound selector of one rule instance.
type listenerEntry struct {
	typ  string
	exit bool
	fn   func(Node)
}

// missingRuleProblem builds the problem ESLint reports for a configured rule it
// has no definition for (linter.js createLintingProblem with DEFAULT_ERROR_LOC).
func missingRuleProblem(ruleID string) *Message {
	return &Message{
		RuleID:    ruleID,
		Severity:  2,
		Message:   MissingRuleMessage(ruleID),
		Line:      1,
		Column:    1,
		EndLine:   1,
		EndColumn: 2,
	}
}

// verify is the core: parse, analyze scopes, dispatch rules, sort problems.
func (l *Linter) verify(text string, cfg *Config, filename string) ([]Message, *SourceCode) {
	if cfg == nil {
		cfg = NewConfig()
	}

	hasBOM := strings.HasPrefix(text, "\uFEFF")
	if hasBOM {
		text = strings.TrimPrefix(text, "\uFEFF")
	}

	// Upstream rewrites a leading shebang into a line comment before parsing:
	//
	//   stripUnicodeBOM(text).replace(shebangPattern, (m, captured) => `//${captured}`)
	//
	// The rewrite is length-preserving, so the `#!` line never reaches the parser
	// (which would reject it as an unexpected character) and every offset, line and
	// column stays where it was. It also makes the prologue a real prologue, so
	// `#!/usr/bin/env node` + `"use strict"` is strict code — as ESLint reports it.
	// The original text is still what SourceCode is built from.
	textToParse := shebangPattern.ReplaceAllString(text, "//$1")

	globalReturn := cfg.GlobalReturn()
	pr, err := Parse(textToParse, cfg.SourceType(), globalReturn)
	if err != nil {
		pe := normalizeParseError(err)
		return []Message{*fatalMessage(err, "Parsing error: "+pe.Message,
			pe.Line, ParseErrorColumn(text, pe.Line, pe.Column))}, nil
	}

	sc := NewSourceCode(text, pr.AST, pr.Tokens, pr.Comments, hasBOM)
	events := collectEvents(pr.AST)
	sm := analyzeScope(pr.AST, cfg.SourceType(), cfg.ECMAVersion(), globalReturn)
	applyConfiguredGlobals(sm, cfg)
	sc.SetScopeManager(sm)

	var problems []*Message
	record := func(m *Message) { problems = append(problems, m) }

	type run struct {
		ctx     *Context
		entries []listenerEntry
	}

	var runs []run
	for _, er := range cfg.EnabledRules(l.rules) {
		if !er.Known {
			// eslint's runRules: an unknown rule contributes a problem rather
			// than being ignored (severity 2, at 1:1..1:2, nodeType null).
			problems = append(problems, missingRuleProblem(er.ID))
			continue
		}
		rule := l.rules[er.ID]
		ctx := &Context{
			ID:               er.ID,
			Severity:         er.Severity,
			Options:          er.Options,
			SourceCode:       sc,
			Filename:         filename,
			PhysicalFilename: filename,
			Settings:         cfg.Settings,
			ParserOptions:    cfg.ParserOptions,
			meta:             rule.Meta,
			collect:          record,
		}
		handlers := rule.Create(ctx)
		entries := make([]listenerEntry, 0, len(handlers))
		for sel, fn := range handlers {
			typ := strings.TrimSuffix(sel, ":exit")
			entries = append(entries, listenerEntry{
				typ:  typ,
				exit: strings.HasSuffix(sel, ":exit"),
				fn:   fn,
			})
		}
		// Deterministic order within a rule (Go map iteration is random).
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].typ != entries[j].typ {
				return entries[i].typ < entries[j].typ
			}
			return !entries[i].exit && entries[j].exit
		})
		runs = append(runs, run{ctx: ctx, entries: entries})
	}

	for _, ev := range events {
		typ := NodeType(ev.node)
		for i := range runs {
			r := &runs[i]
			r.ctx.currentNode = ev.node
			for _, e := range r.entries {
				if e.typ == typ && e.exit == !ev.entering {
					e.fn(ev.node)
				}
			}
		}
	}

	// ESLint sorts problems by position (stable, so report order breaks ties).
	sort.SliceStable(problems, func(i, j int) bool {
		if problems[i].Line != problems[j].Line {
			return problems[i].Line < problems[j].Line
		}
		return problems[i].Column < problems[j].Column
	})
	return derefMessages(problems), sc
}

func messagePointers(messages []Message) []*Message {
	out := make([]*Message, len(messages))
	for i := range messages {
		out[i] = &messages[i]
	}
	return out
}

func derefMessages(msgs []*Message) []Message {
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, *m)
	}
	return out
}
