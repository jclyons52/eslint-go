// Package curly is a port of eslint/lib/rules/curly.js.
//
// The rule enforces consistent brace style for all control statements. Every
// reported problem carries a code fix, so the fix ranges (and the fixes that
// deliberately decline to apply) are as much part of the specification as the
// messages are.
package curly

import (
	"regexp"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "curly"

// Rule is a port of eslint/lib/rules/curly.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "suggestion",
		Docs: eslint.RuleDocs{
			Description: "Enforce consistent brace style for all control statements",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/curly",
		},
		Fixable: "code",
		Schema: []any{
			map[string]any{
				"anyOf": []any{
					map[string]any{
						"type": "array",
						"items": []any{
							map[string]any{"enum": []any{"all"}},
						},
						"minItems": 0,
						"maxItems": 1,
					},
					map[string]any{
						"type": "array",
						"items": []any{
							map[string]any{"enum": []any{"multi", "multi-line", "multi-or-nest"}},
							map[string]any{"enum": []any{"consistent"}},
						},
						"minItems": 0,
						"maxItems": 2,
					},
				},
			},
		},
		Messages: map[string]string{
			"missingCurlyAfter":             "Expected { after '{{name}}'.",
			"missingCurlyAfterCondition":    "Expected { after '{{name}}' condition.",
			"unexpectedCurlyAfter":          "Unnecessary { after '{{name}}'.",
			"unexpectedCurlyAfterCondition": "Unnecessary { after '{{name}}' condition.",
		},
	},
	Create: create,
}

// unsafeTokenStart is the JS /^[([/`+-]/u test the rule uses to decide whether a
// following token could disrupt ASI.
var unsafeTokenStart = regexp.MustCompile("^[([/`+-]")

