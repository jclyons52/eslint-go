// Package nounusedvars is a port of eslint/lib/rules/no-unused-vars.js.
//
// The rule is a single Program:exit pass over the scope tree: it collects every
// variable that has no *read* reference (with a long list of carve-outs:
// parameters, catch params, rest siblings, exported bindings, write-only
// self-updates, ...) and reports the last write reference — or the first
// declaration when the variable is never written.
//
// Known core gap: the JS rule's last branch reports a `/*global x*/` comment
// directive that declares a global nobody uses. ESLint implements those inline
// global comments in the Linter (addDeclaredGlobals + getDirectiveComments),
// which this core does not port, so no variable ever carries
// `eslintExplicitGlobalComments` and that branch is unreachable here. See the
// package report for the exact oracle case.
package nounusedvars

import (
	"regexp"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "no-unused-vars"

// Rule is a port of eslint/lib/rules/no-unused-vars.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Disallow unused variables",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-unused-vars",
		},
		Messages: map[string]string{
			"unusedVar": "'{{varName}}' is {{action}} but never used{{additional}}.",
		},
		Schema: []any{
			map[string]any{
				"oneOf": []any{
					map[string]any{"enum": []any{"all", "local"}},
					map[string]any{
						"type": "object",
						"properties": map[string]any{
							"vars":                           map[string]any{"enum": []any{"all", "local"}},
							"varsIgnorePattern":              map[string]any{"type": "string"},
							"args":                           map[string]any{"enum": []any{"all", "after-used", "none"}},
							"ignoreRestSiblings":             map[string]any{"type": "boolean"},
							"argsIgnorePattern":              map[string]any{"type": "string"},
							"caughtErrors":                   map[string]any{"enum": []any{"all", "none"}},
							"caughtErrorsIgnorePattern":      map[string]any{"type": "string"},
							"destructuredArrayIgnorePattern": map[string]any{"type": "string"},
						},
						"additionalProperties": false,
					},
				},
			},
		},
	},
	Create: create,
}

// jsPattern is a user-supplied RegExp: the Go matcher plus the pattern source,
// because the rule embeds `RegExp.prototype.toString()` ("/^_/u") in the
// message suffix. The "u" flag is JS-only and has no Go equivalent.
type jsPattern struct {
	re  *regexp.Regexp
	src string
}

func newJSPattern(src string) *jsPattern {
	re, err := regexp.Compile(src)
	if err != nil {
		// Go's regexp is not JS's: a pattern it cannot compile (lookbehind,
		// backreferences, \uXXXX escapes, ...) stays inert rather than
		// throwing as JS would. Reported as a porting limitation.
		return &jsPattern{src: src}
	}
	return &jsPattern{re: re, src: src}
}

// test mirrors RegExp.prototype.test; a nil pattern never matches.
func (p *jsPattern) test(s string) bool { return p != nil && p.re != nil && p.re.MatchString(s) }

// String mirrors RegExp.prototype.toString() for a /u-flagged literal.
func (p *jsPattern) String() string { return "/" + p.src + "/u" }

// config is the rule's resolved options object.
type config struct {
	vars               string
	args               string
	ignoreRestSiblings bool
	caughtErrors       string

	varsIgnorePattern              *jsPattern
	argsIgnorePattern              *jsPattern
	caughtErrorsIgnorePattern      *jsPattern
	destructuredArrayIgnorePattern *jsPattern
}

// parseConfig mirrors the `config` object create() builds from options[0].
func parseConfig(ctx *eslint.Context) *config {
	cfg := &config{vars: "all", args: "after-used", caughtErrors: "none"}

	first := ctx.Option(0)
	if first == nil {
		return cfg
	}
	if s, ok := first.(string); ok {
		cfg.vars = s
		return cfg
	}
	opts := ctx.OptionMap(0)
	if opts == nil {
		return cfg
	}
	// Each `|| default` in the original keeps the default for falsy values.
	if v, ok := opts["vars"].(string); ok && v != "" {
		cfg.vars = v
	}
	if v, ok := opts["args"].(string); ok && v != "" {
		cfg.args = v
	}
	if v, ok := opts["ignoreRestSiblings"].(bool); ok && v {
		cfg.ignoreRestSiblings = true
	}
	if v, ok := opts["caughtErrors"].(string); ok && v != "" {
		cfg.caughtErrors = v
	}
	if v, ok := opts["varsIgnorePattern"].(string); ok && v != "" {
		cfg.varsIgnorePattern = newJSPattern(v)
	}
	if v, ok := opts["argsIgnorePattern"].(string); ok && v != "" {
		cfg.argsIgnorePattern = newJSPattern(v)
	}
	if v, ok := opts["caughtErrorsIgnorePattern"].(string); ok && v != "" {
		cfg.caughtErrorsIgnorePattern = newJSPattern(v)
	}
	if v, ok := opts["destructuredArrayIgnorePattern"].(string); ok && v != "" {
		cfg.destructuredArrayIgnorePattern = newJSPattern(v)
	}
	return cfg
}

