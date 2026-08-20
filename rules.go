package eslint

// RuleContext is the object passed to a rule's create(); it exposes the
// subset of ESLint's rule context that the starter rules use.
type RuleContext struct {
	ID         string
	Severity   int
	Options    []any
	SourceCode *SourceCode
	messages   map[string]string
	collect    func(*Message)
}

// Report translates a context.report() descriptor into a Message and appends
// it to the run's output (mirroring the report-translator for the no-fix
// subset).
func (c *RuleContext) Report(d ReportDescriptor) {
	var computed string
	if d.MessageID != "" {
		computed = c.messages[d.MessageID]
	} else if d.Message != "" {
		computed = d.Message
	}
	msg := interpolate(computed, d.Data)
	loc := normalizeReportLoc(&d)
	problem := createProblem(c.ID, c.Severity, d.Node, msg, d.MessageID, loc)
	c.collect(problem)
}

// ruleMeta describes an implemented rule.
type ruleMeta struct {
	create   func(ctx *RuleContext) map[string]func(Node)
	messages map[string]string
}

// ruleRegistry maps rule IDs to their Go implementations.
var ruleRegistry = map[string]*ruleMeta{
	"no-debugger": {
		create: createNoDebugger,
		messages: map[string]string{
			"unexpected": "Unexpected 'debugger' statement.",
		},
	},
	"no-dupe-keys": {
		create: createNoDupeKeys,
		messages: map[string]string{
			"unexpected": "Duplicate key '{{name}}'.",
		},
	},
}

func createNoDebugger(ctx *RuleContext) map[string]func(Node) {
	return map[string]func(Node){
		"DebuggerStatement": func(node Node) {
			ctx.Report(ReportDescriptor{Node: node, MessageID: "unexpected"})
		},
	}
}

// propEntry tracks whether a property key was seen as get/set/init.
type propEntry struct{ get, set bool }

type objInfo struct {
	upper *objInfo
	node  Node
	props map[string]*propEntry
}

func (o *objInfo) getPropertyInfo(node Node) *propEntry {
	name := getStaticPropertyName(node)
	e, ok := o.props[name]
	if !ok {
		e = &propEntry{}
		o.props[name] = e
	}
	return e
}

func (o *objInfo) isPropertyDefined(node Node) bool {
	e := o.getPropertyInfo(node)
	kind := getString(node, "kind")
	return (isGetKind(kind) && e.get) || (isSetKind(kind) && e.set)
}

func (o *objInfo) defineProperty(node Node) {
	e := o.getPropertyInfo(node)
	kind := getString(node, "kind")
	if isGetKind(kind) {
		e.get = true
	}
	if isSetKind(kind) {
		e.set = true
	}
}

func isGetKind(kind string) bool { return kind == "init" || kind == "get" }
func isSetKind(kind string) bool { return kind == "init" || kind == "set" }

func createNoDupeKeys(ctx *RuleContext) map[string]func(Node) {
	var info *objInfo
	return map[string]func(Node){
		"ObjectExpression": func(node Node) {
			info = &objInfo{upper: info, node: node, props: map[string]*propEntry{}}
		},
		"ObjectExpression:exit": func(Node) {
			info = info.upper
		},
		"Property": func(node Node) {
			name := getStaticPropertyName(node)
			// Skip destructuring.
			if p := getNode(node, "parent"); p == nil || nodeType(p) != "ObjectExpression" {
				return
			}
			// Skip if the name is not static.
			if name == "" {
				return
			}
			// Report if the name is defined already.
			if info.isPropertyDefined(node) {
				ctx.Report(ReportDescriptor{
					Node:      info.node,
					Loc:       getLocObject(getNode(node, "key")),
					MessageID: "unexpected",
					Data:      map[string]any{"name": name},
				})
			}
			info.defineProperty(node)
		},
	}
}
