package eslint

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// report.go — the report-translator. A rule's context.report() descriptor is
// turned into ESLint's LintMessage object: messageId lookup + {{data}}
// interpolation, loc normalisation, and the exact set/order of properties
// ESLint emits (which is what makes message-level parity testable).

// Message is one reported problem — ESLint's LintMessage.
type Message struct {
	RuleID    string
	Severity  int
	Message   string
	Line      int
	Column    int
	NodeType  string
	MessageID string
	EndLine   int
	EndColumn int
	Fatal     bool
	Fix       *Fix
	// Suggestions is the reported rule's suggest list, already filtered the
	// way ESLint filters it (entries without a fix are dropped). Present only
	// in the JSON formatter's output — --fix never applies suggestions.
	Suggestions []SuggestionResult
}

// Suggestion is one entry of a report descriptor's `suggest` array.
//
// ESLint emits the entry's own keys in the order the rule wrote them, then
// desc last; Go has no field order, so SuggestionResult marshals the shape
// every rule in this port uses ({messageId, data?, fix?, desc}).
type Suggestion struct {
	// MessageID names a message in the rule's meta.messages (used for desc).
	MessageID string
	// Data supplies {{placeholders}} for the desc template.
	Data map[string]any
	// Desc overrides the message template (ESLint's `suggest[i].desc`).
	Desc string
	// Fix produces the suggested edit.
	Fix func(*Fixer) *Fix
}

// SuggestionResult is a suggestion as emitted in a LintMessage.
type SuggestionResult struct {
	MessageID string
	Data      map[string]any
	Fix       *Fix
	Desc      string
}

// Fix is a rule fix: replace [Range[0], Range[1]) with Text. Range is in byte
// offsets internally; units carries the UTF-16 code-unit range ESLint reports
// (set only for sources containing non-ASCII, see units.go).
type Fix struct {
	Range [2]int
	Text  string
	units *[2]int
}

// FixMessage is one message's fix, used by the fixer.
type FixMessage struct {
	Message *Message
	Fix     Fix
}

// MarshalJSON emits the message in ESLint's property order, with the same
// conditional properties (messageId / endLine+endColumn / fix / fatal) and
// ruleId:null and nodeType:null for fatal parse errors.
func (m Message) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	if m.RuleID == "" {
		b.WriteString(`"ruleId":null`)
	} else {
		b.WriteString(`"ruleId":` + jsonString(m.RuleID))
	}
	if m.Fatal {
		b.WriteString(`,"fatal":true`)
	}
	b.WriteString(`,"severity":` + itoa(m.Severity))
	b.WriteString(`,"message":` + jsonString(m.Message))
	b.WriteString(`,"line":` + itoa(m.Line))
	b.WriteString(`,"column":` + itoa(m.Column))
	if m.NodeType == "" {
		b.WriteString(`,"nodeType":null`)
	} else {
		b.WriteString(`,"nodeType":` + jsonString(m.NodeType))
	}
	if m.MessageID != "" {
		b.WriteString(`,"messageId":` + jsonString(m.MessageID))
	}
	if m.EndLine != 0 || m.EndColumn != 0 {
		b.WriteString(`,"endLine":` + itoa(m.EndLine))
		b.WriteString(`,"endColumn":` + itoa(m.EndColumn))
	}
	if m.Fix != nil {
		r := m.Fix.Range
		if m.Fix.units != nil {
			r = *m.Fix.units
		}
		b.WriteString(`,"fix":{"range":[` + itoa(r[0]) + `,` + itoa(r[1]) +
			`],"text":` + jsonString(m.Fix.Text) + `}`)
	}
	if len(m.Suggestions) > 0 {
		b.WriteString(`,"suggestions":[`)
		for i, s := range m.Suggestions {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('{')
			b.WriteString(`"messageId":` + jsonString(s.MessageID))
			if s.Data != nil {
				b.WriteString(`,"data":` + jsonValue(s.Data))
			}
			if s.Fix != nil {
				r := s.Fix.Range
				if s.Fix.units != nil {
					r = *s.Fix.units
				}
				b.WriteString(`,"fix":{"range":[` + itoa(r[0]) + `,` + itoa(r[1]) +
					`],"text":` + jsonString(s.Fix.Text) + `}`)
			}
			b.WriteString(`,"desc":` + jsonString(s.Desc))
			b.WriteByte('}')
		}
		b.WriteByte(']')
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// jsonValue encodes arbitrary report data. ESLint preserves the rule's key
// order; Go maps do not, so keys are sorted (the port's suggestion data is
// single-key in every rule that carries it).
func jsonValue(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return `null`
	}
	return strings.TrimRight(buf.String(), "\n")
}

// jsonString encodes a Go string as JSON with HTML escaping disabled, so output
// matches JSON.stringify byte for byte (Go's default would escape <, > and &).
func jsonString(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return `""`
	}
	return strings.TrimRight(buf.String(), "\n")
}

