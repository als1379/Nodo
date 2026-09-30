package dictionary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const lessonGenerationAttempts = 2

type StructuredCompletionClient interface {
	CompleteJSON(context.Context, string, string) ([]byte, error)
}

type schemaCompletionClient interface {
	CompleteJSONSchema(context.Context, string, string, string, map[string]any) ([]byte, error)
}

type LearningAgent struct{ client StructuredCompletionClient }

func NewLearningAgent(client StructuredCompletionClient) *LearningAgent {
	return &LearningAgent{client: client}
}

func (a *LearningAgent) Generate(ctx context.Context, details WordDetails) (*LearningGuide, error) {
	evidence, err := json.Marshal(struct {
		Word    string            `json:"word"`
		Entries []DictionaryEntry `json:"entries"`
	}{details.Word, details.Entries})
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt < lessonGenerationAttempts; attempt++ {
		body, err := a.complete(ctx, fmt.Sprintf(learningUserPrompt, evidence))
		if err != nil {
			var provider statusCoder
			if errors.As(err, &provider) && (provider.StatusCode() == 402 || provider.StatusCode() == 429) {
				return nil, err
			}
			lastErr = err
			continue
		}
		lesson, err := decodeLesson(body)
		if err == nil {
			return lesson, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (a *LearningAgent) complete(ctx context.Context, userPrompt string) ([]byte, error) {
	if client, ok := a.client.(schemaCompletionClient); ok {
		return client.CompleteJSONSchema(ctx, learningSystemPrompt, userPrompt, "italian_learning_guide", learningGuideSchema)
	}
	return a.client.CompleteJSON(ctx, learningSystemPrompt, userPrompt)
}
