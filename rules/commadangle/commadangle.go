package commadangle

import (
	"sort"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
)

// Name is the ESLint rule id.
const Name = "comma-dangle"

// Rule is a port of eslint/lib/rules/comma-dangle.js.
var Rule = eslint.Rule{
	ID: Name,
	Meta: eslint.RuleMeta{
		Type: "layout",
		Docs: eslint.RuleDocs{
			Description: "Require or disallow trailing commas",
			Recommended: false,
			URL:         "https://eslint.org/docs/latest/rules/comma-dangle",
		},
		Fixable: "code",
		Schema: []any{
			map[string]any{
				"definitions": map[string]any{
					"value": map[string]any{
						"enum": []any{"always-multiline", "always", "never", "only-multiline"},
					},
					"valueWithIgnore": map[string]any{
						"enum": []any{"always-multiline", "always", "ignore", "never", "only-multiline"},
					},
				},
				"type": "array",
				"items": []any{
					map[string]any{
						"oneOf": []any{
							map[string]any{"$ref": "#/definitions/value"},
							map[string]any{
								"type": "object",
								"properties": map[string]any{
									"arrays":    map[string]any{"$ref": "#/definitions/valueWithIgnore"},
									"objects":   map[string]any{"$ref": "#/definitions/valueWithIgnore"},
									"imports":   map[string]any{"$ref": "#/definitions/valueWithIgnore"},
									"exports":   map[string]any{"$ref": "#/definitions/valueWithIgnore"},
									"functions": map[string]any{"$ref": "#/definitions/valueWithIgnore"},
								},
								"additionalProperties": false,
							},
						},
					},
				},
				"additionalItems": false,
			},
		},
		Messages: map[string]string{
			"unexpected": "Unexpected trailing comma.",
			"missing":    "Missing trailing comma.",
		},
	},
	Create: create,
}

// Option values (the rule's schema enum).
const (
	optAlways          = "always"
	optAlwaysMultiline = "always-multiline"
	optOnlyMultiline   = "only-multiline"
	optNever           = "never"
	optIgnore          = "ignore"
)

// options is the rule's normalized option object.
type options struct {
	arrays    string
	objects   string
	imports   string
	exports   string
	functions string
}

// defaultOptions mirrors DEFAULT_OPTIONS.
var defaultOptions = options{
	arrays:    optNever,
	objects:   optNever,
	imports:   optNever,
	exports:   optNever,
	functions: optNever,
}

// normalizeOptions mirrors the rule's normalizeOptions().
func normalizeOptions(optionValue any, ecmaVersion int) options {
	if s, ok := optionValue.(string); ok {
		functions := s
		if ecmaVersion < 2017 {
			functions = optIgnore
		}
		return options{arrays: s, objects: s, imports: s, exports: s, functions: functions}
	}
	if m, ok := optionValue.(map[string]any); ok && m != nil {
		return options{
			arrays:    stringOr(m["arrays"], defaultOptions.arrays),
			objects:   stringOr(m["objects"], defaultOptions.objects),
			imports:   stringOr(m["imports"], defaultOptions.imports),
			exports:   stringOr(m["exports"], defaultOptions.exports),
			functions: stringOr(m["functions"], defaultOptions.functions),
		}
	}
	return defaultOptions
}

