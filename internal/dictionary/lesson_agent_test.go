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

type failingCompleter struct {
	calls int
	err   error
}

func (f *failingCompleter) CompleteJSON(context.Context, string, string) ([]byte, error) {
	f.calls++
	return nil, f.err
}

type statusError int

func (e statusError) Error() string   { return "provider error" }
func (e statusError) StatusCode() int { return int(e) }

func (f *fakeCompleter) CompleteJSON(context.Context, string, string) ([]byte, error) {
	if f.calls >= len(f.responses) {
		return nil, errors.New("no response")
	}
	response := f.responses[f.calls]
	f.calls++
	return response, nil
}

func TestLearningAgentParsesStructuredLesson(t *testing.T) {
	client := &fakeCompleter{responses: [][]byte{[]byte(`{"learning":{"meaning":"to speak","summary":"A regular verb.","grammar_note":"Uses avere.","formation_rules":["Remove -are."],"tables":[{"title":"Present","forms":[{"label":"io","value":"parlo"}]}],"patterns":[],"examples":[{"italian":"Parlo.","english":"I speak."},{"italian":"Parli?","english":"Do you speak?"},{"italian":"Parliamo.","english":"We speak."},{"italian":"Parlano italiano.","english":"They speak Italian."}]}}`)}}
	guide, err := NewLearningAgent(client).Generate(context.Background(), WordDetails{Word: "parlare"})
	if err != nil {
		t.Fatal(err)
	}
	if guide.Tables[0].Forms[0].Value != "parlo" || len(guide.Examples) != 4 {
		t.Fatalf("unexpected guide: %+v", guide)
	}
}

func TestLearningAgentDoesNotRetryCreditLimit(t *testing.T) {
	client := &failingCompleter{err: statusError(402)}
	_, err := NewLearningAgent(client).Generate(context.Background(), WordDetails{Word: "lavoro"})
	if err == nil {
		t.Fatal("expected provider error")
	}
	if client.calls != 1 {
		t.Fatalf("credit-limit request was retried %d times", client.calls)
	}
}