// Report is the object passed to context.report().
type Report struct {
	// Node is the reported node (may be nil for a loc-only report).
	Node Node
	// Loc overrides the location ({start,end}); a {line,column} point is also
	// accepted and yields a message with no end position.
	Loc map[string]any
	// Message is a literal message (mutually exclusive with MessageID).
	Message string
	// MessageID names a message in the rule's meta.messages.
	MessageID string
	// Data supplies {{placeholders}} for the message template.
	Data map[string]any
	// Fix produces a fix for the problem.
	Fix func(*Fixer) *Fix
	// Suggest lists editable suggestions for the problem (ESLint's
	// descriptor.suggest). Entries are emitted only in the JSON formatter and
	// never applied by --fix.
	Suggest []Suggestion
}

// mapSuggestions replicates report-translator's mapSuggestions: each entry's
// desc is interpolated from its messageId (or explicit desc) and its fix is
// materialised — entries that produce no fix are dropped.
func mapSuggestions(list []Suggestion, messages map[string]string, sc *SourceCode) []SuggestionResult {
	if len(list) == 0 {
		return nil
	}
	out := make([]SuggestionResult, 0, len(list))
	for _, s := range list {
		desc := s.Desc
		if desc == "" {
			desc = messages[s.MessageID]
		}
		desc = interpolate(desc, s.Data)
		var fix *Fix
		if s.Fix != nil {
			fix = s.Fix(&Fixer{sc: sc})
		}
		if fix == nil {
			continue
		}
		fix = (&Fixer{sc: sc}).withUnits(fix)
		out = append(out, SuggestionResult{MessageID: s.MessageID, Data: s.Data, Fix: fix, Desc: desc})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

var interpolateRe = regexp.MustCompile(`\{\{\s*([^{}\s]+)\s*\}\}`)

// interpolate replaces {{key}} placeholders, as ESLint's interpolate.js does
// (an unknown placeholder is left untouched so the gap is visible rather than
// silently dropped — also what ESLint does).
func interpolate(template string, data map[string]any) string {
	if data == nil {
		return template
	}
	return interpolateRe.ReplaceAllStringFunc(template, func(m string) string {
		sub := interpolateRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		v, ok := data[sub[1]]
		if !ok {
			return m
		}
		return dataToString(v)
	})
}

func dataToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return itoa(t)
	case float64:
		return jsNumberString(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	case []any:
		// JS String(array) joins with commas.
		parts := make([]string, 0, len(t))
		for _, el := range t {
			parts = append(parts, dataToString(el))
		}
		return strings.Join(parts, ",")
	case Node:
		// JS String(object) — rules that pass a whole node as data see this.
		return "[object Object]"
	}
	return ""
}

// jsNumberString formats a number the way JS String(number) does for the
// integers and simple decimals rules interpolate.
func jsNumberString(f float64) string {
	if f == float64(int64(f)) {
		return itoa(int(f))
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// normalizeReportLoc replicates report-translator's normalizeReportLoc: an
// explicit loc wins; a bare {line,column} becomes {start, end:null} (which
// suppresses endLine/endColumn); otherwise the node's loc is used.
func normalizeReportLoc(d Report) map[string]any {
	if d.Loc != nil {
		if _, ok := d.Loc["start"]; ok {
			return d.Loc
		}
		return map[string]any{"start": d.Loc, "end": nil}
	}
	return Loc(d.Node)
}

// createProblem replicates report-translator's createProblem. The loc columns
// are converted to ESLint's UTF-16 code-unit space here, the last point at
// which the source text is still available.
func createProblem(ruleID string, severity int, node Node, message, messageID string, loc map[string]any, fix *Fix, sc *SourceCode) *Message {
	nt := ""
	if node != nil {
		nt = NodeType(node)
	}
	if sc != nil {
		loc = sc.locToCodeUnits(loc)
	}
	startLine, startCol := locPoint(loc, "start")
	p := &Message{
		RuleID:    ruleID,
		Severity:  severity,
		Message:   message,
		Line:      startLine,
		Column:    startCol + 1,
		NodeType:  nt,
		MessageID: messageID,
		Fix:       fix,
	}
	if loc != nil {
		if end, ok := loc["end"]; ok && end != nil {
			endLine, endCol := locPoint(loc, "end")
			p.EndLine = endLine
			p.EndColumn = endCol + 1
		}
	}
	return p
}

// fatalMessage builds the message ESLint emits when parsing fails.
func fatalMessage(err error, normalized string, line, col int) *Message {
	return &Message{
		RuleID:   "",
		Fatal:    true,
		Severity: 2,
		Message:  normalized,
		Line:     line,
		Column:   col,
	}
}

// ParseErrorColumn converts a parser error's byte-based 1-based column into the
// UTF-16 code-unit column ESLint reports (see units.go). It is a no-op for
// ASCII-only text.
func ParseErrorColumn(text string, line, col int) int {
	if col <= 1 || !hasNonASCII(text) {
		return col
	}
	li := NewLineIndex(text)
	units := newCodeUnitTable(text)
	byteOffset := li.Index(line, col-1)
	lineStart := li.Index(line, 0)
	return units.CodeUnits(byteOffset) - units.CodeUnits(lineStart) + 1
}

func hasNonASCII(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] >= 0x80 {
			return true
		}
	}
	return false
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}