// state carries the per-file context through the helpers.
type state struct {
	ctx *eslint.Context
	sc  *eslint.SourceCode
	cfg *config
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	st := &state{ctx: ctx, sc: ctx.SourceCode, cfg: parseConfig(ctx)}
	return map[string]func(eslint.Node){
		"Program:exit": func(programNode eslint.Node) {
			unusedVars := st.collectUnusedVariables(ctx.ScopeOf(programNode), nil)
			for _, unusedVar := range unusedVars {
				st.reportUnused(unusedVar)
			}
		},
	}
}

//------------------------------------------------------------------------------
// Message data
//------------------------------------------------------------------------------

// definedMessageData mirrors getDefinedMessageData.
func (c *config) definedMessageData(variable *eslintscope.Variable) map[string]any {
	defType := ""
	if def := firstDef(variable); def != nil {
		defType = def.Type
	}

	var typ, pattern string
	switch {
	case defType == "CatchClause" && c.caughtErrorsIgnorePattern != nil:
		typ, pattern = "args", c.caughtErrorsIgnorePattern.String()
	case defType == "Parameter" && c.argsIgnorePattern != nil:
		typ, pattern = "args", c.argsIgnorePattern.String()
	case defType != "Parameter" && c.varsIgnorePattern != nil:
		typ, pattern = "vars", c.varsIgnorePattern.String()
	}

	additional := ""
	if typ != "" {
		additional = ". Allowed unused " + typ + " must match " + pattern
	}

	return map[string]any{
		"varName":    variable.Name,
		"action":     "defined",
		"additional": additional,
	}
}

// assignedMessageData mirrors getAssignedMessageData.
func (c *config) assignedMessageData(variable *eslintscope.Variable) map[string]any {
	def := firstDef(variable)

	additional := ""
	if c.destructuredArrayIgnorePattern != nil && def != nil &&
		eslint.NodeType(eslint.Parent(def.Name)) == "ArrayPattern" {
		additional = ". Allowed unused elements of array destructuring patterns must match " +
			c.destructuredArrayIgnorePattern.String()
	} else if c.varsIgnorePattern != nil {
		additional = ". Allowed unused vars must match " + c.varsIgnorePattern.String()
	}

	return map[string]any{
		"varName":    variable.Name,
		"action":     "assigned a value",
		"additional": additional,
	}
}

//------------------------------------------------------------------------------
// Helpers
//------------------------------------------------------------------------------

func firstDef(variable *eslintscope.Variable) *eslintscope.Definition {
	if len(variable.Defs) == 0 {
		return nil
	}
	return variable.Defs[0]
}

// variableScopeOf is the port of eslint-scope's Scope#variableScope, which is
// private in eslint-scope-go: the scope itself when its type introduces one,
// otherwise the nearest enclosing one.
func variableScopeOf(s *eslintscope.Scope) *eslintscope.Scope {
	for s != nil {
		switch s.Type {
		case eslintscope.ScopeGlobal, eslintscope.ScopeModule, eslintscope.ScopeFunction,
			eslintscope.ScopeClassFieldInitializer, eslintscope.ScopeClassStaticBlock:
			return s
		}
		s = s.Upper
	}
	return nil
}

// isInLoop mirrors astUtils.isInLoop.
func isInLoop(node eslint.Node) bool {
	for n := node; n != nil && !eslint.IsFunction(n); n = eslint.Parent(n) {
		if eslint.IsLoop(n) {
			return true
		}
	}
	return false
}

// isLogicalAssignmentOperator mirrors astUtils.isLogicalAssignmentOperator.
func isLogicalAssignmentOperator(op string) bool {
	return op == "&&=" || op == "||=" || op == "??="
}

// isStatementType is the STATEMENT_TYPE regex (/(?:Statement|Declaration)$/u).
func isStatementType(t string) bool {
	return strings.HasSuffix(t, "Statement") || strings.HasSuffix(t, "Declaration")
}

