package eslint

import (
	"regexp"
	"strings"
)

// Message is a single reported problem, matching ESLint's LintMessage shape.
type Message struct {
	RuleID    string `json:"ruleId"`
	Severity  int    `json:"severity"`
	Message   string `json:"message"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	NodeType  string `json:"nodeType"`
	MessageID string `json:"messageId,omitempty"`
	EndLine   int    `json:"endLine,omitempty"`
	EndColumn int    `json:"endColumn,omitempty"`
}

// ReportDescriptor mirrors the object passed to context.report().
type ReportDescriptor struct {
	Node      Node
	Loc       map[string]any // a {start,end} location object
	Message   string
	MessageID string
	Data      map[string]any
}

var interpolateRe = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

// interpolate replaces {{key}} placeholders with the matching data value,
// mirroring ESLint's interpolate.js for the placeholder shape we support.
func interpolate(template string, data map[string]any) string {
	if data == nil {
		return template
	}
	return interpolateRe.ReplaceAllStringFunc(template, func(m string) string {
		sub := interpolateRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		if v, ok := data[strings.TrimSpace(sub[1])]; ok {
			s, ok := v.(string)
			if ok {
				return s
			}
			return valueString(v)
		}
		return m
	})
}

func valueString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return itoa(t)
	case float64:
		return itoa(int(t))
	default:
		return ""
	}
}

// normalizeReportLoc replicates report-translator's normalizeReportLoc.
func normalizeReportLoc(d *ReportDescriptor) map[string]any {
	if d.Loc != nil {
		if _, ok := d.Loc["start"]; ok {
			return d.Loc
		}
		return map[string]any{"start": d.Loc, "end": nil}
	}
	return getLocObject(d.Node)
}

func getLocObject(n Node) map[string]any {
	if n == nil {
		return nil
	}
	if loc, ok := n["loc"].(map[string]any); ok {
		return loc
	}
	return nil
}

// createProblem replicates report-translator's createProblem for the
// no-fix subset (fix and suggestions are NotImplemented for the starter set).
func createProblem(ruleID string, severity int, node Node, message, messageID string, loc map[string]any) *Message {
	nt := ""
	if node != nil {
		nt, _ = node["type"].(string)
	}
	if nt == "" {
		nt = "null"
	}
	p := &Message{
		RuleID:    ruleID,
		Severity:  severity,
		Message:   message,
		Line:      locLine(loc, "start"),
		Column:    locColumn(loc, "start") + 1,
		NodeType:  nt,
		MessageID: messageID,
	}
	if loc != nil {
		if _, hasEnd := loc["end"]; hasEnd && loc["end"] != nil {
			p.EndLine = locLine(loc, "end")
			p.EndColumn = locColumn(loc, "end") + 1
		}
	}
	return p
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
