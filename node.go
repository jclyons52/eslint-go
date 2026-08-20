package eslint

// Node is an ESTree AST node carried as a generic map (matches the shape of
// espree JSON output, so the AST can be fed in from the JS oracle or from
// espree-go).
type Node = map[string]any

func nodeType(n Node) string {
	if n == nil {
		return ""
	}
	t, _ := n["type"].(string)
	return t
}

func getNode(n Node, key string) Node {
	if n == nil {
		return nil
	}
	switch c := n[key].(type) {
	case Node:
		return c
	}
	return nil
}

func getNodes(n Node, key string) []Node {
	if n == nil {
		return nil
	}
	arr, ok := n[key].([]any)
	if !ok {
		return nil
	}
	out := make([]Node, 0, len(arr))
	for _, el := range arr {
		if m, ok := el.(Node); ok {
			out = append(out, m)
		} else if m, ok := el.(map[string]any); ok {
			out = append(out, Node(m))
		}
	}
	return out
}

func getString(n Node, key string) string {
	if n == nil {
		return ""
	}
	s, _ := n[key].(string)
	return s
}

func getBool(n Node, key string) bool {
	if n == nil {
		return false
	}
	b, _ := n[key].(bool)
	return b
}

func getFloat(n Node, key string) float64 {
	if n == nil {
		return 0
	}
	f, _ := n[key].(float64)
	return f
}

// locLine / locColumn read {line, column} from a loc object like
// { "start": {"line":1,"column":0}, "end": {...} }.
func locLine(loc map[string]any, which string) int {
	pos, _ := loc[which].(map[string]any)
	if pos == nil {
		return 1
	}
	return int(getFloat(Node(pos), "line"))
}

func locColumn(loc map[string]any, which string) int {
	pos, _ := loc[which].(map[string]any)
	if pos == nil {
		return 0
	}
	return int(getFloat(Node(pos), "column"))
}
