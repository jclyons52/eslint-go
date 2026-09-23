package eslint

import "sort"

// fixer.go — ESLint's rule-fixer (the object a rule uses to build a fix) and
// source-code-fixer's applyFixes() (which applies non-overlapping fixes to the
// text). Both are ported to match the originals, because `--fix` output has to
// be byte-identical to ESLint's.

// Fixer mirrors ESLint's ruleFixer.
type Fixer struct {
	sc *SourceCode
}

// SourceCode exposes the source code the fixer is bound to.
func (f *Fixer) SourceCode() *SourceCode { return f.sc }

// InsertTextAfter inserts text after a node or token.
func (f *Fixer) InsertTextAfter(node Node, text string) *Fix {
	return f.insertTextAt(End(node), text)
}

// InsertTextAfterRange inserts text after an offset range.
func (f *Fixer) InsertTextAfterRange(r [2]int, text string) *Fix {
	return f.insertTextAt(r[1], text)
}

// InsertTextBefore inserts text before a node or token.
func (f *Fixer) InsertTextBefore(node Node, text string) *Fix {
	return f.insertTextAt(Start(node), text)
}

// InsertTextBeforeRange inserts text before an offset range.
func (f *Fixer) InsertTextBeforeRange(r [2]int, text string) *Fix {
	return f.insertTextAt(r[0], text)
}

// ReplaceText replaces a node or token's text.
func (f *Fixer) ReplaceText(node Node, text string) *Fix {
	return f.ReplaceTextRange(MustRange(node), text)
}

// ReplaceTextRange replaces the text in an offset range.
func (f *Fixer) ReplaceTextRange(r [2]int, text string) *Fix {
	return f.withUnits(&Fix{Range: r, Text: text})
}

// Remove removes a node or token.
func (f *Fixer) Remove(node Node) *Fix {
	return f.RemoveRange(MustRange(node))
}

// RemoveRange removes the text in an offset range.
func (f *Fixer) RemoveRange(r [2]int) *Fix {
	return f.withUnits(&Fix{Range: r, Text: ""})
}

// InsertTextAfterRangeOf inserts text after a range given as two offsets.
func (f *Fixer) InsertTextAfterRangeOf(start, end int, text string) *Fix {
	return f.insertTextAt(end, text)
}

func (f *Fixer) insertTextAt(index int, text string) *Fix {
	return f.withUnits(&Fix{Range: [2]int{index, index}, Text: text})
}

// withUnits records the UTF-16 code-unit range for a fix when the source is not
// pure ASCII, so the emitted message carries ESLint's range while the fixer
// keeps applying byte ranges.
func (f *Fixer) withUnits(fix *Fix) *Fix {
	if f.sc == nil || !f.sc.NonASCII() {
		return fix
	}
	units := [2]int{f.sc.CodeUnit(fix.Range[0]), f.sc.CodeUnit(fix.Range[1])}
	fix.units = &units
	return fix
}

// applyFixes mirrors SourceCodeFixer.applyFixes: sort the messages that have a
// fix by (start, end), then apply them left to right, keeping the first of any
// overlapping fixes and pushing the losers back into the remaining messages.
func applyFixes(sourceText string, messages []*Message, shouldFix func(*Message) bool) (bool, []*Message, string) {
	if shouldFix == nil {
		shouldFix = func(*Message) bool { return true }
	}
	const bom = "\uFEFF"
	hasBOM := len(sourceText) > 0 && sourceText[:len(bom)] == bom
	text := sourceText
	if hasBOM {
		text = sourceText[len(bom):]
	}

	remaining := []*Message{}
	fixes := []*Message{}
	for _, m := range messages {
		if m.Fix != nil {
			fixes = append(fixes, m)
		} else {
			remaining = append(remaining, m)
		}
	}

	if len(fixes) == 0 {
		return false, messages, sourceText
	}

	sort.SliceStable(fixes, func(i, j int) bool {
		if fixes[i].Fix.Range[0] != fixes[j].Fix.Range[0] {
			return fixes[i].Fix.Range[0] < fixes[j].Fix.Range[0]
		}
		return fixes[i].Fix.Range[1] < fixes[j].Fix.Range[1]
	})

	lastPos := -1 << 62
	output := ""
	if hasBOM {
		output = bom
	}
	fixesWereApplied := false

	attemptFix := func(m *Message) {
		start, end := m.Fix.Range[0], m.Fix.Range[1]
		if lastPos >= start || start > end {
			remaining = append(remaining, m)
			return
		}
		if (start < 0 && end >= 0) || (start == 0 && len(m.Fix.Text) > 0 && m.Fix.Text[:len(bom)] == bom) {
			output = ""
		}
		output += text[clampInt(lastPos, 0, len(text)):clampInt(start, 0, len(text))]
		output += m.Fix.Text
		lastPos = end
	}

	for _, m := range fixes {
		if shouldFix(m) {
			attemptFix(m)
			fixesWereApplied = true
		} else {
			remaining = append(remaining, m)
		}
	}
	if lastPos < 0 {
		lastPos = 0
	}
	output += text[clampInt(lastPos, 0, len(text)):]

	sort.SliceStable(remaining, func(i, j int) bool {
		if remaining[i].Line != remaining[j].Line {
			return remaining[i].Line < remaining[j].Line
		}
		return remaining[i].Column < remaining[j].Column
	})
	return fixesWereApplied, remaining, output
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