// isRestPropertyType is REST_PROPERTY_TYPE
// (/^(?:RestElement|(?:Experimental)?RestProperty)$/u).
func isRestPropertyType(t string) bool {
	return t == "RestElement" || t == "RestProperty" || t == "ExperimentalRestProperty"
}

// isExported mirrors isExported: the definition's enclosing statement is an
// export declaration.
func isExported(variable *eslintscope.Variable) bool {
	definition := firstDef(variable)
	if definition == nil {
		return false
	}
	node := definition.Node
	switch {
	case eslint.NodeType(node) == "VariableDeclarator":
		node = eslint.Parent(node)
	case definition.Type == "Parameter":
		return false
	}
	return strings.HasPrefix(eslint.NodeType(eslint.Parent(node)), "Export")
}

// hasRestSibling mirrors hasRestSibling.
func hasRestSibling(node eslint.Node) bool {
	if eslint.NodeType(node) != "Property" {
		return false
	}
	parent := eslint.Parent(node)
	if eslint.NodeType(parent) != "ObjectPattern" {
		return false
	}
	properties := eslint.GetNodes(parent, "properties")
	if len(properties) == 0 {
		return false
	}
	return isRestPropertyType(eslint.NodeType(properties[len(properties)-1]))
}

// hasRestSpreadSibling mirrors hasRestSpreadSibling.
func (st *state) hasRestSpreadSibling(variable *eslintscope.Variable) bool {
	if !st.cfg.ignoreRestSiblings {
		return false
	}
	for _, def := range variable.Defs {
		if hasRestSibling(eslint.Parent(def.Name)) {
			return true
		}
	}
	for _, ref := range variable.References {
		if hasRestSibling(eslint.Parent(ref.Identifier)) {
			return true
		}
	}
	return false
}

// isSelfReference mirrors isSelfReference.
func isSelfReference(ref *eslintscope.Reference, nodes []eslint.Node) bool {
	for scope := ref.From; scope != nil; scope = scope.Upper {
		for _, node := range nodes {
			if eslint.SameNode(scope.Block, node) {
				return true
			}
		}
	}
	return false
}

// getFunctionDefinitions mirrors getFunctionDefinitions.
func getFunctionDefinitions(variable *eslintscope.Variable) []eslint.Node {
	var functionDefinitions []eslint.Node
	for _, def := range variable.Defs {
		if def.Type == "FunctionName" {
			functionDefinitions = append(functionDefinitions, def.Node)
		}
		if def.Type == "Variable" && def.Node != nil {
			init := eslint.GetNode(def.Node, "init")
			if init != nil && (eslint.NodeType(init) == "FunctionExpression" ||
				eslint.NodeType(init) == "ArrowFunctionExpression") {
				functionDefinitions = append(functionDefinitions, init)
			}
		}
	}
	return functionDefinitions
}

// isInside mirrors isInside. The JS original would throw on a null `outer`;
// Go returns false, which is only reachable where JS never gets.
func isInside(inner, outer eslint.Node) bool {
	if inner == nil || outer == nil {
		return false
	}
	return eslint.Start(inner) >= eslint.Start(outer) && eslint.End(inner) <= eslint.End(outer)
}

// isUnusedExpression mirrors isUnusedExpression.
func isUnusedExpression(node eslint.Node) bool {
	parent := eslint.Parent(node)
	switch eslint.NodeType(parent) {
	case "ExpressionStatement":
		return true
	case "SequenceExpression":
		expressions := eslint.GetNodes(parent, "expressions")
		if len(expressions) == 0 || !eslint.SameNode(expressions[len(expressions)-1], node) {
			return true
		}
		return isUnusedExpression(parent)
	}
	return false
}

// getRhsNode mirrors getRhsNode.
func getRhsNode(ref *eslintscope.Reference, prevRhsNode eslint.Node) eslint.Node {
	id := ref.Identifier
	parent := eslint.Parent(id)
	refScope := variableScopeOf(ref.From)
	var varScope *eslintscope.Scope
	if ref.Resolved != nil {
		varScope = variableScopeOf(ref.Resolved.Scope)
	}
	canBeUsedLater := refScope != varScope || isInLoop(id)

	// Inherits the previous node if this reference is in the node.
	// This is for `a = a + a`-like code.
	if prevRhsNode != nil && isInside(id, prevRhsNode) {
		return prevRhsNode
	}

	if eslint.NodeType(parent) == "AssignmentExpression" &&
		isUnusedExpression(parent) &&
		eslint.SameNode(id, eslint.GetNode(parent, "left")) &&
		!canBeUsedLater {
		return eslint.GetNode(parent, "right")
	}
	return nil
}

