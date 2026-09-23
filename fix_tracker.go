package eslint

// fix_tracker.go — a port of ESLint's rules/utils/fix-tracker.js. It widens a
// fix's range to cover a "retained" region so that two rules' fixes in the same
// pass cannot both touch that region (the mechanism ESLint uses to stop, e.g.,
// no-extra-semi from fighting with semi).

// FixTracker combines fix options into one fix command, retaining a region
// that other fixes may not modify in the same pass.
type FixTracker struct {
	fixer         *Fixer
	sourceCode    *SourceCode
	retainedRange *[2]int
}

// NewFixTracker builds a tracker bound to a fixer and source code.
func NewFixTracker(fixer *Fixer, sourceCode *SourceCode) *FixTracker {
	return &FixTracker{fixer: fixer, sourceCode: sourceCode}
}

// RetainRange marks a range as retained.
func (ft *FixTracker) RetainRange(r [2]int) *FixTracker {
	ft.retainedRange = &r
	return ft
}

// RetainEnclosingFunction marks the function containing node (or the whole
// program) as retained.
func (ft *FixTracker) RetainEnclosingFunction(node Node) *FixTracker {
	if fn := GetUpperFunction(node); fn != nil {
		return ft.RetainRange(MustRange(fn))
	}
	return ft.RetainRange(MustRange(ft.sourceCode.AST()))
}

// RetainSurroundingTokens marks the span of the neighbouring tokens as
// retained, so a small edit cannot collide with a fix that wants to reflow the
// surrounding syntax.
func (ft *FixTracker) RetainSurroundingTokens(nodeOrToken Node) *FixTracker {
	before := ft.sourceCode.GetTokenBefore(nodeOrToken)
	if before == nil {
		before = nodeOrToken
	}
	after := ft.sourceCode.GetTokenAfter(nodeOrToken)
	if after == nil {
		after = nodeOrToken
	}
	return ft.RetainRange([2]int{Start(before), End(after)})
}

// ReplaceTextRange replaces a range with text, expanded over any retained range
// with the retained text preserved.
func (ft *FixTracker) ReplaceTextRange(r [2]int, text string) *Fix {
	actual := r
	if ft.retainedRange != nil {
		actual = [2]int{minInt(ft.retainedRange[0], r[0]), maxInt(ft.retainedRange[1], r[1])}
	}
	expanded := ft.sourceCode.GetTextRange([2]int{actual[0], r[0]}) +
		text +
		ft.sourceCode.GetTextRange([2]int{r[1], actual[1]})
	return ft.fixer.ReplaceTextRange(actual, expanded)
}

// Remove removes a node or token, accounting for retained ranges.
func (ft *FixTracker) Remove(nodeOrToken Node) *Fix {
	return ft.ReplaceTextRange(MustRange(nodeOrToken), "")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
