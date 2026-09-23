package nodupekeys

import eslint "github.com/jclyons52/eslint-go"

// Name is the ESLint rule id.
const Name = "no-dupe-keys"

// Rule is a port of eslint/lib/rules/no-dupe-keys.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Disallow duplicate keys in object literals",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-dupe-keys",
		},
		Messages: map[string]string{
			"unexpected": "Duplicate key '{{name}}'.",
		},
	},
	Create: create,
}

// propEntry tracks whether a name was already seen as get/set/init.
type propEntry struct{ get, set bool }

// objectInfo stores an ObjectExpression's property information, chained to the
// enclosing object (the ES6 scope of duplicate detection is the object node).
type objectInfo struct {
	upper      *objectInfo
	node       eslint.Node
	properties map[string]*propEntry
}

func (o *objectInfo) getPropertyInfo(node eslint.Node) *propEntry {
	name, _ := eslint.GetStaticPropertyName(node)
	e, ok := o.properties[name]
	if !ok {
		e = &propEntry{}
		o.properties[name] = e
	}
	return e
}

func (o *objectInfo) isPropertyDefined(node eslint.Node) bool {
	e := o.getPropertyInfo(node)
	kind := eslint.GetString(node, "kind")
	return (kind == "init" || kind == "get") && e.get || (kind == "init" || kind == "set") && e.set
}

func (o *objectInfo) defineProperty(node eslint.Node) {
	e := o.getPropertyInfo(node)
	kind := eslint.GetString(node, "kind")
	if kind == "init" || kind == "get" {
		e.get = true
	}
	if kind == "init" || kind == "set" {
		e.set = true
	}
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	var info *objectInfo
	return map[string]func(eslint.Node){
		"ObjectExpression": func(node eslint.Node) {
			info = &objectInfo{upper: info, node: node, properties: map[string]*propEntry{}}
		},
		"ObjectExpression:exit": func(eslint.Node) {
			info = info.upper
		},
		"Property": func(node eslint.Node) {
			name, static := eslint.GetStaticPropertyName(node)

			// Skip destructuring.
			if eslint.NodeType(eslint.Parent(node)) != "ObjectExpression" {
				return
			}
			// Skip if the name is not static.
			if !static {
				return
			}

			if info.isPropertyDefined(node) {
				ctx.Report(eslint.Report{
					Node:      info.node,
					Loc:       eslint.Loc(eslint.GetNode(node, "key")),
					MessageID: "unexpected",
					Data:      map[string]any{"name": name},
				})
			}
			info.defineProperty(node)
		},
	}
}
