// Package eslint is a Go implementation of ESLint's Linter pipeline.
//
// It runs real ESLint rules over a real ESTree AST produced entirely in Go
// (espree-go → acorn-go), performs eslint-scope scope analysis, reproduces the
// report-translator's message objects, and can apply rule fixes — all validated
// by JS-oracle parity against eslint 8.57.0 (the Go output must be identical to
// the npm original's).
//
// Layer map (matching ESLint 8):
//
//	eslint.go        Linter.verify / verifyAndFix
//	source_code.go   SourceCode (+ token store, the "cursor" API)
//	traverser.go     node-event-generator / traverser
//	rule.go          rule context + report API
//	report.go        report-translator (createProblem)
//	fixer.go         rule-fixer + source-code-fixer (applyFixes)
//	scope.go         eslint-scope integration (via eslint-scope-go)
//	ast_utils.go     rules/utils/ast-utils subset
//	rules/           the ported rule set (one package per rule)
package eslint
