package flashcard

import (
	"errors"
	"strings"
	"time"
)

type SessionMode string

const (
	ModeLearn  SessionMode = "learn"
	ModeReview SessionMode = "review"
)

var ErrInvalidSessionMode = errors.New("mode must be learn or review")

func ParseSessionMode(value string) (SessionMode, error) {
	mode := SessionMode(strings.ToLower(strings.TrimSpace(value)))
	if mode == "" {
		return ModeLearn, nil
	}
	if mode != ModeLearn && mode != ModeReview {
		return "", ErrInvalidSessionMode
	}
	return mode, nil
}

type Card struct {
	ID           int64    `json:"id"`
	Word         string   `json:"word"`
	Difficulty   *float64 `json:"difficulty_score,omitempty"`
	PartOfSpeech string   `json:"part_of_speech,omitempty"`
	Review       bool     `json:"review"`
}

type Session struct {
	ID       string      `json:"id"`
	Target   float64     `json:"difficulty_target"`
	Total    int         `json:"total"`
	Answered int         `json:"answered"`
	Recalled int         `json:"recalled"`
	Mode     SessionMode `json:"mode"`
	Cards    []Card      `json:"cards"`
}

type Settings struct {
	DifficultyTarget float64 `json:"difficulty_target"`
}

type Overview struct {
	Settings
	Learning        int        `json:"learning"`
	Learned         int        `json:"learned"`
	Due             int        `json:"due"`
	ReviewAvailable int        `json:"review_available"`
	Practiced       int        `json:"practiced_24h"`
	NextReview      *time.Time `json:"next_review,omitempty"`
}

type WordProgress struct {
	Card
	State        string `json:"state"`
	KnownCount   int    `json:"known_count"`
	UnknownCount int    `json:"unknown_count"`
	Due          string `json:"due"`
}
