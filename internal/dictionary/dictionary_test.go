package dictionary

import (
	"context"
	"errors"
	"testing"
)

type fakeDictionarySource struct{ body []byte }

func (f fakeDictionarySource) LookupItalian(context.Context, string) ([]byte, error) {
	return f.body, nil
}

type fakeLearningSource struct{ guide *LearningGuide }

func (f fakeLearningSource) Explain(context.Context, WordDetails) (*LearningGuide, error) {
	return f.guide, nil
}

type failingLearningSource struct{}

func (failingLearningSource) Explain(context.Context, WordDetails) (*LearningGuide, error) {
	return nil, errors.New("provider unavailable")
}

func TestDictionaryAddsAgentLearningGuide(t *testing.T) {
	source := fakeDictionarySource{body: []byte(`{"word":"parlare","entries":[{"pos":"verb","senses":[{"glosses":["to speak"]}]}]}`)}
	guide := &LearningGuide{Summary: "A regular verb", GrammarNote: "Uses avere", Examples: []LearningExample{{Italian: "Parlo.", English: "I speak."}}}
	details, err := NewDictionary(source, fakeLearningSource{guide: guide}).Lookup(context.Background(), "parlare")
	if err != nil {
		t.Fatal(err)
	}
	if details.Learning != guide {
		t.Fatalf("missing agent learning guide: %+v", details.Learning)
	}
}

func TestDictionaryMapsWiktAPIEntry(t *testing.T) {
	source := fakeDictionarySource{body: []byte(`{
		"word":"casa",
		"entries":[{
			"pos":"noun",
			"categories":["Italian nouns"],
			"synonyms":[{"word":"abitazione"}],
			"etymology_text":"From Latin casa.",
			"senses":[{"glosses":["house, home"],"raw_glosses":["(common) house, home"],"tags":["common","feminine"],"topics":["housing"],"examples":[{"text":"Torno a casa.","translation":"I am going home."}]}],
			"sounds":[{"ipa":"/ˈka.za/"}],
			"forms":[{"form":"case","tags":["plural"]}]
		}]
	}`)}
	details, err := NewDictionary(source).Lookup(context.Background(), "casa")
	if err != nil {
		t.Fatal(err)
	}
	entry := details.Entries[0]
	if entry.Grammar[0] != "feminine" || entry.Synonyms[0].Word != "abitazione" || entry.Definitions[0].Examples[0] != "Torno a casa. - I am going home." || entry.Definitions[0].Remarks[0] != "(common) house, home" {
		t.Fatalf("missing rich dictionary details: %+v", entry)
	}
	if details.Word != "casa" || entry.PartOfSpeech != "noun" || entry.Definitions[0].Meaning != "house, home" || entry.Pronunciation[0] != "/ˈka.za/" || entry.Forms[0].Form != "case" {
		t.Fatalf("unexpected details: %+v", details)
	}
}
