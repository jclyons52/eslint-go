package eslint

import "strings"

// Linter reproduces the observable behaviour of ESLint's Linter.verify for
// the supported rule set. Parser integration (espree-go) is a separate
// concern: VerifyParsed takes an already-parsed ESTree AST + text, which is
// also exactly what the JS-oracle test feeds it.
type Linter struct{}

// VerifyParsed runs the enabled rules from config over the given AST, in the
// same traversal + report order as ESLint, returning the messages.
func (l *Linter) VerifyParsed(code string, ast Node, config map[string]any) []Message {
	sc := newSourceCode(code, ast)
	rules := resolveRules(config)

	// Build per-rule contexts + per-type dispatch.
	type listener struct {
		typ  string
		exit bool
		fn   func(Node)
	}
	var listeners []listener

	// One shared collector backed by the messages slice, so reports append
	// in ESLint's report order across all enabled rules.
	messages := []Message{}
	record := func(m *Message) {
		messages = append(messages, *m)
	}

	for _, r := range rules {
		ctx := &RuleContext{
			ID:         r.RuleID,
			Severity:   r.Severity,
			Options:    r.Options,
			SourceCode: sc,
			messages:   ruleRegistry[r.RuleID].messages,
			collect:    record,
		}
		handlers := ruleRegistry[r.RuleID].create(ctx)
		for sel, fn := range handlers {
			typ := strings.TrimSuffix(sel, ":exit")
			exit := strings.HasSuffix(sel, ":exit")
			listeners = append(listeners, listener{typ: typ, exit: exit, fn: fn})
		}
	}

	// Traverse and dispatch.
	events := collectEvents(ast)
	for _, ev := range events {
		typ := nodeType(ev.node)
		for _, ln := range listeners {
			if ln.typ != typ || ln.exit != !ev.isEntering {
				continue
			}
			ln.fn(ev.node)
		}
	}
	return messages
}

// enabledRule is a resolved, enabled rule with its severity and options.
type enabledRule struct {
	RuleID   string
	Severity int
	Options  []any
}

// resolveRules parses config["rules"] and returns the enabled rules in the
// configured rule order that we implement.
func resolveRules(config map[string]any) []enabledRule {
	var rules map[string]any
	if config != nil {
		if r, ok := config["rules"].(map[string]any); ok {
			rules = r
		}
	}
	var out []enabledRule
	for id, v := range rules {
		m, ok := ruleRegistry[id]
		if !ok {
			continue
		}
		sev, opts, enabled := parseRuleConfig(v)
		if !enabled || sev == 0 {
			continue
		}
		_ = m
		out = append(out, enabledRule{RuleID: id, Severity: sev, Options: opts})
	}
	return out
}

// parseRuleConfig mirrors ESLint's legacy severity + options normalization.
func parseRuleConfig(v any) (severity int, options []any, enabled bool) {
	severity, options, enabled = 0, nil, true
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
		return int(t), nil, t != 0
	case bool:
		if t {
			return 2, nil, true
		}
		return 0, nil, false
	case []any:
		if len(t) == 0 {
			return 0, nil, false
		}
		sev, _, _ := parseRuleConfig(t[0])
		return sev, t[1:], sev != 0
	default:
		return 0, nil, false
	}
}