// preparedCheck mirrors the object prepareCheck() returns: the current state of
// one body, the state the rule wants, and the check() that reports on it.
type preparedCheck struct {
	ctx       *eslint.Context
	sc        *eslint.SourceCode
	node      eslint.Node
	body      eslint.Node
	name      string
	condition bool
	actual    bool
	// expected is nil for the JS `null` ("either is fine").
	expected *bool
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	multiOnly := ctx.OptionString(0, "") == "multi"
	multiLine := ctx.OptionString(0, "") == "multi-line"
	multiOrNest := ctx.OptionString(0, "") == "multi-or-nest"
	consistent := ctx.OptionString(1, "") == "consistent"

	sc := ctx.SourceCode

	// isCollapsedOneLiner: a one-liner on the same line as the code before it.
	isCollapsedOneLiner := func(node eslint.Node) bool {
		before := sc.GetTokenBefore(node)
		last := sc.GetLastToken(node)
		if eslint.IsSemicolonToken(last) {
			last = sc.GetTokenBefore(last)
		}
		beforeLine, _ := eslint.LocStart(before)
		lastEndLine, _ := eslint.LocEnd(last)
		return beforeLine == lastEndLine
	}

	// isOneLiner: first token and last (semicolon-excluded) token on one line.
	isOneLiner := func(node eslint.Node) bool {
		if eslint.NodeType(node) == "EmptyStatement" {
			return true
		}
		first := sc.GetFirstToken(node)
		last := sc.GetLastToken(node)
		if eslint.IsSemicolonToken(last) {
			last = sc.GetTokenBefore(last)
		}
		firstLine, _ := eslint.LocStart(first)
		lastEndLine, _ := eslint.LocEnd(last)
		return firstLine == lastEndLine
	}

	// isLexicalDeclaration: curly's own narrow version (let/const/function/class).
	isLexicalDeclaration := func(node eslint.Node) bool {
		if eslint.NodeType(node) == "VariableDeclaration" {
			kind := eslint.GetString(node, "kind")
			return kind == "const" || kind == "let"
		}
		t := eslint.NodeType(node)
		return t == "FunctionDeclaration" || t == "ClassDeclaration"
	}

	isElseKeywordToken := func(token eslint.Node) bool {
		return eslint.TokenValue(token) == "else" && eslint.NodeType(token) == eslint.TokenKeyword
	}

	isFollowedByElseKeyword := func(node eslint.Node) bool {
		next := sc.GetTokenAfter(node)
		return next != nil && isElseKeywordToken(next)
	}

	// needsSemicolon: whether removing the braces would need a semicolon
	// inserted to avoid a SyntaxError or an ASI behaviour change.
	needsSemicolon := func(closingBracket eslint.Node) bool {
		tokenBefore := sc.GetTokenBefore(closingBracket)
		tokenAfter := sc.GetTokenAfter(closingBracket)
		lastBlockNode := sc.GetNodeByRangeIndex(eslint.Start(tokenBefore))

		if eslint.IsSemicolonToken(tokenBefore) {
			// The last statement already ends in a semicolon.
			return false
		}
		if tokenAfter == nil {
			// Nothing follows the block, so no semicolon is needed.
			return false
		}
		if eslint.NodeType(lastBlockNode) == "BlockStatement" {
			parentType := eslint.NodeType(eslint.Parent(lastBlockNode))
			if parentType != "FunctionExpression" && parentType != "ArrowFunctionExpression" {
				/*
				 * The last node inside the braces is itself a block (not a
				 * function body): a semicolon there would parse as a separate
				 * statement.
				 */
				return false
			}
		}
		beforeEndLine, _ := eslint.LocEnd(tokenBefore)
		afterStartLine, _ := eslint.LocStart(tokenAfter)
		if beforeEndLine == afterStartLine {
			return true
		}
		if unsafeTokenStart.MatchString(eslint.TokenValue(tokenAfter)) {
			return true
		}
		if eslint.NodeType(tokenBefore) == eslint.TokenPunctuator {
			v := eslint.TokenValue(tokenBefore)
			if v == "++" || v == "--" {
				return true
			}
		}
		return false
	}

	// hasUnsafeIf: whether the code would grab an `else` appended to it.
	var hasUnsafeIf func(node eslint.Node) bool
	hasUnsafeIf = func(node eslint.Node) bool {
		switch eslint.NodeType(node) {
		case "IfStatement":
			alt := eslint.GetNode(node, "alternate")
			if alt == nil {
				return true
			}
			return hasUnsafeIf(alt)
		case "ForStatement", "ForInStatement", "ForOfStatement",
			"LabeledStatement", "WithStatement", "WhileStatement":
			return hasUnsafeIf(eslint.GetNode(node, "body"))
		}
		return false
	}

	// areBracesNecessary: braces that must stay to preserve semantics.
	areBracesNecessary := func(node eslint.Node) bool {
		body := eslint.GetNodes(node, "body")
		if len(body) == 0 {
			return false
		}
		statement := body[0]
		return isLexicalDeclaration(statement) ||
			hasUnsafeIf(statement) && isFollowedByElseKeyword(node)
	}

	prepareCheck := func(node, body eslint.Node, name string, condition bool) *preparedCheck {
		hasBlock := eslint.NodeType(body) == "BlockStatement"
		var expected *bool

		setTrue := func() { v := true; expected = &v }
		setFalse := func() { v := false; expected = &v }

		switch {
		case hasBlock && (len(eslint.GetNodes(body, "body")) != 1 || areBracesNecessary(body)):
			setTrue()
		case multiOnly:
			setFalse()
		case multiLine:
			if !isCollapsedOneLiner(body) {
				setTrue()
			}
			// Otherwise braces are optional for this body.
		case multiOrNest:
			if hasBlock {
				statement := eslint.GetNodes(body, "body")[0]
				leadingCommentsInBlock := sc.GetCommentsBefore(statement)
				if !isOneLiner(statement) || len(leadingCommentsInBlock) > 0 {
					setTrue()
				} else {
					setFalse()
				}
			} else if !isOneLiner(body) {
				setTrue()
			} else {
				setFalse()
			}
		default:
			// default "all"
			setTrue()
		}

		return &preparedCheck{
			ctx: ctx, sc: sc, node: node, body: body, name: name,
			condition: condition, actual: hasBlock, expected: expected,
		}
	}

	prepareIfChecks := func(node eslint.Node) []*preparedCheck {
		var preparedChecks []*preparedCheck

		for currentNode := node; currentNode != nil; currentNode = eslint.GetNode(currentNode, "alternate") {
			preparedChecks = append(preparedChecks,
				prepareCheck(currentNode, eslint.GetNode(currentNode, "consequent"), "if", true))
			alt := eslint.GetNode(currentNode, "alternate")
			if alt != nil && eslint.NodeType(alt) != "IfStatement" {
				preparedChecks = append(preparedChecks, prepareCheck(currentNode, alt, "else", false))
				break
			}
		}

		if consistent {
			/*
			 * If any body should have (or already has) braces, make sure they
			 * all do; if none should, make sure none do.
			 */
			expected := false
			for _, pc := range preparedChecks {
				if pc.expected != nil {
					if *pc.expected {
						expected = true
					}
				} else if pc.actual {
					expected = true
				}
			}
			for _, pc := range preparedChecks {
				v := expected
				pc.expected = &v
			}
		}

		return preparedChecks
	}

	// check reports the body if the rule's expectation differs from reality.
	check := func(pc *preparedCheck) {
		if pc.expected == nil || *pc.expected == pc.actual {
			return
		}
		if *pc.expected {
			messageID := "missingCurlyAfter"
			if pc.condition {
				messageID = "missingCurlyAfterCondition"
			}
			body := pc.body
			ctx.Report(eslint.Report{
				Node:      pc.node,
				Loc:       eslint.Loc(body),
				MessageID: messageID,
				Data:      map[string]any{"name": pc.name},
				Fix: func(f *eslint.Fixer) *eslint.Fix {
					return f.ReplaceText(body, "{"+sc.GetText(body)+"}")
				},
			})
			return
		}

		messageID := "unexpectedCurlyAfter"
		if pc.condition {
			messageID = "unexpectedCurlyAfterCondition"
		}
		node, body, name := pc.node, pc.body, pc.name
		ctx.Report(eslint.Report{
			Node:      node,
			Loc:       eslint.Loc(body),
			MessageID: messageID,
			Data:      map[string]any{"name": name},
			Fix: func(f *eslint.Fixer) *eslint.Fix {
				/*
				 * `do while` sometimes needs a space after `do`, e.g.
				 * `do{foo()} while (bar)` must become `do foo() while (bar)`.
				 */
				needsPrecedingSpace := false
				if eslint.NodeType(node) == "DoWhileStatement" {
					before := sc.GetTokenBefore(body)
					first := sc.GetFirstToken(body, eslint.TokenOpt{Skip: 1})
					needsPrecedingSpace = before != nil && eslint.End(before) == eslint.Start(body) &&
						!canTokensBeAdjacent("do", first)
				}

				openingBracket := sc.GetFirstToken(body)
				closingBracket := sc.GetLastToken(body)

				if needsSemicolon(closingBracket) {
					/*
					 * Removing the braces would need a semicolon to be inserted
					 * (multiple statements on one line, or ASI), so don't fix.
					 */
					return nil
				}

				// The body text with the braces stripped.
				resultingBodyText := sc.GetTextRange([2]int{eslint.End(openingBracket), eslint.Start(closingBracket)})
				if needsPrecedingSpace {
					resultingBodyText = " " + resultingBodyText
				}
				return f.ReplaceText(body, resultingBodyText)
			},
		})
	}

	return map[string]func(eslint.Node){
		"IfStatement": func(node eslint.Node) {
			parent := eslint.Parent(node)
			isElseIf := eslint.NodeType(parent) == "IfStatement" &&
				eslint.SameNode(eslint.GetNode(parent, "alternate"), node)

			if !isElseIf {
				// A top `if`: check the whole `if`-`else if`-`else` chain.
				for _, pc := range prepareIfChecks(node) {
					check(pc)
				}
			}
			// `else if` is skipped — the top `if` already checked it.
		},

		"WhileStatement": func(node eslint.Node) {
			check(prepareCheck(node, eslint.GetNode(node, "body"), "while", true))
		},

		"DoWhileStatement": func(node eslint.Node) {
			check(prepareCheck(node, eslint.GetNode(node, "body"), "do", false))
		},

		"ForStatement": func(node eslint.Node) {
			check(prepareCheck(node, eslint.GetNode(node, "body"), "for", true))
		},

		"ForInStatement": func(node eslint.Node) {
			check(prepareCheck(node, eslint.GetNode(node, "body"), "for-in", false))
		},

		"ForOfStatement": func(node eslint.Node) {
			check(prepareCheck(node, eslint.GetNode(node, "body"), "for-of", false))
		},
	}
}

