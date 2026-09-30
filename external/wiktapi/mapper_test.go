package wiktapi

import (
	"encoding/json"
	"testing"
)

func TestMapResponseBuildsDictionaryDetails(t *testing.T) {
	data := []byte(`{
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
	}`)
	var response response
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	details, err := mapResponse("casa", response)
	if err != nil {
		t.Fatal(err)
	}
	entry := details.Entries[0]
	if entry.Grammar[0] != "feminine" || entry.Synonyms[0].Word != "abitazione" || entry.Definitions[0].Examples[0] != "Torno a casa. - I am going home." {
		t.Fatalf("missing mapped details: %+v", entry)
	}
	if details.Word != "casa" || entry.Definitions[0].Meaning != "house, home" || entry.Forms[0].Form != "case" {
		t.Fatalf("unexpected details: %+v", details)
	}
}
