package eslint

// position.go — line/column indexing, the Go counterpart of the
// lineStartIndices machinery in ESLint's SourceCode. Unlike espree's copy
// (which only has to number lines as acorn scans) this one is used at runtime
// by rules that convert between offsets and {line,column} (e.g. indent-style
// rules and the fixer), so it must agree with espree exactly.

// LineIndex maps offsets ↔ {line, column} for one source text. Lines are
// 1-based, columns 0-based (ESTree).
type LineIndex struct {
	text   string
	starts []int
}

// NewLineIndex builds the line table for text using the same line-break rules
// acorn applies: LF, CRLF, lone CR, U+2028 and U+2029.
func NewLineIndex(text string) *LineIndex {
	starts := []int{0}
	for i := 0; i < len(text); i++ {
		switch c := text[i]; {
		case c == '\n':
			starts = append(starts, i+1)
		case c == '\r':
			if i+1 < len(text) && text[i+1] == '\n' {
				i++
			}
			starts = append(starts, i+1)
		case c == 0xE2 && i+2 < len(text) && text[i+1] == 0x80 &&
			(text[i+2] == 0xA8 || text[i+2] == 0xA9):
			i += 2
			starts = append(starts, i+1)
		}
	}
	return &LineIndex{text: text, starts: starts}
}

// Text returns the indexed source text.
func (li *LineIndex) Text() string { return li.text }

// LineCount returns the number of lines.
func (li *LineIndex) LineCount() int { return len(li.starts) }

// Loc converts a byte offset to a 1-based line and 0-based column.
func (li *LineIndex) Loc(off int) (line, col int) {
	if off < 0 {
		off = 0
	}
	if off > len(li.text) {
		off = len(li.text)
	}
	lo, hi := 0, len(li.starts)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if li.starts[mid] <= off {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1, off - li.starts[lo]
}

// Index converts a 1-based line and 0-based column back to a byte offset,
// clamped into the text (ESLint's getIndexFromLoc, but tolerant of ranges
// ESLint would throw on — the CLI prefers a message over a crash).
func (li *LineIndex) Index(line, col int) int {
	if line < 1 {
		line = 1
	}
	if line > len(li.starts) {
		line = len(li.starts)
	}
	idx := li.starts[line-1] + col
	if idx < 0 {
		return 0
	}
	if idx > len(li.text) {
		return len(li.text)
	}
	return idx
}

// Lines returns the source split into lines, mirroring SourceCode.lines: the
// split is on \n (and \r\n) with a trailing newline NOT producing an extra
// empty element.
func (li *LineIndex) Lines() []string {
	out := make([]string, 0, len(li.starts))
	for i, start := range li.starts {
		if i == len(li.starts)-1 {
			out = append(out, li.text[start:])
		} else {
			end := li.starts[i+1] - 1
			if end > start && li.text[end-1] == '\r' {
				end--
			}
			out = append(out, li.text[start:end])
		}
	}
	return out
}
