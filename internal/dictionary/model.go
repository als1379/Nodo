package dictionary

type WordDetails struct {
	Word          string            `json:"word"`
	Entries       []DictionaryEntry `json:"entries"`
	SourceURL     string            `json:"source_url"`
	Learning      *LearningGuide    `json:"learning,omitempty"`
	LearningError string            `json:"learning_error,omitempty"`
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
