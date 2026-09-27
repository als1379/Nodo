package quiz

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

func TestLLMGeneratorRetriesMalformedJSON(t *testing.T) {
	client := &fakeCompleter{responses: [][]byte{
		[]byte(`{"word":`),
		[]byte(`{"word":"brief","options":["short","loud","old","bright"],"correct_option":0}`),
	}}
	question, err := NewLLMGenerator(client).Generate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if client.calls != 2 || question.Word != "brief" {
		t.Fatalf("calls=%d word=%q", client.calls, question.Word)
	}
}
