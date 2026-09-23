package eslint

// units.go — the byte/code-unit boundary.
//
// ESLint's positions (range, loc columns) are UTF-16 code-unit indices, because
// JS strings are UTF-16. This port works in BYTE offsets internally: acorn-go
// reports byte offsets, so AST ranges, token ranges and fix ranges are all
// byte-based and stay mutually consistent (and slicing a Go string by a byte
// range is what actually produces the fixed text).
//
// The two spaces agree for ASCII, which is why nothing notices on ASCII-only
// files. For a file containing any non-ASCII character (emoji, CJK, U+00A0, …)
// a byte offset is larger than the code-unit offset, so reported columns and
// fix ranges would differ from ESLint's.
//
// The conversion is therefore applied exactly where the boundary is: when a
// message (or a fix range inside it) is handed to the outside world. Rules keep
// working in bytes, the fixer keeps slicing bytes, and the emitted LintMessage
// is in ESLint's code-unit space.

// codeUnitTable maps byte offsets to UTF-16 code-unit offsets. It is only built
// for text that actually contains non-ASCII bytes, so the common case costs
// nothing.
type codeUnitTable struct {
	text    string
	perByte []int32 // perByte[i] = code units before byte offset i
	useful  bool
}

// newCodeUnitTable builds the mapping (or a pass-through for ASCII text).
func newCodeUnitTable(text string) *codeUnitTable {
	t := &codeUnitTable{text: text}
	nonASCII := false
	for i := 0; i < len(text); i++ {
		if text[i] >= 0x80 {
			nonASCII = true
			break
		}
	}
	if !nonASCII {
		return t
	}
	per := make([]int32, len(text)+1)
	units := int32(0)
	for i := 0; i < len(text); {
		per[i] = units
		b := text[i]
		var width int
		switch {
		case b < 0x80:
			width = 1
			units++
		case b < 0xE0:
			width = 2
			units++
		case b < 0xF0:
			width = 3
			units++
		case b < 0xF8:
			// Outside the BMP: two UTF-16 code units (surrogate pair).
			width = 4
			units += 2
		default:
			width = 1
			units++
		}
		for j := i + 1; j < i+width && j < len(text); j++ {
			per[j] = per[i]
		}
		i += width
	}
	per[len(text)] = units
	t.perByte = per
	t.useful = true
	return t
}

// CodeUnits converts a byte offset to a UTF-16 code-unit offset.
func (t *codeUnitTable) CodeUnits(byteOffset int) int {
	if t == nil || !t.useful {
		return byteOffset
	}
	if byteOffset < 0 {
		return 0
	}
	if byteOffset >= len(t.perByte) {
		return int(t.perByte[len(t.perByte)-1])
	}
	return int(t.perByte[byteOffset])
}

// ByteOffset converts a UTF-16 code-unit offset back to a byte offset (needed
// when a rule derives offsets from the reported columns). Offsets past the end
// clamp to the text length.
func (t *codeUnitTable) ByteOffset(codeUnits int) int {
	if t == nil || !t.useful {
		return codeUnits
	}
	for i := 0; i < len(t.perByte); i++ {
		if int(t.perByte[i]) >= codeUnits {
			return i
		}
	}
	return len(t.perByte) - 1
}

// CodeUnit converts a byte offset to a code-unit offset for this source.
func (s *SourceCode) CodeUnit(byteOffset int) int {
	if s == nil || s.units == nil {
		return byteOffset
	}
	return s.units.CodeUnits(byteOffset)
}

// NonASCII reports whether the source contains any non-ASCII byte, i.e. whether
// byte and code-unit offsets differ for it.
func (s *SourceCode) NonASCII() bool { return s.units.useful }

// locToCodeUnits rewrites a loc object's columns from byte offsets to UTF-16
// code units (the space ESLint reports in). Line numbers are unchanged.
func (s *SourceCode) locToCodeUnits(loc map[string]any) map[string]any {
	if loc == nil || !s.NonASCII() {
		return loc
	}
	out := make(map[string]any, len(loc))
	for k, v := range loc {
		out[k] = v
	}
	for _, which := range []string{"start", "end"} {
		pt, ok := loc[which].(map[string]any)
		if !ok || pt == nil {
			continue
		}
		line, col := numToInt(pt["line"]), numToInt(pt["column"])
		byteOffset := s.lineIndex.Index(line, col)
		lineStart := s.lineIndex.Index(line, 0)
		conv := s.CodeUnit(byteOffset) - s.CodeUnit(lineStart)
		out[which] = map[string]any{"line": float64(line), "column": float64(conv)}
	}
	return out
}
