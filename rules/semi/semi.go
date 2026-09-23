package semi

import (
	"regexp"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "semi"

// Rule is a port of eslint/lib/rules/semi.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type:       "layout",
		Deprecated: true,
		Docs: eslint.RuleDocs{
			Description: "Require or disallow semicolons instead of ASI",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/semi",
		},
		Fixable: "code",
		Messages: map[string]string{
			"missingSemi": "Missing semicolon.",
			"extraSemi":   "Extra semicolon.",
		},
		Schema: []any{
			map[string]any{
				"anyOf": []any{
					map[string]any{
						"type": "array",
						"items": []any{
							map[string]any{"enum": []any{"never"}},
							map[string]any{
								"type": "object",
								"properties": map[string]any{
									"beforeStatementContinuationChars": map[string]any{
										"enum": []any{"always", "any", "never"},
									},
								},
								"additionalProperties": false,
							},
						},
						"minItems": 0,
						"maxItems": 2,
					},
					map[string]any{
						"type": "array",
						"items": []any{
							map[string]any{"enum": []any{"always"}},
							map[string]any{
								"type": "object",
								"properties": map[string]any{
									"omitLastInOneLineBlock":     map[string]any{"type": "boolean"},
									"omitLastInOneLineClassBody": map[string]any{"type": "boolean"},
								},
								"additionalProperties": false,
							},
						},
						"minItems": 0,
						"maxItems": 2,
					},
				},
			},
		},
	},
	Create: create,
}

// optOutPattern is the source rule's /^[-[(/+`]/u — a token whose first
// character is one of `[`, `(`, `/`, `+`, `-`, “ ` “.
const optOutFirstChars = "[(/+`-"

var (
	// linebreakRe matches the line terminators SourceCode.lines splits on
	// (astUtils.createGlobalLinebreakMatcher: \r\n | \r | \n | U+2028 | U+2029).
	linebreakRe = regexp.MustCompile("\\r\\n|[\\r\\n\\x{2028}\\x{2029}]")
)

// unsafeClassFieldNames / unsafeClassFieldFollowers are the source rule's two
// Sets of names that make a class field's trailing semicolon load-bearing.
var (
	unsafeClassFieldNames     = map[string]bool{"get": true, "set": true, "static": true}
	unsafeClassFieldFollowers = map[string]bool{"*": true, "in": true, "instanceof": true}
)

// point builds a {line,column} source-location point.
func point(line, col int) map[string]any {
	return map[string]any{"line": float64(line), "column": float64(col)}
}

// jsLines splits text the way ESLint's SourceCode.lines does: on the global
// linebreak matcher, and *including* the trailing empty element a final
// newline produces (the core's SourceCode.Lines deliberately drops it, so the
// rule reproduces the original here).
func jsLines(text string) []string {
	var lines []string
	start := 0
	for _, m := range linebreakRe.FindAllStringIndex(text, -1) {
		lines = append(lines, text[start:m[0]])
		start = m[1]
	}
	return append(lines, text[start:])
}

// nextLocation is astUtils.getNextLocation: the location one column after
// {line, column}, rolling to the next line, or nil at the very end of the file.
func nextLocation(sc *eslint.SourceCode, line, column int) map[string]any {
	lines := jsLines(sc.Text())
	if line-1 >= 0 && line-1 < len(lines) && column < len(lines[line-1]) {
		return point(line, column+1)
	}
	if line < len(lines) {
		return point(line+1, 0)
	}
	return nil
}

// isTokenOnSameLine is astUtils.isTokenOnSameLine.
func isTokenOnSameLine(left, right eslint.Node) bool {
	if left == nil || right == nil {
		return false
	}
	leftLine, _ := eslint.LocEnd(left)
	rightLine, _ := eslint.LocStart(right)
	return leftLine == rightLine
}

