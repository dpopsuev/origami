package rca

import (
	framework "github.com/dpopsuev/origami"

	"github.com/dpopsuev/origami/schematics/rca/vocabulary"
)

var defaultVocab = vocabulary.New()

func vocabName(code string) string {
	return defaultVocab.Name(code)
}

func vocabNameWithCode(code string) string {
	return framework.NameWithCode(defaultVocab, code)
}

func vocabStagePath(codes []string) string {
	return vocabulary.StagePath(defaultVocab, codes)
}

func vocabSourceIssueTag(issueType string, autoAnalyzed bool) string {
	return vocabulary.SourceIssueTag(defaultVocab, issueType, autoAnalyzed)
}

