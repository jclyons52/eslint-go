package eslint

// SourceCode wraps the text and AST being linted, mirroring the minimal
// surface of ESLint's SourceCode that the starter rules need.
type SourceCode struct {
	Text string
	AST  Node
}

func newSourceCode(text string, ast Node) *SourceCode {
	return &SourceCode{Text: text, AST: ast}
}

// event is a single nodeEventGenerator step: entering or leaving a node.
type event struct {
	isEntering bool
	node       Node
}

// collectEvents runs a Traverser-style DFS over the AST, attaching `parent`
// to every node and returning the ordered enter/leave event queue (exactly
// as runRules builds nodeQueue).
func collectEvents(ast Node) []event {
	var queue []event
	var walk func(node, parent Node)
	walk = func(node, parent Node) {
		if !isNode(node) {
			return
		}
		if parent != nil {
			node["parent"] = parent
		}
		queue = append(queue, event{isEntering: true, node: node})
		keys := visitorKeys[nodeType(node)]
		for _, k := range keys {
			child := node[k]
			switch c := child.(type) {
			case []any:
				for _, el := range c {
					walk(asNode(el), node)
				}
			case Node:
				walk(c, node)
			}
		}
		queue = append(queue, event{isEntering: false, node: node})
	}
	walk(ast, nil)
	return queue
}

func isNode(x any) bool {
	if x == nil {
		return false
	}
	m, ok := x.(Node)
	if !ok {
		return false
	}
	_, ok = m["type"].(string)
	return ok
}

func asNode(x any) Node {
	if m, ok := x.(Node); ok {
		return m
	}
	return nil
}
