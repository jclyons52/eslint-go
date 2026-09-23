package formatters

import (
	"regexp"
	"strings"
)

// text_table.go — a port of text-table@0.2.0 (the layout engine ESLint's
// stylish formatter uses). Rows are [][]string; options mirror the JS
// {hsep, align, stringLength}.
type tableOptions struct {
	// hsep is the column separator (default two spaces).
	hsep string
	// align is one of "", "l", "r", "c", "." per column.
	align []string
	// stringLength measures a cell's visible width.
	stringLength func(string) int
}

var dotRe = regexp.MustCompile(`\.[^.]*$`)

// dotIndex mirrors text-table's dotindex(): the index just after the last dot
// (or the string length when there is none).
func dotIndex(c string) int {
	m := dotRe.FindStringIndex(c)
	if m == nil {
		return len(c)
	}
	return m[0] + 1
}

func textTable(rows [][]string, opts tableOptions) string {
	if opts.hsep == "" {
		opts.hsep = "  "
	}
	if opts.stringLength == nil {
		opts.stringLength = func(s string) int { return utf16Len(s) }
	}

	// dotsizes: widest dot-prefix width per column.
	dotsizes := map[int]int{}
	for _, row := range rows {
		for ix, c := range row {
			n := dotIndex(c)
			if cur, ok := dotsizes[ix]; !ok || n > cur {
				dotsizes[ix] = n
			}
		}
	}

	// Pad dot-aligned cells.
	padded := make([][]string, len(rows))
	for ri, row := range rows {
		out := make([]string, len(row))
		for ix, c := range row {
			if alignAt(opts.align, ix) == "." {
				index := dotIndex(c)
				size := dotsizes[ix] + 1 // "/\\./.test(c) ? 1 : 2" – rows here always contain a dot when aligned
				if !strings.Contains(c, ".") {
					size = dotsizes[ix] + 2
				}
				size -= opts.stringLength(c) - index
				out[ix] = c + strings.Repeat(" ", maxInt(size-1, 0))
				continue
			}
			out[ix] = c
		}
		padded[ri] = out
	}

	// sizes: widest cell per column.
	sizes := map[int]int{}
	for _, row := range padded {
		for ix, c := range row {
			n := opts.stringLength(c)
			if cur, ok := sizes[ix]; !ok || n > cur {
				sizes[ix] = n
			}
		}
	}

	lines := make([]string, 0, len(padded))
	for _, row := range padded {
		cells := make([]string, 0, len(row))
		for ix, c := range row {
			n := sizes[ix] - opts.stringLength(c)
			if n < 0 {
				n = 0
			}
			spaces := strings.Repeat(" ", n)
			switch alignAt(opts.align, ix) {
			case "r", ".":
				cells = append(cells, spaces+c)
			case "c":
				left := strings.Repeat(" ", ceilDiv(n, 2))
				right := strings.Repeat(" ", floorDiv(n, 2))
				cells = append(cells, left+c+right)
			default:
				cells = append(cells, c+spaces)
			}
		}
		line := strings.Join(cells, opts.hsep)
		lines = append(lines, strings.TrimRight(line, " \t\n\r"))
	}
	return strings.Join(lines, "\n")
}

func alignAt(align []string, ix int) string {
	if ix < len(align) {
		return align[ix]
	}
	return ""
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func floorDiv(a, b int) int { return a / b }

func ceilDiv(a, b int) int {
	if a%b == 0 {
		return a / b
	}
	return a/b + 1
}
