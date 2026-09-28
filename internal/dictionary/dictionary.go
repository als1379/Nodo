package dictionary

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
)

type DictionarySource interface {
	LookupItalian(context.Context, string) ([]byte, error)
}

type Dictionary struct {
	source  DictionarySource
	teacher LearningSource
	mu      sync.RWMutex
	lessons map[string]*LearningGuide
}

type WordDetails struct {
	Word            string            `json:"word"`
	Entries         []DictionaryEntry `json:"entries"`
	SourceURL       string            `json:"source_url"`
	Learning        *LearningGuide    `json:"learning,omitempty"`
	LearningError   string            `json:"learning_error,omitempty"`
	LearningFailure error             `json:"-"`
}

type DictionaryEntry struct {
	PartOfSpeech  string        `json:"part_of_speech"`
	Grammar       []string      `json:"grammar,omitempty"`
	Categories    []string      `json:"categories,omitempty"`
	Definitions   []Definition  `json:"definitions"`
	Pronunciation []string      `json:"pronunciation,omitempty"`
	Forms         []WordForm    `json:"forms,omitempty"`
	Synonyms      []RelatedWord `json:"synonyms,omitempty"`
	Antonyms      []RelatedWord `json:"antonyms,omitempty"`
	Related       []RelatedWord `json:"related,omitempty"`
	Etymology     string        `json:"etymology,omitempty"`
}

type Definition struct {
	Meaning  string        `json:"meaning"`
	Tags     []string      `json:"tags,omitempty"`
	Topics   []string      `json:"topics,omitempty"`
	Remarks  []string      `json:"remarks,omitempty"`
	Examples []string      `json:"examples,omitempty"`
	Synonyms []RelatedWord `json:"synonyms,omitempty"`
	Antonyms []RelatedWord `json:"antonyms,omitempty"`
}

type WordForm struct {
	Form   string   `json:"form"`
	Tags   []string `json:"tags,omitempty"`
	Source string   `json:"source,omitempty"`
}

