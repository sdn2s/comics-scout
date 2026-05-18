package words

import (
	"slices"
	"testing"
)

func TestNorm_RemovesStopWordsAndDuplicates(t *testing.T) {
	phrase := "The quick brown fox jumps over the lazy dog! QUICK foxes, 123fox."

	got := Norm(phrase)
	slices.Sort(got)

	expected := []string{"123fox", "brown", "dog", "fox", "jump", "lazi", "quick"}
	if !slices.Equal(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}
