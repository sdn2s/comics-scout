package words

import (
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/kljensen/snowball/english"
)

func Norm(phrase string) []string {
	words := make(map[string]bool)
	splitted := strings.FieldsFunc(phrase, func(r rune) bool {
		return !unicode.IsDigit(r) && !unicode.IsLetter(r)
	})
	for _, w := range splitted {
		w := strings.ToLower(w)
		if english.IsStopWord(w) {
			continue
		}
		words[english.Stem(w, false)] = true
	}
	return slices.Collect(maps.Keys(words))
}
