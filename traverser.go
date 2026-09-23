package eslint

// traverser.go — the traversal that dispatches node events to rules,
// mirroring ESLint's shared/traverser.js (a keyed DFS) plus linter.js's
// runRules: one enter/leave event per node, in source order, with `parent`
// attached, and rule listeners invoked in configuration order.

// nodeEvent is one step of the event stream.
type nodeEvent struct {
	node     Node
	entering bool
}

// collectEvents runs the keyed DFS over the AST, attaches `parent` to every
// node, and returns the enter/leave event stream.
func collectEvents(ast Node) []nodeEvent {
	var queue []nodeEvent
	var walk func(node, parent Node)
	walk = func(node, parent Node) {
		if !isNode(node) {
			return
		}
		if parent != nil {
			node["parent"] = parent
		}
		queue = append(queue, nodeEvent{node: node, entering: true})
		for _, key := range VisitorKeys[NodeType(node)] {
			switch child := node[key].(type) {
			case []any:
				for _, el := range child {
					walk(asNode(el), node)
				}
			case Node:
				walk(child, node)
			}
		}
		queue = append(queue, nodeEvent{node: node, entering: false})
	}
	walk(ast, nil)
	return queue
}

// listener is one bound rule selector.
type listener struct {
	rule *ruleRun
	typ  string
	exit bool
	fn   func(Node)
}

// ruleRun holds a rule instance's runtime state for one file.
type ruleRun struct {
	id  string
	ctx *Context
}
