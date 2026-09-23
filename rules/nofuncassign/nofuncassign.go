package nofuncassign

import (
	eslint "github.com/jclyons52/eslint-go"
	eslintscope "github.com/jclyons52/eslint-scope-go"
)

// Name is the ESLint rule id.
const Name = "no-func-assign"

// Rule is a port of eslint/lib/rules/no-func-assign.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "problem",
		Docs: eslint.RuleDocs{
			Description: "Disallow reassigning `function` declarations",
			Recommended: true,
			URL:         "https://eslint.org/docs/latest/rules/no-func-assign",
		},
		Schema: []any{},
		Messages: map[string]string{
			"isAFunction": "'{{name}}' is a function.",
		},
	},
	Create: create,
}

// isModifyingReference mirrors astUtils' private isModifyingReference: a
// reference is modifying when it is non-initializer, writable, and not a
// repeated write of the same identifier (destructuring defaults).
func isModifyingReference(references []*eslintscope.Reference, index int) bool {
	reference := references[index]
	identifier := reference.Identifier

	modifyingDifferentIdentifier := index == 0 ||
		!eslint.SameNode(references[index-1].Identifier, identifier)

	return identifier != nil &&
		!reference.Init &&
		reference.IsWrite() &&
		modifyingDifferentIdentifier
}

// getModifyingReferences is astUtils.getModifyingReferences.
func getModifyingReferences(references []*eslintscope.Reference) []*eslintscope.Reference {
	out := make([]*eslintscope.Reference, 0, len(references))
	for i := range references {
		if isModifyingReference(references, i) {
			out = append(out, references[i])
		}
	}
	return out
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	checkReference := func(references []*eslintscope.Reference) {
		for _, reference := range getModifyingReferences(references) {
			ctx.Report(eslint.Report{
				Node:      reference.Identifier,
				MessageID: "isAFunction",
				Data: map[string]any{
					"name": eslint.GetString(reference.Identifier, "name"),
				},
			})
		}
	}

	checkVariable := func(variable *eslintscope.Variable) {
		if len(variable.Defs) > 0 && variable.Defs[0].Type == eslintscope.VarFunctionName {
			checkReference(variable.References)
		}
	}

	checkForFunction := func(node eslint.Node) {
		for _, variable := range ctx.DeclaredVariables(node) {
			checkVariable(variable)
		}
	}

	return map[string]func(eslint.Node){
		"FunctionDeclaration": checkForFunction,
		"FunctionExpression":  checkForFunction,
	}
}