// maybeAsiHazardBefore is the source rule's maybeAsiHazardBefore.
func maybeAsiHazardBefore(token eslint.Node) bool {
	if token == nil {
		return false
	}
	v := eslint.TokenValue(token)
	if v == "++" || v == "--" {
		return false
	}
	return len(v) > 0 && strings.IndexByte(optOutFirstChars, v[0]) >= 0
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	options := ctx.OptionMap(1)
	never := ctx.Option(0) == "never"
	exceptOneLine := false
	if b, ok := options["omitLastInOneLineBlock"].(bool); ok {
		exceptOneLine = b
	}
	exceptOneLineClassBody := false
	if b, ok := options["omitLastInOneLineClassBody"].(bool); ok {
		exceptOneLineClassBody = b
	}
	beforeStatementContinuationChars := "any"
	if v, ok := options["beforeStatementContinuationChars"].(string); ok && v != "" {
		beforeStatementContinuationChars = v
	}

	// report is the source rule's report(node, missing): a falsy `missing`
	// means the semicolon is absent ("missingSemi"), a truthy one means it is
	// present but removable ("extraSemi").
	report := func(node eslint.Node, extraSemi bool) {
		lastToken := sc.GetLastToken(node)
		var loc map[string]any
		var fix func(*eslint.Fixer) *eslint.Fix
		messageID := "missingSemi"

		if !extraSemi {
			endLine, endCol := eslint.LocEnd(lastToken)
			loc = map[string]any{"start": point(endLine, endCol)}
			if next := nextLocation(sc, endLine, endCol); next != nil {
				loc["end"] = next
			} else {
				loc["end"] = nil
			}
			token := lastToken
			fix = func(f *eslint.Fixer) *eslint.Fix {
				return f.InsertTextAfter(token, ";")
			}
		} else {
			messageID = "extraSemi"
			loc = eslint.Loc(lastToken)
			token := lastToken
			fix = func(f *eslint.Fixer) *eslint.Fix {
				/*
				 * Expand the replacement range to include the surrounding
				 * tokens to avoid conflicting with no-extra-semi.
				 * https://github.com/eslint/eslint/issues/7928
				 */
				return eslint.NewFixTracker(f, sc).RetainSurroundingTokens(token).Remove(token)
			}
		}

		ctx.Report(eslint.Report{
			Node:      node,
			Loc:       loc,
			MessageID: messageID,
			Fix:       fix,
		})
	}

	// isRedundantSemi: true when the next token is `;` or `}` (or EOF).
	isRedundantSemi := func(semiToken eslint.Node) bool {
		nextToken := sc.GetTokenAfter(semiToken)
		return nextToken == nil ||
			eslint.IsClosingBraceToken(nextToken) ||
			eslint.IsSemicolonToken(nextToken)
	}

	// isEndOfArrowBlock: true when the token is the closing brace of an arrow
	// function's body block.
	isEndOfArrowBlock := func(lastToken eslint.Node) bool {
		if !eslint.IsClosingBraceToken(lastToken) {
			return false
		}
		node := sc.GetNodeByRangeIndex(eslint.Start(lastToken))
		return eslint.NodeType(node) == "BlockStatement" &&
			eslint.NodeType(eslint.Parent(node)) == "ArrowFunctionExpression"
	}

	// maybeClassFieldAsiHazard: true when a class field's semicolon cannot be
	// removed without changing what the following token means.
	maybeClassFieldAsiHazard := func(node eslint.Node) bool {
		if eslint.NodeType(node) != "PropertyDefinition" {
			return false
		}

		/*
		 * Computed property names and non-identifiers are always safe
		 * as they can be distinguished from keywords easily.
		 */
		key := eslint.GetNode(node, "key")
		needsNameCheck := !eslint.GetBool(node, "computed") && eslint.NodeType(key) == "Identifier"

		/*
		 * Certain names are problematic unless they also have a
		 * a way to distinguish between keywords and property
		 * names.
		 */
		if needsNameCheck && unsafeClassFieldNames[eslint.GetString(key, "name")] {

			/*
			 * Special case: If the field name is `static`,
			 * it is only valid if the field is marked as static,
			 * so "static static" is okay but "static" is not.
			 */
			isStaticStatic := eslint.GetBool(node, "static") && eslint.GetString(key, "name") == "static"

			/*
			 * For other unsafe names, we only care if there is no
			 * initializer. No initializer = hazard.
			 */
			if !isStaticStatic && eslint.Get(node, "value") == nil {
				return true
			}
		}

		followingToken := sc.GetTokenAfter(node)
		return unsafeClassFieldFollowers[eslint.TokenValue(followingToken)]
	}

	// isOnSameLineWithNextToken.
	isOnSameLineWithNextToken := func(node eslint.Node) bool {
		prevToken := sc.GetLastToken(node, eslint.TokenOpt{Skip: 1})
		nextToken := sc.GetTokenAfter(node)
		return nextToken != nil && isTokenOnSameLine(prevToken, nextToken)
	}

	// maybeAsiHazardAfter: false when ASI is safe after the node.
	maybeAsiHazardAfter := func(node eslint.Node) bool {
		switch eslint.NodeType(node) {
		case "DoWhileStatement", "BreakStatement", "ContinueStatement",
			"DebuggerStatement", "ImportDeclaration", "ExportAllDeclaration":
			return false
		case "ReturnStatement":
			return eslint.GetNode(node, "argument") != nil
		case "ExportNamedDeclaration":
			return eslint.GetNode(node, "declaration") != nil
		}
		if isEndOfArrowBlock(sc.GetLastToken(node, eslint.TokenOpt{Skip: 1})) {
			return false
		}
		return true
	}

	// canRemoveSemicolon: whether the semicolon of the given node is
	// unnecessary under `never`.
	canRemoveSemicolon := func(node eslint.Node) bool {
		if isRedundantSemi(sc.GetLastToken(node)) {
			return true // `;;` or `;}`
		}
		if maybeClassFieldAsiHazard(node) {
			return false
		}
		if isOnSameLineWithNextToken(node) {
			return false // One liner.
		}

		// continuation characters should not apply to class fields
		if eslint.NodeType(node) != "PropertyDefinition" &&
			beforeStatementContinuationChars == "never" &&
			!maybeAsiHazardAfter(node) {
			return true // ASI works. This statement doesn't connect to the next.
		}
		if !maybeAsiHazardBefore(sc.GetTokenAfter(node)) {
			return true // ASI works. The next token doesn't connect to this statement.
		}

		return false
	}

	// isLastInOneLinerBlock: the node is the last item of a one-liner block
	// (BlockStatement or StaticBlock).
	isLastInOneLinerBlock := func(node eslint.Node) bool {
		parent := eslint.Parent(node)
		nextToken := sc.GetTokenAfter(node)

		if nextToken == nil || eslint.TokenValue(nextToken) != "}" {
			return false
		}

		switch eslint.NodeType(parent) {
		case "BlockStatement":
			start, _ := eslint.LocStart(parent)
			end, _ := eslint.LocEnd(parent)
			return start == end
		case "StaticBlock":
			openingBrace := sc.GetFirstToken(parent, eslint.TokenOpt{Skip: 1}) // skip the `static` token
			start, _ := eslint.LocStart(openingBrace)
			end, _ := eslint.LocEnd(parent)
			return start == end
		}

		return false
	}

	// isLastInOneLinerClassBody: the node is the last item of a one-liner
	// ClassBody.
	isLastInOneLinerClassBody := func(node eslint.Node) bool {
		parent := eslint.Parent(node)
		nextToken := sc.GetTokenAfter(node)

		if nextToken == nil || eslint.TokenValue(nextToken) != "}" {
			return false
		}

		if eslint.NodeType(parent) == "ClassBody" {
			start, _ := eslint.LocStart(parent)
			end, _ := eslint.LocEnd(parent)
			return start == end
		}

		return false
	}

	// checkForSemicolon.
	checkForSemicolon := func(node eslint.Node) {
		isSemi := eslint.IsSemicolonToken(sc.GetLastToken(node))

		if never {
			if isSemi && canRemoveSemicolon(node) {
				report(node, true)
			} else if !isSemi && beforeStatementContinuationChars == "always" &&
				eslint.NodeType(node) != "PropertyDefinition" &&
				maybeAsiHazardBefore(sc.GetTokenAfter(node)) {
				report(node, false)
			}
			return
		}

		oneLinerBlock := exceptOneLine && isLastInOneLinerBlock(node)
		oneLinerClassBody := exceptOneLineClassBody && isLastInOneLinerClassBody(node)
		oneLinerBlockOrClassBody := oneLinerBlock || oneLinerClassBody

		if isSemi && oneLinerBlockOrClassBody {
			report(node, true)
		} else if !isSemi && !oneLinerBlockOrClassBody {
			report(node, false)
		}
	}

	// checkForSemicolonForVariableDeclaration.
	checkForSemicolonForVariableDeclaration := func(node eslint.Node) {
		parent := eslint.Parent(node)
		parentType := eslint.NodeType(parent)

		inForInit := parentType == "ForStatement" &&
			eslint.SameNode(eslint.GetNode(parent, "init"), node)
		inForLeft := (parentType == "ForInStatement" || parentType == "ForOfStatement") &&
			eslint.SameNode(eslint.GetNode(parent, "left"), node)

		if !inForInit && !inForLeft {
			checkForSemicolon(node)
		}
	}

	return map[string]func(eslint.Node){
		"VariableDeclaration":  checkForSemicolonForVariableDeclaration,
		"ExpressionStatement":  checkForSemicolon,
		"ReturnStatement":      checkForSemicolon,
		"ThrowStatement":       checkForSemicolon,
		"DoWhileStatement":     checkForSemicolon,
		"DebuggerStatement":    checkForSemicolon,
		"BreakStatement":       checkForSemicolon,
		"ContinueStatement":    checkForSemicolon,
		"ImportDeclaration":    checkForSemicolon,
		"ExportAllDeclaration": checkForSemicolon,
		"ExportNamedDeclaration": func(node eslint.Node) {
			if eslint.GetNode(node, "declaration") == nil {
				checkForSemicolon(node)
			}
		},
		"ExportDefaultDeclaration": func(node eslint.Node) {
			declType := eslint.NodeType(eslint.GetNode(node, "declaration"))
			if declType != "ClassDeclaration" && declType != "FunctionDeclaration" {
				checkForSemicolon(node)
			}
		},
		"PropertyDefinition": checkForSemicolon,
	}
}