type RelatedWord struct {
	Word  string   `json:"word"`
	Sense string   `json:"sense,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

type rawRelatedWord struct {
	Word  string   `json:"word"`
	Sense string   `json:"sense"`
	Tags  []string `json:"tags"`
}

func NewDictionary(source DictionarySource, teacher ...LearningSource) *Dictionary {
	dictionary := &Dictionary{source: source, lessons: make(map[string]*LearningGuide)}
	if len(teacher) > 0 {
		dictionary.teacher = teacher[0]
	}
	return dictionary
}

func (d *Dictionary) Lookup(ctx context.Context, word string) (WordDetails, error) {
	body, err := d.source.LookupItalian(ctx, word)
	if err != nil {
		return WordDetails{}, err
	}
	var response struct {
		Word    string `json:"word"`
		Entries []struct {
			POS           string           `json:"pos"`
			EtymologyText string           `json:"etymology_text"`
			Tags          []string         `json:"tags"`
			Categories    []string         `json:"categories"`
			Synonyms      []rawRelatedWord `json:"synonyms"`
			Antonyms      []rawRelatedWord `json:"antonyms"`
			Related       []rawRelatedWord `json:"related"`
			Senses        []struct {
				Glosses    []string         `json:"glosses"`
				RawGlosses []string         `json:"raw_glosses"`
				Tags       []string         `json:"tags"`
				Topics     []string         `json:"topics"`
				Categories []string         `json:"categories"`
				Synonyms   []rawRelatedWord `json:"synonyms"`
				Antonyms   []rawRelatedWord `json:"antonyms"`
				Examples   []struct {
					Text        string `json:"text"`
					Translation string `json:"translation"`
				} `json:"examples"`
			} `json:"senses"`
			Sounds []struct {
				IPA string `json:"ipa"`
			} `json:"sounds"`
			Forms []struct {
				Form   string   `json:"form"`
				Tags   []string `json:"tags"`
				Source string   `json:"source"`
			} `json:"forms"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return WordDetails{}, fmt.Errorf("decode WiktAPI response: %w", err)
	}
	details := WordDetails{Word: response.Word, SourceURL: "https://en.wiktionary.org/wiki/" + url.PathEscape(word)}
	for _, raw := range response.Entries {
		entry := DictionaryEntry{PartOfSpeech: raw.POS, Grammar: raw.Tags, Categories: raw.Categories, Etymology: raw.EtymologyText, Synonyms: mapRelated(raw.Synonyms), Antonyms: mapRelated(raw.Antonyms), Related: mapRelated(raw.Related)}
		for _, sound := range raw.Sounds {
			if sound.IPA != "" {
				entry.Pronunciation = appendUnique(entry.Pronunciation, sound.IPA)
			}
		}
		for _, form := range raw.Forms {
			if form.Form != "" && form.Form != word {
				entry.Forms = append(entry.Forms, WordForm{Form: form.Form, Tags: form.Tags, Source: form.Source})
			}
		}
		for _, sense := range raw.Senses {
			for _, tag := range sense.Tags {
				if isGrammarTag(tag) {
					entry.Grammar = appendUnique(entry.Grammar, tag)
				}
			}
			if len(sense.Glosses) == 0 {
				continue
			}
			definition := Definition{Meaning: sense.Glosses[0], Tags: sense.Tags, Topics: sense.Topics, Synonyms: mapRelated(sense.Synonyms), Antonyms: mapRelated(sense.Antonyms)}
			for _, category := range sense.Categories {
				entry.Categories = appendUnique(entry.Categories, category)
			}
			for _, rawGloss := range sense.RawGlosses {
				if rawGloss != "" && rawGloss != definition.Meaning {
					definition.Remarks = appendUnique(definition.Remarks, rawGloss)
				}
			}
			for _, example := range sense.Examples {
				if example.Text == "" {
					continue
				}
				text := example.Text
				if example.Translation != "" {
					text += " - " + example.Translation
				}
				definition.Examples = append(definition.Examples, text)
			}
			entry.Definitions = append(entry.Definitions, definition)
		}
		if entry.PartOfSpeech == "" {
			entry.PartOfSpeech = inferPartOfSpeech(entry)
		}
		if len(entry.Definitions) > 0 {
			details.Entries = append(details.Entries, entry)
		}
	}
	if details.Word == "" {
		details.Word = word
	}
	if len(details.Entries) == 0 {
		return WordDetails{}, fmt.Errorf("no Italian dictionary entry found for %q", word)
	}
	if d.teacher != nil {
		d.mu.RLock()
		guide := d.lessons[details.Word]
		d.mu.RUnlock()
		if guide == nil {
			var err error
			guide, err = d.teacher.Explain(ctx, details)
			if err != nil {
				details.LearningFailure = err
				details.LearningError = learningErrorMessage(err)
			} else {
				d.mu.Lock()
				d.lessons[details.Word] = guide
				d.mu.Unlock()
			}
		}
		details.Learning = guide
	}
	return details, nil
}

func inferPartOfSpeech(entry DictionaryEntry) string {
	hasGender, hasPlural := false, false
	for _, grammar := range entry.Grammar {
		if grammar == "feminine" || grammar == "masculine" {
			hasGender = true
		}
	}
	for _, form := range entry.Forms {
		for _, tag := range form.Tags {
			if tag == "plural" {
				hasPlural = true
			}
		}
	}
	if hasGender && hasPlural {
		return "noun"
	}
	for _, form := range entry.Forms {
		for _, tag := range form.Tags {
			if tag == "indicative" || tag == "infinitive" || tag == "gerund" {
				return "verb"
			}
		}
	}
	return ""
}

func learningErrorMessage(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "not configured"):
		return "The teaching agent is not configured. Add OPENROUTER_API_KEY to .env."
	case strings.Contains(message, "status 401"), strings.Contains(message, "status 403"):
		return "OpenRouter rejected the API key. Add a valid OPENROUTER_API_KEY to .env."
	case strings.Contains(message, "status 429"):
		return "The teaching agent is busy. Please try this word again shortly."
	default:
		return "The teaching agent could not build this lesson. Please try again."
	}
}

func mapRelated(words []rawRelatedWord) []RelatedWord {
	result := make([]RelatedWord, 0, len(words))
	for _, word := range words {
		if word.Word != "" {
			result = append(result, RelatedWord{Word: word.Word, Sense: word.Sense, Tags: word.Tags})
		}
	}
	return result
}

func isGrammarTag(tag string) bool {
	switch tag {
	case "masculine", "feminine", "neuter", "common-gender", "transitive", "intransitive", "ambitransitive", "reflexive", "pronominal", "impersonal", "countable", "uncountable":
		return true
	default:
		return false
	}
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
