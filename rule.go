package eslint

import (
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// rule.go — the rule authoring API. A rule is a Meta description plus a
// Create function returning a visitor map keyed by node type (exactly the
// shape ESLint rules return from create(), including ":exit" suffixes).

// Rule is a Go port of an ESLint rule.
type Rule struct {
	// ID is the rule id as it appears in configuration (e.g. "no-debugger").
	ID string
	// Meta is the rule's meta block (type, docs, messages, fixable, schema).
	Meta RuleMeta
	// Create receives the per-file context and returns the visitor map keyed
	// by node type, with ":exit" suffixes for leave handlers (exactly the
	// object ESLint rules return from create()).
	Create func(ctx *Context) map[string]func(Node)
}

// RuleMeta mirrors ESLint's rule meta block.
type RuleMeta struct {
	// Type is "problem", "suggestion" or "layout".
	Type string
	// Docs is the documentation block.
	Docs RuleDocs
	// Fixable is "", "code" or "whitespace".
	Fixable string
	// Messages holds the messageId → template map rules report against.
	Messages map[string]string
	// Schema is the rule's JSON-schema options declaration (unused at runtime;
	// carried for the registry/CLI help and for auditing against the original).
	Schema []any
	// Deprecated marks a rule ESLint reports as deprecated.
	Deprecated bool
}

// RuleDocs mirrors meta.docs.
type RuleDocs struct {
	Description string
	Recommended bool
	URL         string
}

// Context is the rule context: the object ESLint passes to create().
type Context struct {
	// ID is the rule id (e.g. "no-debugger").
	ID string
	// Severity is the resolved severity (1 = warn, 2 = error).
	Severity int
	// Options is the rule's options array (config[1:]).
	Options []any
	// SourceCode is the text/AST/token/scope facade.
	SourceCode *SourceCode
	// Filename is the path being linted ("" when linting a string).
	Filename string
	// PhysicalFilename is the path on disk (same as Filename for our purposes).
	PhysicalFilename string
	// Settings is config.settings.
	Settings map[string]any
	// ParserOptions is the resolved parserOptions (rules read sourceType etc.).
	ParserOptions map[string]any

	meta        RuleMeta
	collect     func(*Message)
	currentNode Node
}

// Report files a problem, translating the descriptor exactly as ESLint's
// report-translator does (messageId lookup, {{data}} interpolation,
// loc normalisation, fix capture).
func (c *Context) Report(d Report) {
	msg := ""
	if d.MessageID != "" {
		msg = c.meta.Messages[d.MessageID]
	} else if d.Message != "" {
		msg = d.Message
	}
	msg = interpolate(msg, d.Data)
	loc := normalizeReportLoc(d)
	var fix *Fix
	if d.Fix != nil {
		fix = d.Fix(&Fixer{sc: c.SourceCode})
	}
	problem := createProblem(c.ID, c.Severity, d.Node, msg, d.MessageID, loc, fix, c.SourceCode)
	c.collect(problem)
}

// Option returns options[i] (nil when absent).
func (c *Context) Option(i int) any {
	if i < 0 || i >= len(c.Options) {
		return nil
	}
	return c.Options[i]
}

// OptionString returns options[i] as a string, or def.
func (c *Context) OptionString(i int, def string) string {
	if s, ok := c.Option(i).(string); ok {
		return s
	}
	return def
}

// OptionBool returns options[i] as a bool, or def.
func (c *Context) OptionBool(i int, def bool) bool {
	if b, ok := c.Option(i).(bool); ok {
		return b
	}
	return def
}

// OptionInt returns options[i] as an int, or def.
func (c *Context) OptionInt(i int, def int) int {
	switch v := c.Option(i).(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return def
}

// OptionMap returns options[i] as an object, or nil.
func (c *Context) OptionMap(i int) map[string]any {
	m, _ := c.Option(i).(map[string]any)
	return m
}

// OptionMapValue returns options[i][key], or def.
func (c *Context) OptionMapValue(i int, key string, def any) any {
	m := c.OptionMap(i)
	if m == nil {
		return def
	}
	if v, ok := m[key]; ok {
		return v
	}
	return def
}

// GetScope returns the scope of the node currently being visited
// (context.getScope()).
func (c *Context) GetScope() *eslintscope.Scope {
	return c.SourceCode.Scope(c.currentNode)
}

// ScopeOf returns the scope for an arbitrary node (the ESLint 9
// sourceCode.getScope(node) form, also useful in :exit handlers).
func (c *Context) ScopeOf(node Node) *eslintscope.Scope {
	return c.SourceCode.Scope(node)
}

// Ancestors returns the ancestor chain of the current node, root first
// (context.getAncestors()).
func (c *Context) Ancestors() []Node { return AncestorsOf(c.currentNode) }

// DeclaredVariables returns the variables declared by a node
// (context.getDeclaredVariables(node)).
func (c *Context) DeclaredVariables(node Node) []*eslintscope.Variable {
	return c.SourceCode.DeclaredVariables(node)
}

// MarkVariableAsUsed marks a variable as used in the current scope
// (context.markVariableAsUsed(name)).
func (c *Context) MarkVariableAsUsed(name string) bool {
	return c.SourceCode.MarkVariableAsUsed(name, c.currentNode)
}

// GetSourceCode is the ESLint method name for the SourceCode field.
func (c *Context) GetSourceCode() *SourceCode { return c.SourceCode }

// GetFilename returns the filename being linted.
func (c *Context) GetFilename() string { return c.Filename }

// RuleFunc is a visitor map keyed by node type / "Type:exit".
type RuleFunc = func(Node)
