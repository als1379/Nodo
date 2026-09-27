package quiz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

var ErrQuestionNotFound = errors.New("question not found or already answered")

type Question struct {
	ID      string   `json:"id"`
	Word    string   `json:"word"`
	Options []string `json:"options"`
}

type generatedQuestion struct {
	Word          string   `json:"word"`
	Options       []string `json:"options"`
	CorrectOption int      `json:"correct_option"`
}

type Generator interface {
	Generate(context.Context) (generatedQuestion, error)
}

type Service struct {
	generator Generator
	mu        sync.Mutex
	answers   map[string]generatedQuestion
}

func NewService(generator Generator) *Service {
	return &Service{generator: generator, answers: make(map[string]generatedQuestion)}
}

func (s *Service) Next(ctx context.Context) (Question, error) {
	generated, err := s.generator.Generate(ctx)
	if err != nil {
		return Question{}, err
	}
	if generated.Word == "" || len(generated.Options) != 4 || generated.CorrectOption < 0 || generated.CorrectOption > 3 {
		return Question{}, errors.New("LLM returned an invalid question")
	}
	id, err := randomID()
	if err != nil {
		return Question{}, err
	}
	s.mu.Lock()
	s.answers[id] = generated
	s.mu.Unlock()
	return Question{ID: id, Word: generated.Word, Options: generated.Options}, nil
}

func (s *Service) Answer(id string, option int) (bool, string, error) {
	s.mu.Lock()
	current, ok := s.answers[id]
	s.mu.Unlock()
	if !ok {
		return false, "", ErrQuestionNotFound
	}
	s.mu.Lock()
	delete(s.answers, id)
	s.mu.Unlock()
	return option == current.CorrectOption, current.Options[current.CorrectOption], nil
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
