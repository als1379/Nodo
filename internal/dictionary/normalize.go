package dictionary

import (
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const MaxWordLength = 80

var ErrWordNotFound = errors.New("Italian word not found")

var ErrInvalidWord = errors.New("word must be between 1 and 80 characters")

func NormalizeWord(word string) (string, error) {
	word = norm.NFC.String(strings.ToLower(strings.TrimSpace(word)))
	if word == "" || utf8.RuneCountInString(word) > MaxWordLength {
		return "", ErrInvalidWord
	}
	return word, nil
}
