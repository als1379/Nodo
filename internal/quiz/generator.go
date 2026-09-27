package quiz

import (
	"context"
	"encoding/json"
	"fmt"
)

type JSONCompleter interface {
	CompleteJSON(context.Context, string, string) ([]byte, error)
}

type LLMGenerator struct {
	client JSONCompleter
}

func NewLLMGenerator(client JSONCompleter) *LLMGenerator {
	return &LLMGenerator{client: client}
}

func (g *LLMGenerator) Generate(ctx context.Context) (generatedQuestion, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		content, err := g.client.CompleteJSON(ctx, wordQuestionSystemPrompt, wordQuestionUserPrompt)
		if err != nil {
			lastErr = err
			continue
		}
		var question generatedQuestion
		if err := json.Unmarshal(content, &question); err != nil {
			lastErr = fmt.Errorf("decode generated question: %w", err)
			continue
		}
		if question.Word == "" || len(question.Options) != 4 || question.CorrectOption < 0 || question.CorrectOption > 3 {
			lastErr = fmt.Errorf("LLM returned an invalid question")
			continue
		}
		return question, nil
	}
	return generatedQuestion{}, lastErr
}