// isStorableFunction mirrors isStorableFunction.
func isStorableFunction(funcNode, rhsNode eslint.Node) bool {
	node := funcNode
	parent := eslint.Parent(funcNode)

	for parent != nil && isInside(parent, rhsNode) {
		switch eslint.NodeType(parent) {
		case "SequenceExpression":
			expressions := eslint.GetNodes(parent, "expressions")
			if len(expressions) == 0 || !eslint.SameNode(expressions[len(expressions)-1], node) {
				return false
			}
		case "CallExpression", "NewExpression":
			return !eslint.SameNode(eslint.GetNode(parent, "callee"), node)
		case "AssignmentExpression", "TaggedTemplateExpression", "YieldExpression":
			return true
		default:
			if isStatementType(eslint.NodeType(parent)) {
				// A complex pattern: return true to avoid false positives.
				return true
			}
		}
		node = parent
		parent = eslint.Parent(parent)
	}
	return false
}

// isInsideOfStorableFunction mirrors isInsideOfStorableFunction.
func isInsideOfStorableFunction(id, rhsNode eslint.Node) bool {
	funcNode := eslint.GetUpperFunction(id)
	return funcNode != nil && isInside(funcNode, rhsNode) && isStorableFunction(funcNode, rhsNode)
}

// isReadForItself mirrors isReadForItself.
func isReadForItself(ref *eslintscope.Reference, rhsNode eslint.Node) bool {
	id := ref.Identifier
	parent := eslint.Parent(id)

	if !ref.IsRead() {
		return false
	}

	// self update, e.g. `a += 1`, `a++`
	if (eslint.NodeType(parent) == "AssignmentExpression" &&
		eslint.SameNode(eslint.GetNode(parent, "left"), id) &&
		isUnusedExpression(parent) &&
		!isLogicalAssignmentOperator(eslint.GetString(parent, "operator"))) ||
		(eslint.NodeType(parent) == "UpdateExpression" && isUnusedExpression(parent)) {
		return true
	}

	// in RHS of an assignment for itself, e.g. `a = a + 1`
	return rhsNode != nil && isInside(id, rhsNode) && !isInsideOfStorableFunction(id, rhsNode)
}

// isForInOfRef mirrors isForInOfRef.
func isForInOfRef(ref *eslintscope.Reference) bool {
	target := eslint.Parent(ref.Identifier)

	// "for (var ...) { return; }"
	if eslint.NodeType(target) == "VariableDeclarator" {
		target = eslint.Parent(eslint.Parent(target))
	}
	if eslint.NodeType(target) != "ForInStatement" && eslint.NodeType(target) != "ForOfStatement" {
		return false
	}

	body := eslint.GetNode(target, "body")
	if eslint.NodeType(body) == "BlockStatement" {
		statements := eslint.GetNodes(body, "body")
		if len(statements) == 0 {
			return false // empty loop body
		}
		target = statements[0]
	} else {
		target = body
	}
	if target == nil {
		return false
	}
	return eslint.NodeType(target) == "ReturnStatement"
}

// isUsedVariable mirrors isUsedVariable.
func (st *state) isUsedVariable(variable *eslintscope.Variable) bool {
	functionNodes := getFunctionDefinitions(variable)
	isFunctionDefinition := len(functionNodes) > 0
	var rhsNode eslint.Node

	for _, ref := range variable.References {
		if isForInOfRef(ref) {
			return true
		}
		forItself := isReadForItself(ref, rhsNode)
		rhsNode = getRhsNode(ref, rhsNode)
		if ref.IsRead() && !forItself &&
			!(isFunctionDefinition && isSelfReference(ref, functionNodes)) {
			return true
		}
	}
	return false
}

// isAfterLastUsedArg mirrors isAfterLastUsedArg.
func (st *state) isAfterLastUsedArg(variable *eslintscope.Variable) bool {
	def := firstDef(variable)
	if def == nil {
		return false
	}
	params := st.sc.DeclaredVariables(def.Node)
	index := -1
	for i, param := range params {
		if param == variable {
			index = i
			break
		}
	}
	// slice(indexOf + 1) — indexOf -1 yields the whole list, as in JS.
	for _, posterior := range params[index+1:] {
		if len(posterior.References) > 0 || posterior.ESLintUsed {
			return false
		}
	}
	return true
}

