package dictionary

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type LearningGuide struct {
	Meaning        string            `json:"meaning"`
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

func (g LearningGuide) Validate() error {
	if strings.TrimSpace(g.Meaning) == "" || strings.TrimSpace(g.Summary) == "" || strings.TrimSpace(g.GrammarNote) == "" || len(g.Examples) < 4 {
		return errors.New("learning agent returned an incomplete guide")
	}
	for _, example := range g.Examples {
		if strings.TrimSpace(example.Italian) == "" || strings.TrimSpace(example.English) == "" {
			return errors.New("learning agent returned an incomplete example")
		}
	}
	return nil
}

type statusCoder interface{ StatusCode() int }
type configurationError interface{ IsConfigurationError() bool }

func lessonErrorMessage(err error) string {
	var configuration configurationError
	if errors.As(err, &configuration) && configuration.IsConfigurationError() {
		return "Word lessons are temporarily unavailable. You can still study the dictionary meaning."
	}
	var provider statusCoder
	if errors.As(err, &provider) {
		switch provider.StatusCode() {
		case 401, 403:
			return "Word lessons are temporarily unavailable. Please try again later."
		case 429:
			return "The teaching agent is busy. Please try this word again shortly."
		case 402:
			return "Word lessons are temporarily unavailable. Please try again later."
		}
	}
	return "The teaching agent could not build this lesson. Please try again."
}

func decodeLesson(body []byte) (*LearningGuide, error) {
	var response struct {
		Learning LearningGuide `json:"learning"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode learning guide: %w", err)
	}
	if err := response.Learning.Validate(); err != nil {
		return nil, err
	}
	return &response.Learning, nil
}
