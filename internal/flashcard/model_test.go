package flashcard

import "testing"

func TestParseSessionMode(t *testing.T) {
	if mode, err := ParseSessionMode(""); err != nil || mode != ModeLearn {
		t.Fatalf("default mode=%q err=%v", mode, err)
	}
	if mode, err := ParseSessionMode("REVIEW"); err != nil || mode != ModeReview {
		t.Fatalf("review mode=%q err=%v", mode, err)
	}
	if _, err := ParseSessionMode("mixed"); err == nil {
		t.Fatal("expected unsupported mode error")
	}
}

func TestInterleaveCards(t *testing.T) {
	reviews := []Card{{Word: "r1"}, {Word: "r2"}}
	newCards := []Card{{Word: "n1"}, {Word: "n2"}, {Word: "n3"}}
	result := interleaveCards(reviews, newCards)
	want := []string{"r1", "n1", "r2", "n2", "n3"}
	for i, word := range want {
		if result[i].Word != word {
			t.Fatalf("position %d = %q, want %q", i, result[i].Word, word)
		}
	}
}