// canTokensBeAdjacent is a port of astUtils.canTokensBeAdjacent for the shape
// the ported rules use it in: a string left value (tokenized with espree) and a
// real token on the right. Only the last token of the left string matters, and
// for the identifier-shaped strings the callers pass, that token is never a
// punctuator, string, template or numeric token — so the outcome is decided by
// the right token (with the one exception of the `/` punctuator, kept here for
// faithfulness).
func canTokensBeAdjacent(leftValue string, rightToken eslint.Node) bool {
	if rightToken == nil {
		return false
	}
	leftType := identifierTokenType(leftValue)
	rightType := eslint.NodeType(rightToken)

	if leftType == eslint.TokenPunctuator || rightType == eslint.TokenPunctuator {
		if leftType == eslint.TokenPunctuator && rightType == eslint.TokenPunctuator {
			plus := map[string]bool{"+": true, "++": true}
			minus := map[string]bool{"-": true, "--": true}
			rv := eslint.TokenValue(rightToken)
			return !((plus[leftValue] && plus[rv]) || (minus[leftValue] && minus[rv]))
		}
		if leftType == eslint.TokenPunctuator && leftValue == "/" {
			switch rightType {
			case "Block", "Line", eslint.TokenRegularExpression:
				return false
			}
			return true
		}
		return true
	}

	if leftType == eslint.TokenString || rightType == eslint.TokenString ||
		leftType == eslint.TokenTemplate || rightType == eslint.TokenTemplate {
		return true
	}

	if leftType != eslint.TokenNumeric && rightType == eslint.TokenNumeric &&
		strings.HasPrefix(eslint.TokenValue(rightToken), ".") {
		return true
	}

	if leftType == "Block" || rightType == "Block" || rightType == "Line" {
		return true
	}

	if rightType == eslint.TokenPrivateIdent {
		return true
	}

	return false
}

