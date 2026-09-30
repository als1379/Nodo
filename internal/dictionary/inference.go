package dictionary

func InferPartOfSpeech(entry DictionaryEntry) string {
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

func IsGrammarTag(tag string) bool {
	switch tag {
	case "masculine", "feminine", "neuter", "common-gender", "transitive", "intransitive", "ambitransitive", "reflexive", "pronominal", "impersonal", "countable", "uncountable":
		return true
	default:
		return false
	}
}
