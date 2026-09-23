// Package formatters implements ESLint's output formatters. They are ports of
// the formatter modules shipped with eslint 8.57 (lib/cli-engine/formatters/),
// including the chalk (ANSI) behaviour and the text-table layout, because the
// CLI's output has to be byte-identical to real ESLint's.
package formatters

import "strings"

// Chalk is a faithful subset of the npm chalk v4 API used by ESLint's
// formatters. Chalk's only subtlety is nesting: applying a style to a string
// that already contains escape sequences re-opens the outer style after every
// inner close, so `red("a" + yellow("b") + "c")` yields
// `ESC[31m a ESC[33m b ESC[39m ESC[31m c ESC[39m`.
type Chalk struct {
	// Level is chalk's color level: 0 disables all styling, 1 is basic
	// 16-colour output (what FORCE_COLOR=1 / a TTY gives ESLint).
	Level int
}

// style is one ANSI style: the open and close sequences.
type style struct {
	open  string
	close string
}

var (
	sRed       = style{"\x1b[31m", "\x1b[39m"}
	sYellow    = style{"\x1b[33m", "\x1b[39m"}
	sGreen     = style{"\x1b[32m", "\x1b[39m"}
	sBlue      = style{"\x1b[34m", "\x1b[39m"}
	sMagenta   = style{"\x1b[35m", "\x1b[39m"}
	sCyan      = style{"\x1b[36m", "\x1b[39m"}
	sGray      = style{"\x1b[90m", "\x1b[39m"}
	sDim       = style{"\x1b[2m", "\x1b[22m"}
	sBold      = style{"\x1b[1m", "\x1b[22m"}
	sUnderline = style{"\x1b[4m", "\x1b[24m"}
	sReset     = style{"\x1b[0m", "\x1b[0m"}
)

// NewChalk returns a chalk emulation at the given level.
func NewChalk(level int) *Chalk { return &Chalk{Level: level} }

// apply composes styles exactly as chalk does: each style in the chain
// re-opens after its own close sequence, and — for a string containing
// newlines — the style is closed at the end of every line and reopened on the
// next (chalk's stringEncaseCRLFWithFirstIndex). That is why ESLint's colour
// output has a reset at the start and end of each line.
func (c *Chalk) apply(text string, chain ...style) string {
	if c.Level <= 0 || text == "" {
		return text
	}
	for _, st := range chain {
		if strings.Contains(text, st.close) {
			text = strings.ReplaceAll(text, st.close, st.close+st.open)
		}
	}
	var opens, closes strings.Builder
	for i := len(chain) - 1; i >= 0; i-- {
		opens.WriteString(chain[i].open)
	}
	for _, st := range chain {
		closes.WriteString(st.close)
	}
	openAll, closeAll := opens.String(), closes.String()
	if strings.Contains(text, "\n") {
		text = strings.ReplaceAll(text, "\r\n", closeAll+"\r\n"+openAll)
		text = strings.ReplaceAll(text, "\n", closeAll+"\n"+openAll)
	}
	return openAll + text + closeAll
}

// Red styles text red.
func (c *Chalk) Red(text string) string { return c.apply(text, sRed) }

// Yellow styles text yellow.
func (c *Chalk) Yellow(text string) string { return c.apply(text, sYellow) }

// Green styles text green.
func (c *Chalk) Green(text string) string { return c.apply(text, sGreen) }

// Blue styles text blue.
func (c *Chalk) Blue(text string) string { return c.apply(text, sBlue) }

// Magenta styles text magenta.
func (c *Chalk) Magenta(text string) string { return c.apply(text, sMagenta) }

// Cyan styles text cyan.
func (c *Chalk) Cyan(text string) string { return c.apply(text, sCyan) }

// Gray styles text gray (bright black).
func (c *Chalk) Gray(text string) string { return c.apply(text, sGray) }

// Dim styles text dim.
func (c *Chalk) Dim(text string) string { return c.apply(text, sDim) }

// Bold styles text bold.
func (c *Chalk) Bold(text string) string { return c.apply(text, sBold) }

// Underline styles text underlined.
func (c *Chalk) Underline(text string) string { return c.apply(text, sUnderline) }

// RedBold is chalk.red.bold.
func (c *Chalk) RedBold(text string) string { return c.apply(text, sBold, sRed) }

// YellowBold is chalk.yellow.bold.
func (c *Chalk) YellowBold(text string) string { return c.apply(text, sBold, sYellow) }

// YellowUnderline is chalk.yellow.underline.
func (c *Chalk) YellowUnderline(text string) string { return c.apply(text, sUnderline, sYellow) }

// RedUnderline is chalk.red.underline.
func (c *Chalk) RedUnderline(text string) string { return c.apply(text, sUnderline, sRed) }

// GrayUnderline is chalk.gray.underline.
func (c *Chalk) GrayUnderline(text string) string { return c.apply(text, sUnderline, sGray) }

// Reset wraps text with chalk.reset (an opening and a closing reset).
func (c *Chalk) Reset(text string) string { return c.apply(text, sReset) }

// stripAnsi removes ANSI escape sequences (strip-ansi, as text-table's
// stringLength option uses it to measure visible width).
func stripAnsi(s string) string {
	if !strings.Contains(s, "\x1b") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			continue
		}
		// Skip ESC [ ... <final byte in @-~>
		j := i + 1
		if j < len(s) && (s[j] == '[' || s[j] == ']') {
			j++
			for j < len(s) && !(s[j] >= 0x40 && s[j] <= 0x7e) {
				j++
			}
		}
		i = j
	}
	return b.String()
}

// visibleLen is the string length after stripping ANSI escapes (JS
// String.length counts UTF-16 code units; for the ASCII-ish formatter output
// this matches, and non-ASCII paths are measured the same way ESLint measures
// them).
func visibleLen(s string) int { return utf16Len(stripAnsi(s)) }

// utf16Len counts UTF-16 code units, matching JS string length.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
			continue
		}
		n++
	}
	return n
}
