package dictionary

import (
	"context"
	"errors"
	"testing"
)

type fakeCompleter struct {
	responses [][]byte
	calls     int
}

func (f *fakeCompleter) CompleteJSON(context.Context, string, string) ([]byte, error) {
	if f.calls >= len(f.responses) {
		return nil, errors.New("no response")
	}
	response := f.responses[f.calls]
	f.calls++
	return response, nil
}

func TestLearningAgentParsesStructuredLesson(t *testing.T) {
	client := &fakeCompleter{responses: [][]byte{[]byte(`{"learning":{"summary":"A regular verb.","grammar_note":"Uses avere.","formation_rules":["Remove -are."],"tables":[{"title":"Present","forms":[{"label":"io","value":"parlo"}]}],"patterns":[],"examples":[{"italian":"Parlo.","english":"I speak."},{"italian":"Parli?","english":"Do you speak?"},{"italian":"Parliamo.","english":"We speak."},{"italian":"Parlano italiano.","english":"They speak Italian."}]}}`)}}
	guide, err := NewLearningAgent(client).Explain(context.Background(), WordDetails{Word: "parlare"})
	if err != nil {
		t.Fatal(err)
	}
	if guide.Tables[0].Forms[0].Value != "parlo" || len(guide.Examples) != 4 {
		t.Fatalf("unexpected guide: %+v", guide)
	}
}
