package main

import "testing"

func TestParseCurriculumPageKeepsOnlyNounsAndVerbs(t *testing.T) {
	body := []byte(`
		1. <a href="#">a</a> (prep.)<br>
		2. <a href="#">abitare</a> (v. int.)<br>
		3. <a href="#">aceto</a> (s.m.)<br>
		4. <a href="#">aereo(aeroplano)</a> (s.m.)<br>
		5. <a href="#">amico/a</a> (s.m./f.)<br>`)
	got := parseCurriculumPage(body, "A1")
	want := []curriculumWord{
		{Word: "abitare", Level: "A1", PartOfSpeech: "v", Rank: 2},
		{Word: "aceto", Level: "A1", PartOfSpeech: "n", Rank: 3},
		{Word: "aereo", Level: "A1", PartOfSpeech: "n", Rank: 4},
		{Word: "aeroplano", Level: "A1", PartOfSpeech: "n", Rank: 4},
		{Word: "amico", Level: "A1", PartOfSpeech: "n", Rank: 5},
	}
	if len(got) != len(want) {
		t.Fatalf("got %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestNormalizeLemmasRejectsPhrases(t *testing.T) {
	if got := normalizeLemmas("andare via"); len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}
