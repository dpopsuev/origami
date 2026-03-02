package cmd

import (
	framework "github.com/dpopsuev/origami"

	"github.com/dpopsuev/origami/marbles/rca/vocabulary"
)

var cmdVocab = vocabulary.New()

func vocabName(code string) string {
	return cmdVocab.Name(code)
}

func vocabNameWithCode(code string) string {
	return framework.NameWithCode(cmdVocab, code)
}
