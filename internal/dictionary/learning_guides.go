package dictionary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type JSONCompleter interface {
	CompleteJSON(context.Context, string, string) ([]byte, error)
}

type LearningGuide struct {
	Summary        string            `json:"summary"`
	GrammarNote    string            `json:"grammar_note"`
	FormationRules []string          `json:"formation_rules,omitempty"`
	Tables         []LearningTable   `json:"tables,omitempty"`
	Patterns       []string          `json:"patterns,omitempty"`
	Examples       []LearningExample `json:"examples"`
}

type LearningTable struct {
	Title string         `json:"title"`
	Note  string         `json:"note,omitempty"`
	Forms []LearningForm `json:"forms"`
}
type LearningForm struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
type LearningExample struct {
	Italian string `json:"italian"`
	English string `json:"english"`
}

type LearningSource interface {
	Explain(context.Context, WordDetails) (*LearningGuide, error)
}
type LearningAgent struct{ client JSONCompleter }

type schemaJSONCompleter interface {
	CompleteJSONSchema(context.Context, string, string, string, map[string]any) ([]byte, error)
}

func NewLearningAgent(client JSONCompleter) *LearningAgent { return &LearningAgent{client: client} }

func (a *LearningAgent) Explain(ctx context.Context, details WordDetails) (*LearningGuide, error) {
	evidence, err := json.Marshal(struct {
		Word    string            `json:"word"`
		Entries []DictionaryEntry `json:"entries"`
	}{details.Word, details.Entries})
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		var body []byte
		var err error
		if client, ok := a.client.(schemaJSONCompleter); ok {
			body, err = client.CompleteJSONSchema(ctx, learningSystemPrompt, fmt.Sprintf(learningUserPrompt, evidence), "italian_learning_guide", learningGuideSchema)
		} else {
			body, err = a.client.CompleteJSON(ctx, learningSystemPrompt, fmt.Sprintf(learningUserPrompt, evidence))
		}
		if err != nil {
			lastErr = err
			continue
		}
		var response struct {
			Learning LearningGuide `json:"learning"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			lastErr = fmt.Errorf("decode learning guide: %w", err)
			continue
		}
		if err := validateLearningGuide(response.Learning); err != nil {
			lastErr = err
			continue
		}
		return &response.Learning, nil
	}
	return nil, lastErr
}

var learningGuideSchema = map[string]any{
	"type": "object", "additionalProperties": false,
	"required": []string{"learning"},
	"properties": map[string]any{"learning": map[string]any{
		"type": "object", "additionalProperties": false,
		"required": []string{"summary", "grammar_note", "formation_rules", "tables", "patterns", "examples"},
		"properties": map[string]any{
			"summary":         map[string]any{"type": "string"},
			"grammar_note":    map[string]any{"type": "string"},
			"formation_rules": map[string]any{"type": "array", "maxItems": 6, "items": map[string]any{"type": "string"}},
			"tables": map[string]any{"type": "array", "maxItems": 4, "items": map[string]any{
				"type": "object", "additionalProperties": false, "required": []string{"title", "note", "forms"},
				"properties": map[string]any{
					"title": map[string]any{"type": "string"}, "note": map[string]any{"type": "string"},
					"forms": map[string]any{"type": "array", "maxItems": 6, "items": map[string]any{
						"type": "object", "additionalProperties": false, "required": []string{"label", "value"},
						"properties": map[string]any{"label": map[string]any{"type": "string"}, "value": map[string]any{"type": "string"}},
					}},
				},
			}},
			"patterns": map[string]any{"type": "array", "maxItems": 5, "items": map[string]any{"type": "string"}},
			"examples": map[string]any{"type": "array", "minItems": 4, "maxItems": 6, "items": map[string]any{
				"type": "object", "additionalProperties": false, "required": []string{"italian", "english"},
				"properties": map[string]any{"italian": map[string]any{"type": "string"}, "english": map[string]any{"type": "string"}},
			}},
		},
	}},
}

func validateLearningGuide(guide LearningGuide) error {
	if guide.Summary == "" || guide.GrammarNote == "" || len(guide.Examples) < 4 {
		return errors.New("learning agent returned an incomplete guide")
	}
	for _, example := range guide.Examples {
		if example.Italian == "" || example.English == "" {
			return errors.New("learning agent returned an incomplete example")
		}
	}
	return nil
}
