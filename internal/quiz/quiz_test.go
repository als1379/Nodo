package quiz

import (
	"context"
	"errors"
	"testing"
)

type fakeGenerator struct {
	questions []generatedQuestion
	index     int
}

func (f *fakeGenerator) Generate(context.Context) (generatedQuestion, error) {
	q := f.questions[f.index]
	f.index++
	return q, nil
}

func TestAnswerReturnsResult(t *testing.T) {
	generator := &fakeGenerator{questions: []generatedQuestion{
		{Word: "brief", Options: []string{"short", "loud", "old", "bright"}, CorrectOption: 0},
		{Word: "vivid", Options: []string{"dull", "clear", "slow", "quiet"}, CorrectOption: 1},
	}}
	service := NewService(generator)
	first, err := service.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	correct, answer, err := service.Answer(first.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !correct || answer != "short" {
		t.Fatalf("unexpected result: correct=%v answer=%q", correct, answer)
	}
	if _, _, err := service.Answer(first.ID, 0); !errors.Is(err, ErrQuestionNotFound) {
		t.Fatalf("expected ErrQuestionNotFound, got %v", err)
	}
}
