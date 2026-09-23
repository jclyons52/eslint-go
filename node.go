package eslint

import "reflect"

// This file holds the shared AST node vocabulary. Nodes are the ESTree
// map[string]any shape that espree produces and that every ESLint rule in the
// original codebase consumes; the helpers below are the Go equivalent of the
// small property-access idiom used throughout ESLint's rules.

// Node is an ESTree AST node or token, in the same JSON shape espree emits.
type Node = map[string]any

// NodeType returns the node's "type", or "" for nil.
func NodeType(n Node) string {
	if n == nil {
		return ""
	}
	t, _ := n["type"].(string)
	return t
}

// Get returns the raw value of a property.
func Get(n Node, key string) any {
	if n == nil {
		return nil
	}
	return n[key]
}

// GetNode returns an object-valued property as a Node (nil when absent).
func GetNode(n Node, key string) Node {
	if n == nil {
		return nil
	}
	c, _ := n[key].(Node)
	return c
}

// GetNodes returns an array-valued property as []Node, skipping null elements
// (e.g. ArrayExpression elements) and non-node entries.
func GetNodes(n Node, key string) []Node {
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
		}
	}
	return out
}

// GetString returns a string-valued property ("" when absent).
func GetString(n Node, key string) string {
	if n == nil {
		return ""
	}
	s, _ := n[key].(string)
	return s
}

// GetStringOK is GetString with an explicit presence check — needed wherever a
// missing name and an empty name are semantically different.
func GetStringOK(n Node, key string) (string, bool) {
	if n == nil {
		return "", false
	}
	s, ok := n[key].(string)
	return s, ok
}

// GetBool returns a bool-valued property (false when absent).
func GetBool(n Node, key string) bool {
	if n == nil {
		return false
	}
	b, _ := n[key].(bool)
	return b
}

// GetFloat returns a numeric property as float64 (0 when absent).
func GetFloat(n Node, key string) float64 {
	if n == nil {
		return 0
	}
	switch v := n[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

// GetInt returns a numeric property as int.
func GetInt(n Node, key string) int { return int(GetFloat(n, key)) }

// GetMap returns an object-valued property as a plain map.
func GetMap(n Node, key string) map[string]any {
	if n == nil {
		return nil
	}
	m, _ := n[key].(map[string]any)
	return m
}

// HasRange reports whether the node carries source offsets.
func HasRange(n Node) bool {
	if _, ok := n["range"].([]any); ok {
		return true
	}
	_, s := n["start"]
	_, e := n["end"]
	return s && e
}

// Range returns the node's [start, end] source offsets. It prefers the ESTree
// `range` array and falls back to `start`/`end`; ok is false when neither is
// present (e.g. a synthetic node).
func Range(n Node) ([2]int, bool) {
	if n == nil {
		return [2]int{}, false
	}
	if r, ok := n["range"].([]any); ok && len(r) == 2 {
		return [2]int{numToInt(r[0]), numToInt(r[1])}, true
	}
	s, sok := n["start"]
	e, eok := n["end"]
	if sok && eok {
		return [2]int{numToInt(s), numToInt(e)}, true
	}
	return [2]int{}, false
}

// MustRange returns Range without the ok flag ({0,0} for a node with no
// position information). Rules call this on parser-produced nodes, which
// always carry positions.
func MustRange(n Node) [2]int {
	r, _ := Range(n)
	return r
}

// Start returns the node's start offset (0 when absent).
func Start(n Node) int {
	r, ok := Range(n)
	if !ok {
		return 0
	}
	return r[0]
}

// End returns the node's end offset (0 when absent).
func End(n Node) int {
	r, ok := Range(n)
	if !ok {
		return 0
	}
	return r[1]
}

// Loc returns the node's ESTree loc object ({start:{line,column},end:{...}}).
func Loc(n Node) map[string]any {
	if n == nil {
		return nil
	}
	m, _ := n["loc"].(map[string]any)
	return m
}

// LocStart returns the 1-based line and 0-based column of a node's start.
func LocStart(n Node) (line, col int) { return locPoint(Loc(n), "start") }

// LocEnd returns the 1-based line and 0-based column of a node's end.
func LocEnd(n Node) (line, col int) { return locPoint(Loc(n), "end") }

func locPoint(loc map[string]any, which string) (line, col int) {
	if loc == nil {
		return 0, 0
	}
	pt, _ := loc[which].(map[string]any)
	if pt == nil {
		return 0, 0
	}
	return numToInt(pt["line"]), numToInt(pt["column"])
}

// LocOf builds a {start:{line,column},end:{line,column}} loc object from two
// line/column pairs, mirroring the object literals rules pass to report().
func LocOf(startLine, startCol, endLine, endCol int) map[string]any {
	return map[string]any{
		"start": map[string]any{"line": float64(startLine), "column": float64(startCol)},
		"end":   map[string]any{"line": float64(endLine), "column": float64(endCol)},
	}
}

// StartLoc builds a {start:{...}} loc with no end, which suppresses
// endLine/endColumn in the reported message (report-translator behaviour).
func StartLoc(line, col int) map[string]any {
	return map[string]any{"start": map[string]any{"line": float64(line), "column": float64(col)}}
}

// Parent returns the node's parent (attached by the traverser), or nil.
func Parent(n Node) Node { return GetNode(n, "parent") }

// SetParent attaches a parent pointer.
func SetParent(n Node, p Node) {
	if n != nil {
		n["parent"] = p
	}
}

// AncestorsOf returns the ancestor chain of a node, root first, excluding the
// node itself (SourceCode.getAncestors).
func AncestorsOf(n Node) []Node {
	var rev []Node
	for p := Parent(n); p != nil; p = Parent(p) {
		rev = append(rev, p)
	}
	out := make([]Node, 0, len(rev))
	for i := len(rev) - 1; i >= 0; i-- {
		out = append(out, rev[i])
	}
	return out
}

// IsNode reports whether a value is a node-shaped map carrying a "type".
func IsNode(x any) bool {
	m, ok := x.(Node)
	if !ok {
		return false
	}
	_, ok = m["type"].(string)
	return ok
}

func isNode(n Node) bool {
	if n == nil {
		return false
	}
	_, ok := n["type"].(string)
	return ok
}

func asNode(x any) Node {
	m, _ := x.(Node)
	return m
}

func numToInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return 0
}

// nodeKey returns a comparable identity for a node map, used for identity
// maps and comparisons (Go maps are not comparable, so `a == b` is illegal).
func nodeKey(n Node) uintptr {
	if n == nil {
		return 0
	}
	return reflect.ValueOf(n).Pointer()
}

// SameNode reports whether two nodes are the same object.
func SameNode(a, b Node) bool { return nodeKey(a) == nodeKey(b) }
