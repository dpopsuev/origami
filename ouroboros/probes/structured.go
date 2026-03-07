package probes

import (
	"strings"
)

// ParsedResponse holds labeled fields extracted from a structured response.
// Fields are indexed by their uppercase label (e.g., "ROOT_CAUSE", "VERDICT").
type ParsedResponse struct {
	Fields map[string]string
	Lists  map[string][]string
}

// ParseStructured extracts labeled fields and lists from a response.
// Recognized formats:
//
//	LABEL: single-line value
//	LABEL:
//	- list item 1
//	- list item 2
//
// Multi-line values continue until the next label or end of input.
func ParseStructured(raw string) ParsedResponse {
	pr := ParsedResponse{
		Fields: make(map[string]string),
		Lists:  make(map[string][]string),
	}

	lines := strings.Split(raw, "\n")
	var currentLabel string
	var currentItems []string
	var multilineValue strings.Builder

	flushCurrent := func() {
		if currentLabel == "" {
			return
		}
		if len(currentItems) > 0 {
			pr.Lists[currentLabel] = currentItems
			pr.Fields[currentLabel] = strings.Join(currentItems, "; ")
		} else if multilineValue.Len() > 0 {
			pr.Fields[currentLabel] = strings.TrimSpace(multilineValue.String())
		}
		currentLabel = ""
		currentItems = nil
		multilineValue.Reset()
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "- ") && currentLabel != "" {
			currentItems = append(currentItems, strings.TrimSpace(trimmed[2:]))
			continue
		}

		if label, value, ok := parseLabel(trimmed); ok {
			flushCurrent()
			currentLabel = label
			if value != "" {
				multilineValue.WriteString(value)
			}
			continue
		}

		if currentLabel != "" && trimmed != "" {
			if multilineValue.Len() > 0 {
				multilineValue.WriteString(" ")
			}
			multilineValue.WriteString(trimmed)
		}
	}
	flushCurrent()

	return pr
}

// parseLabel extracts a label and optional inline value from a line like "LABEL: value".
// Labels must be uppercase letters, digits, and underscores.
func parseLabel(line string) (label, value string, ok bool) {
	colonIdx := strings.Index(line, ":")
	if colonIdx < 1 {
		return "", "", false
	}

	candidate := line[:colonIdx]
	for _, ch := range candidate {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return "", "", false
		}
	}

	return candidate, strings.TrimSpace(line[colonIdx+1:]), true
}

// HasField returns true if the parsed response contains the given label.
func (pr ParsedResponse) HasField(label string) bool {
	_, ok := pr.Fields[label]
	return ok
}

// FieldContains returns true if the named field contains the substring (case-insensitive).
func (pr ParsedResponse) FieldContains(label, substring string) bool {
	v, ok := pr.Fields[label]
	if !ok {
		return false
	}
	return strings.Contains(strings.ToLower(v), strings.ToLower(substring))
}

// ListLen returns the number of items in the named list, or 0 if not present.
func (pr ParsedResponse) ListLen(label string) int {
	return len(pr.Lists[label])
}
