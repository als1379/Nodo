package wiktapi

import (
	"fmt"
	"net/url"

	"nodo/internal/dictionary"
)

func mapResponse(requestedWord string, response response) (dictionary.WordDetails, error) {
	details := dictionary.WordDetails{
		Word:      response.Word,
		SourceURL: "https://en.wiktionary.org/wiki/" + url.PathEscape(requestedWord),
	}
	for _, raw := range response.Entries {
		entry := mapEntry(requestedWord, raw)
		if len(entry.Definitions) > 0 {
			details.Entries = append(details.Entries, entry)
		}
	}
	if details.Word == "" {
		details.Word = requestedWord
	}
	if len(details.Entries) == 0 {
		return dictionary.WordDetails{}, fmt.Errorf("%w: %s", dictionary.ErrWordNotFound, requestedWord)
	}
	return details, nil
}

func mapEntry(word string, raw rawEntry) dictionary.DictionaryEntry {
	entry := dictionary.DictionaryEntry{
		PartOfSpeech: raw.POS, Grammar: raw.Tags, Categories: raw.Categories,
		Etymology: raw.EtymologyText, Synonyms: mapRelated(raw.Synonyms),
		Antonyms: mapRelated(raw.Antonyms), Related: mapRelated(raw.Related),
	}
	for _, sound := range raw.Sounds {
		if sound.IPA != "" {
			entry.Pronunciation = appendUnique(entry.Pronunciation, sound.IPA)
		}
	}
	for _, form := range raw.Forms {
		if form.Form != "" && form.Form != word {
			entry.Forms = append(entry.Forms, dictionary.WordForm{Form: form.Form, Tags: form.Tags, Source: form.Source})
		}
	}
	for _, sense := range raw.Senses {
		entry = mapSense(entry, sense)
	}
	if entry.PartOfSpeech == "" {
		entry.PartOfSpeech = dictionary.InferPartOfSpeech(entry)
	}
	return entry
}

func mapSense(entry dictionary.DictionaryEntry, sense rawSense) dictionary.DictionaryEntry {
	for _, tag := range sense.Tags {
		if dictionary.IsGrammarTag(tag) {
			entry.Grammar = appendUnique(entry.Grammar, tag)
		}
	}
	if len(sense.Glosses) == 0 {
		return entry
	}
	definition := dictionary.Definition{
		Meaning: sense.Glosses[0], Tags: sense.Tags, Topics: sense.Topics,
		Synonyms: mapRelated(sense.Synonyms), Antonyms: mapRelated(sense.Antonyms),
	}
	for _, category := range sense.Categories {
		entry.Categories = appendUnique(entry.Categories, category)
	}
	for _, gloss := range sense.RawGlosses {
		if gloss != "" && gloss != definition.Meaning {
			definition.Remarks = appendUnique(definition.Remarks, gloss)
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
	return entry
}

func mapRelated(words []rawRelatedWord) []dictionary.RelatedWord {
	result := make([]dictionary.RelatedWord, 0, len(words))
	for _, word := range words {
		if word.Word != "" {
			result = append(result, dictionary.RelatedWord{Word: word.Word, Sense: word.Sense, Tags: word.Tags})
		}
	}
	return result
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
