# Vocabulary data source

Nodo imports its Italian learner vocabulary from *Profilo della lingua italiana* by the Università per Stranieri di Perugia.

- Project: https://www.unistrapg.it/profilo_lingua_italiana/site/
- Lists used: the official A1, A2, B1 and B2 lexical inventories
- Selection used by Nodo: single-word nouns and verbs
- Level policy: a lemma is assigned to the earliest official CEFR list in which it appears

The project describes the linguistic content expected at each CEFR level for learners of Italian as a non-native language. Its inventories were reviewed through a validation process involving Italian-language teachers and subject experts. This makes the level a teaching classification rather than an automatic label inferred only from corpus frequency.

The source pages do not define an A0 level, so Nodo does not invent one. They are lexical inventories rather than frequency lists; `curriculum_rank` records stable source order and must not be interpreted as word frequency.

The official site does not publish a clear open-data license for redistributing its word-to-level mapping. Keep the attribution above and obtain permission from the Università per Stranieri di Perugia before commercial redistribution of the imported dataset.

## Difficulty evidence

Difficulty scoring uses Montefinese, Vinson, Vigliocco and Ambrosini (2019), *Italian Age of Acquisition Norms for a Large Set of Words (ItAoA)*: https://doi.org/10.3389/fpsyg.2019.00278. The official workbook is downloaded from https://osf.io/3trg2/ and verified against its published SHA-256 checksum. ItAoA is distributed under CC BY 4.0.

Corpus frequency for model features comes from the Italian data distributed by `wordfreq`. See its source attribution and redistribution conditions at https://github.com/rspeer/wordfreq.
