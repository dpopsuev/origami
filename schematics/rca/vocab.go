package rca

import (
	"fmt"
	"strings"

	framework "github.com/dpopsuev/origami"
	"gopkg.in/yaml.v3"
)

type vocabFile struct {
	DefectTypes map[string]framework.VocabEntry `yaml:"defect_types"`
	Stages      map[string]framework.VocabEntry `yaml:"stages"`
	Metrics     map[string]framework.VocabEntry `yaml:"metrics"`
	Heuristics  map[string]framework.VocabEntry `yaml:"heuristics"`
}

// NewVocabulary builds and returns a fully populated RichMapVocabulary
// containing domain codes: defect types, circuit stages, metrics, and
// heuristics. When data is nil an empty vocabulary is returned.
func NewVocabulary(data []byte) *framework.RichMapVocabulary {
	v := framework.NewRichMapVocabulary()
	if data == nil {
		return v
	}

	var f vocabFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		panic(fmt.Sprintf("vocabulary: parse YAML: %v", err))
	}

	v.RegisterEntries(f.DefectTypes)
	v.RegisterEntries(f.Stages)
	v.RegisterEntries(f.Metrics)
	v.RegisterEntries(f.Heuristics)
	return v
}

// SourceIssueTag formats a source-provided issue type with a trust indicator.
func SourceIssueTag(v framework.Vocabulary, issueType string, autoAnalyzed bool) string {
	if issueType == "" {
		return ""
	}
	tag := "[human]"
	if autoAnalyzed {
		tag = "[auto]"
	}
	return v.Name(issueType) + " " + tag
}

// StagePath converts a slice of stage codes to a human-readable path.
func StagePath(v framework.Vocabulary, codes []string) string {
	names := make([]string, len(codes))
	for i, c := range codes {
		names[i] = v.Name(c)
	}
	return strings.Join(names, " \u2192 ")
}

// ClusterKey humanizes a pipe-delimited cluster key.
func ClusterKey(v framework.Vocabulary, key string) string {
	parts := strings.Split(key, "|")
	for i, p := range parts {
		if name := v.Name(p); name != p {
			parts[i] = name
		}
	}
	return strings.Join(parts, " / ")
}

var defaultVocab = NewVocabulary(nil)

// InitVocab replaces the package-level vocabulary with one loaded from data.
func InitVocab(data []byte) {
	if data != nil {
		defaultVocab = NewVocabulary(data)
	}
}

func vocabName(code string) string {
	return defaultVocab.Name(code)
}

func vocabNameWithCode(code string) string {
	return framework.NameWithCode(defaultVocab, code)
}

func vocabStagePath(codes []string) string {
	return StagePath(defaultVocab, codes)
}

func vocabSourceIssueTag(issueType string, autoAnalyzed bool) string {
	return SourceIssueTag(defaultVocab, issueType, autoAnalyzed)
}