// identifierTokenType reports the espree token type the given source text
// tokenizes to. Callers only ever pass an identifier-shaped string.
func identifierTokenType(text string) string {
	switch text {
	case "true", "false":
		return eslint.TokenBoolean
	case "null":
		return eslint.TokenNull
	}
	if jsKeywords[text] {
		return eslint.TokenKeyword
	}
	return eslint.TokenIdentifier
}

// jsKeywords is acorn/espree's keyword set (the same 56 ES3 keywords the
// dot-notation rule checks against).
var jsKeywords = map[string]bool{
	"abstract": true, "boolean": true, "break": true, "byte": true, "case": true,
	"catch": true, "char": true, "class": true, "const": true, "continue": true,
	"debugger": true, "default": true, "delete": true, "do": true, "double": true,
	"else": true, "enum": true, "export": true, "extends": true, "false": true,
	"final": true, "finally": true, "float": true, "for": true, "function": true,
	"goto": true, "if": true, "implements": true, "import": true, "in": true,
	"instanceof": true, "int": true, "interface": true, "long": true, "native": true,
	"new": true, "null": true, "package": true, "private": true, "protected": true,
	"public": true, "return": true, "short": true, "static": true, "super": true,
	"switch": true, "synchronized": true, "this": true, "throw": true, "throws": true,
	"transient": true, "true": true, "try": true, "typeof": true, "var": true,
	"void": true, "volatile": true, "while": true, "with": true,
}