// collectUnusedVariables mirrors collectUnusedVariables.
func (st *state) collectUnusedVariables(scope *eslintscope.Scope, unusedVars []*eslintscope.Variable) []*eslintscope.Variable {
	if scope == nil {
		return unusedVars
	}

	if scope.Type != eslintscope.ScopeGlobal || st.cfg.vars == "all" {
		for _, variable := range scope.Variables {
			// skip a variable of class itself name in the class scope
			if scope.Type == eslintscope.ScopeClass && len(variable.Identifiers) > 0 &&
				eslint.SameNode(eslint.GetNode(scope.Block, "id"), variable.Identifiers[0]) {
				continue
			}

			// skip function expression names and variables marked with
			// markVariableAsUsed()
			if scope.FunctionExpressionScope || variable.ESLintUsed {
				continue
			}

			// skip implicit "arguments" variable
			if scope.Type == eslintscope.ScopeFunction && variable.Name == "arguments" &&
				len(variable.Identifiers) == 0 {
				continue
			}

			// explicit global variables don't have definitions.
			if def := firstDef(variable); def != nil {
				defType := def.Type
				name := eslint.GetString(def.Name, "name")
				refUsedInArrayPatterns := false
				for _, ref := range variable.References {
					if eslint.NodeType(eslint.Parent(ref.Identifier)) == "ArrayPattern" {
						refUsedInArrayPatterns = true
						break
					}
				}

				// skip elements of array destructuring patterns
				if (eslint.NodeType(eslint.Parent(def.Name)) == "ArrayPattern" || refUsedInArrayPatterns) &&
					st.cfg.destructuredArrayIgnorePattern.test(name) {
					continue
				}

				// skip catch variables
				if defType == "CatchClause" {
					if st.cfg.caughtErrors == "none" {
						continue
					}
					// skip ignored parameters
					if st.cfg.caughtErrorsIgnorePattern.test(name) {
						continue
					}
				}

				if defType == "Parameter" {
					defParent := eslint.Parent(def.Node)
					defParentType := eslint.NodeType(defParent)

					// skip any setter argument
					if (defParentType == "Property" || defParentType == "MethodDefinition") &&
						eslint.GetString(defParent, "kind") == "set" {
						continue
					}

					// if "args" option is "none", skip any parameter
					if st.cfg.args == "none" {
						continue
					}

					// skip ignored parameters
					if st.cfg.argsIgnorePattern.test(name) {
						continue
					}

					// if "args" option is "after-used", skip used variables
					if st.cfg.args == "after-used" &&
						eslint.IsFunction(eslint.Parent(def.Name)) &&
						!st.isAfterLastUsedArg(variable) {
						continue
					}
				} else {
					// skip ignored variables
					if st.cfg.varsIgnorePattern.test(name) {
						continue
					}
				}
			}

			if !st.isUsedVariable(variable) && !isExported(variable) && !st.hasRestSpreadSibling(variable) {
				unusedVars = append(unusedVars, variable)
			}
		}
	}

	for _, child := range scope.ChildScopes {
		unusedVars = st.collectUnusedVariables(child, unusedVars)
	}
	return unusedVars
}

// reportUnused mirrors the Program:exit report loop. The JS rule's second
// branch (an unused `/*global x*/` directive) cannot fire here — see the
// package doc.
func (st *state) reportUnused(unusedVar *eslintscope.Variable) {
	if len(unusedVar.Defs) == 0 {
		return
	}

	// report last write reference (eslint issue #14324)
	var writeReferences []*eslintscope.Reference
	for _, ref := range unusedVar.References {
		if ref.IsWrite() && variableScopeOf(ref.From) == variableScopeOf(unusedVar.Scope) {
			writeReferences = append(writeReferences, ref)
		}
	}

	var node eslint.Node
	if len(writeReferences) > 0 {
		node = writeReferences[len(writeReferences)-1].Identifier
	} else if len(unusedVar.Identifiers) > 0 {
		node = unusedVar.Identifiers[0]
	}

	assigned := false
	for _, ref := range unusedVar.References {
		if ref.IsWrite() {
			assigned = true
			break
		}
	}

	data := st.cfg.definedMessageData(unusedVar)
	if assigned {
		data = st.cfg.assignedMessageData(unusedVar)
	}

	st.ctx.Report(eslint.Report{
		Node:      node,
		MessageID: "unusedVar",
		Data:      data,
	})
}
