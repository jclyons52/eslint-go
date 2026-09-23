package nosparsearrays

import eslint "github.com/jclyons52/eslint-go"

// Name is the ESLint rule id.
const Name = "no-sparse-arrays"

// Rule is a port of eslint/lib/rules/no-sparse-arrays.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Disallow sparse arrays",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-sparse-arrays",
		},
		Schema: []any{},
		Messages: map[string]string{
			"unexpectedSparseArray": "Unexpected comma in middle of array.",
		},
	},
	Create: create,
}

// hasEmptySpot mirrors `node.elements.includes(null)`. The parser represents a
// hole as a nil entry in the raw elements array, so the check runs over the
// untyped slice rather than eslint.GetNodes (which drops nils).
func hasEmptySpot(node eslint.Node) bool {
	elements, ok := eslint.Get(node, "elements").([]any)
	if !ok {
		return false
	}
	for _, element := range elements {
		if element == nil {
			return true
		}
	}
	return false
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	return map[string]func(eslint.Node){
		"ArrayExpression": func(node eslint.Node) {
			if hasEmptySpot(node) {
				ctx.Report(eslint.Report{Node: node, MessageID: "unexpectedSparseArray"})
			}
		},
	}
}
