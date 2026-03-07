package framework
// Category: Processing & Support

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// JSONExtractor parses JSON bytes into a typed Go struct.
// Low floor: works with any json-deserializable type via generics.
func NewJSONExtractor[T any](name string) Extractor {
	return &jsonExtractor[T]{name: name}
}

type jsonExtractor[T any] struct {
	name string
}

func (e *jsonExtractor[T]) Name() string { return e.name }

func (e *jsonExtractor[T]) Extract(_ context.Context, input any) (any, error) {
	data, ok := input.([]byte)
	if !ok {
		if s, ok2 := input.(string); ok2 {
			data = []byte(s)
		} else {
			return nil, fmt.Errorf("JSONExtractor %q: expected []byte or string, got %T", e.name, input)
		}
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("JSONExtractor %q: %w", e.name, err)
	}
	return &result, nil
}

// RegexExtractor extracts named capture groups from text.
// The pattern is compiled at construction time (not per-call) to avoid ReDoS
// on repeated invocations.
func NewRegexExtractor(name string, pattern string) (Extractor, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("RegexExtractor %q: compile pattern: %w", name, err)
	}
	return &regexExtractor{name: name, re: re}, nil
}

// MustRegexExtractor is like NewRegexExtractor but panics on invalid pattern.
func MustRegexExtractor(name string, pattern string) Extractor {
	ext, err := NewRegexExtractor(name, pattern)
	if err != nil {
		panic(err)
	}
	return ext
}

type regexExtractor struct {
	name string
	re   *regexp.Regexp
}

func (e *regexExtractor) Name() string { return e.name }

func (e *regexExtractor) Extract(_ context.Context, input any) (any, error) {
	text, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("RegexExtractor %q: expected string, got %T", e.name, input)
	}
	match := e.re.FindStringSubmatch(text)
	if match == nil {
		return nil, fmt.Errorf("RegexExtractor %q: no match in input (len=%d)", e.name, len(text))
	}
	result := make(map[string]string)
	for i, name := range e.re.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		result[name] = match[i]
	}
	return result, nil
}

// CodeBlockExtractor extracts the content of the first fenced code block
// (``` or ```lang) from text. Falls back to content after "func " if no
// fenced block is found.
func NewCodeBlockExtractor(name string) Extractor {
	return &codeBlockExtractor{name: name}
}

var codeBlockRe = regexp.MustCompile("(?s)```(?:\\w+)?\\s*\\n(.*?)\\n```")

type codeBlockExtractor struct {
	name string
}

func (e *codeBlockExtractor) Name() string { return e.name }

func (e *codeBlockExtractor) Extract(_ context.Context, input any) (any, error) {
	text, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("CodeBlockExtractor %q: expected string, got %T", e.name, input)
	}
	match := codeBlockRe.FindStringSubmatch(text)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1]), nil
	}
	return nil, fmt.Errorf("CodeBlockExtractor %q: no fenced code block found (len=%d)", e.name, len(text))
}

// LineSplitExtractor splits text on newlines and removes blank lines.
func NewLineSplitExtractor(name string) Extractor {
	return &lineSplitExtractor{name: name}
}

type lineSplitExtractor struct {
	name string
}

func (e *lineSplitExtractor) Name() string { return e.name }

func (e *lineSplitExtractor) Extract(_ context.Context, input any) (any, error) {
	text, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("LineSplitExtractor %q: expected string, got %T", e.name, input)
	}
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, l := range raw {
		if trimmed := strings.TrimSpace(l); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines, nil
}