// stringOr is JS's `a || b` for the option strings (an empty/absent/invalid
// value falls back to the default).
func stringOr(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

// ecmaVersion mirrors `context.languageOptions.ecmaVersion`: the raw
// parserOptions value, with "latest" resolving the way eslint 8.57 does.
func ecmaVersion(ctx *eslint.Context) int {
	switch v := ctx.ParserOptions["ecmaVersion"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if v == "latest" {
			return 2024
		}
	}
	return 2022
}

func create(ctx *eslint.Context) map[string]func(eslint.Node) {
	sc := ctx.SourceCode
	opts := normalizeOptions(ctx.Option(0), ecmaVersion(ctx))

	out := map[string]func(eslint.Node){}
	add := func(nodeType, mode string) {
		switch mode {
		case optAlways:
			out[nodeType] = func(node eslint.Node) { forceTrailingComma(ctx, sc, node) }
		case optAlwaysMultiline:
			out[nodeType] = func(node eslint.Node) { forceTrailingCommaIfMultiline(ctx, sc, node) }
		case optOnlyMultiline:
			out[nodeType] = func(node eslint.Node) { allowTrailingCommaIfMultiline(ctx, sc, node) }
		case optNever:
			out[nodeType] = func(node eslint.Node) { forbidTrailingComma(ctx, sc, node) }
		case optIgnore:
			out[nodeType] = func(eslint.Node) {}
		}
	}

	add("ObjectExpression", opts.objects)
	add("ObjectPattern", opts.objects)

	add("ArrayExpression", opts.arrays)
	add("ArrayPattern", opts.arrays)

	add("ImportDeclaration", opts.imports)

	add("ExportNamedDeclaration", opts.exports)

	add("FunctionDeclaration", opts.functions)
	add("FunctionExpression", opts.functions)
	add("ArrowFunctionExpression", opts.functions)
	add("CallExpression", opts.functions)
	add("NewExpression", opts.functions)

	return out
}

// isTrailingCommaAllowed reports whether a trailing comma may follow the last
// item (it may not after a rest element/property).
func isTrailingCommaAllowed(lastItem eslint.Node) bool {
	switch eslint.NodeType(lastItem) {
	case "RestElement", "RestProperty", "ExperimentalRestProperty":
		return false
	}
	return true
}

// getLastItem mirrors the rule's getLastItem(): the raw last array element
// (null when the list is empty or ends in a hole, which is what JS's
// `array[array.length - 1]` yields).
func getLastItem(node eslint.Node) eslint.Node {
	var key string
	switch eslint.NodeType(node) {
	case "ObjectExpression", "ObjectPattern":
		key = "properties"
	case "ArrayExpression", "ArrayPattern":
		key = "elements"
	case "ImportDeclaration", "ExportNamedDeclaration":
		key = "specifiers"
	case "FunctionDeclaration", "FunctionExpression", "ArrowFunctionExpression":
		key = "params"
	case "CallExpression", "NewExpression":
		key = "arguments"
	default:
		return nil
	}
	arr, _ := eslint.Get(node, key).([]any)
	if len(arr) == 0 {
		return nil
	}
	last, _ := arr[len(arr)-1].(eslint.Node)
	return last
}

// getTrailingToken mirrors the rule's getTrailingToken(): the trailing comma
// token, or the token the comma would be inserted after.
func getTrailingToken(sc *eslint.SourceCode, node, lastItem eslint.Node) eslint.Node {
	switch eslint.NodeType(node) {
	case "ObjectExpression", "ArrayExpression", "CallExpression", "NewExpression":
		return sc.GetLastToken(node, eslint.TokenOpt{Skip: 1})
	default:
		nextToken := sc.GetTokenAfter(lastItem)
		if eslint.IsCommaToken(nextToken) {
			return nextToken
		}
		return sc.GetLastToken(lastItem)
	}
}

// isMultiline mirrors the rule's isMultiline(): the node is multiline when the
// token after the trailing token is not on the same line as it.
func isMultiline(sc *eslint.SourceCode, node eslint.Node) bool {
	lastItem := getLastItem(node)
	if lastItem == nil {
		return false
	}
	penultimateToken := getTrailingToken(sc, node, lastItem)
	lastToken := sc.GetTokenAfter(penultimateToken)
	penLine, _ := eslint.LocEnd(penultimateToken)
	lastLine, _ := eslint.LocEnd(lastToken)
	return lastLine != penLine
}

// forbidTrailingComma mirrors the rule's forbidTrailingComma().
func forbidTrailingComma(ctx *eslint.Context, sc *eslint.SourceCode, node eslint.Node) {
	lastItem := getLastItem(node)
	if lastItem == nil ||
		(eslint.NodeType(node) == "ImportDeclaration" && eslint.NodeType(lastItem) != "ImportSpecifier") {
		return
	}

	trailingToken := getTrailingToken(sc, node, lastItem)
	if !eslint.IsCommaToken(trailingToken) {
		return
	}

	cmds := []fixCmd{
		{r: eslint.MustRange(trailingToken), text: ""},
		{r: pointRange(sc.GetTokenBefore(trailingToken)), text: ""},
		{r: endPointRange(sc.GetTokenAfter(trailingToken)), text: ""},
	}
	ctx.Report(eslint.Report{
		Node:      lastItem,
		Loc:       eslint.Loc(trailingToken),
		MessageID: "unexpected",
		Fix: func(f *eslint.Fixer) *eslint.Fix {
			return mergeFixes(f, sc, cmds)
		},
	})
}

// forceTrailingComma mirrors the rule's forceTrailingComma().
func forceTrailingComma(ctx *eslint.Context, sc *eslint.SourceCode, node eslint.Node) {
	lastItem := getLastItem(node)
	if lastItem == nil ||
		(eslint.NodeType(node) == "ImportDeclaration" && eslint.NodeType(lastItem) != "ImportSpecifier") {
		return
	}
	if !isTrailingCommaAllowed(lastItem) {
		forbidTrailingComma(ctx, sc, node)
		return
	}

	trailingToken := getTrailingToken(sc, node, lastItem)
	if eslint.TokenValue(trailingToken) == "," {
		return
	}

	loc := reportLoc(sc, trailingToken)
	cmds := []fixCmd{
		{r: endPointRange(trailingToken), text: ","},
		{r: pointRange(trailingToken), text: ""},
		{r: endPointRange(sc.GetTokenAfter(trailingToken)), text: ""},
	}
	ctx.Report(eslint.Report{
		Node:      lastItem,
		Loc:       loc,
		MessageID: "missing",
		Fix: func(f *eslint.Fixer) *eslint.Fix {
			return mergeFixes(f, sc, cmds)
		},
	})
}

// forceTrailingCommaIfMultiline mirrors the rule's forceTrailingCommaIfMultiline().
func forceTrailingCommaIfMultiline(ctx *eslint.Context, sc *eslint.SourceCode, node eslint.Node) {
	if isMultiline(sc, node) {
		forceTrailingComma(ctx, sc, node)
	} else {
		forbidTrailingComma(ctx, sc, node)
	}
}

// allowTrailingCommaIfMultiline mirrors the rule's allowTrailingCommaIfMultiline().
func allowTrailingCommaIfMultiline(ctx *eslint.Context, sc *eslint.SourceCode, node eslint.Node) {
	if !isMultiline(sc, node) {
		forbidTrailingComma(ctx, sc, node)
	}
}

// reportLoc builds the "missing" report's loc: from the trailing token's end to
// the next location (astUtils.getNextLocation), which is null at end of file.
func reportLoc(sc *eslint.SourceCode, trailingToken eslint.Node) map[string]any {
	line, col := eslint.LocEnd(trailingToken)
	loc := map[string]any{
		"start": map[string]any{"line": float64(line), "column": float64(col)},
	}
	nextLine, nextCol, ok := getNextLocation(sc, line, col)
	if ok {
		loc["end"] = map[string]any{"line": float64(nextLine), "column": float64(nextCol)}
	} else {
		loc["end"] = nil
	}
	return loc
}

// getNextLocation mirrors astUtils.getNextLocation.
func getNextLocation(sc *eslint.SourceCode, line, col int) (int, int, bool) {
	lines := sc.Lines()
	if line >= 1 && line <= len(lines) && col < len(lines[line-1]) {
		return line, col + 1, true
	}
	if line < len(lines) {
		return line + 1, 0, true
	}
	return 0, 0, false
}

// pointRange is the zero-width range at a node's start (insertTextBefore).
func pointRange(node eslint.Node) [2]int {
	s := eslint.Start(node)
	return [2]int{s, s}
}

// endPointRange is the zero-width range at a node's end (insertTextAfter).
func endPointRange(node eslint.Node) [2]int {
	e := eslint.End(node)
	return [2]int{e, e}
}

// fixCmd is one fix a report contributes.
type fixCmd struct {
	r    [2]int
	text string
}

// mergeFixes is report-translator's mergeFixes(): the rule yields three fixes
// (remove the comma, and zero-width edits around it) and ESLint merges them
// into one fix whose range spans all of them, with the source between them
// preserved. The merged range is part of the emitted message, so it has to
// match exactly.
func mergeFixes(f *eslint.Fixer, sc *eslint.SourceCode, cmds []fixCmd) *eslint.Fix {
	if len(cmds) == 0 {
		return nil
	}
	sorted := append([]fixCmd(nil), cmds...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].r[0] != sorted[j].r[0] {
			return sorted[i].r[0] < sorted[j].r[0]
		}
		return sorted[i].r[1] < sorted[j].r[1]
	})
	if len(sorted) == 1 {
		return f.ReplaceTextRange(sorted[0].r, sorted[0].text)
	}

	original := sc.Text()
	start := sorted[0].r[0]
	end := sorted[len(sorted)-1].r[1]
	var sb strings.Builder
	lastPos := -1 << 62 // Number.MIN_SAFE_INTEGER
	for _, c := range sorted {
		if c.r[0] >= 0 {
			sb.WriteString(sliceText(original, maxOf(0, start, lastPos), c.r[0]))
		}
		sb.WriteString(c.text)
		lastPos = c.r[1]
	}
	sb.WriteString(sliceText(original, maxOf(0, start, lastPos), end))
	return f.ReplaceTextRange([2]int{start, end}, sb.String())
}

// sliceText is JS's String.prototype.slice(a, b) for non-negative offsets.
func sliceText(s string, a, b int) string {
	if a < 0 {
		a = 0
	}
	if b < 0 {
		b = 0
	}
	if a > len(s) {
		a = len(s)
	}
	if b > len(s) {
		b = len(s)
	}
	if a >= b {
		return ""
	}
	return s[a:b]
}

func maxOf(vs ...int) int {
	m := vs[0]
	for _, v := range vs[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
