package wiktapi

type response struct {
	Word    string     `json:"word"`
	Entries []rawEntry `json:"entries"`
}

type rawEntry struct {
	POS           string           `json:"pos"`
	EtymologyText string           `json:"etymology_text"`
	Tags          []string         `json:"tags"`
	Categories    []string         `json:"categories"`
	Synonyms      []rawRelatedWord `json:"synonyms"`
	Antonyms      []rawRelatedWord `json:"antonyms"`
	Related       []rawRelatedWord `json:"related"`
	Senses        []rawSense       `json:"senses"`
	Sounds        []rawSound       `json:"sounds"`
	Forms         []rawForm        `json:"forms"`
}

type rawSense struct {
	Glosses    []string         `json:"glosses"`
	RawGlosses []string         `json:"raw_glosses"`
	Tags       []string         `json:"tags"`
	Topics     []string         `json:"topics"`
	Categories []string         `json:"categories"`
	Synonyms   []rawRelatedWord `json:"synonyms"`
	Antonyms   []rawRelatedWord `json:"antonyms"`
	Examples   []rawExample     `json:"examples"`
}

type rawRelatedWord struct {
	Word  string   `json:"word"`
	Sense string   `json:"sense"`
	Tags  []string `json:"tags"`
}

type rawSound struct {
	IPA string `json:"ipa"`
}
type rawForm struct {
	Form   string   `json:"form"`
	Tags   []string `json:"tags"`
	Source string   `json:"source"`
}
type rawExample struct {
	Text        string `json:"text"`
	Translation string `json:"translation"`
}
