# Vocabulary learning policy

## Two different difficulty values

Each word has a global score from 0 to 100. Direct matches use human age-of-acquisition ratings from ItAoA; other words use predictions from a versioned ridge-regression model trained on those ratings. Features include Italian corpus frequency, word length, approximate syllable count, part of speech and character n-grams. Source, estimated acquisition age, model version and scoring date remain in the database.

This is a proxy for lexical difficulty, not a calibrated measure of difficulty for an adult learning Italian. Native acquisition age, corpus frequency and personal familiarity are different concepts. Predicted words carry more uncertainty. The stored confidence values are metadata heuristics, not probabilities of correctness. Source CEFR labels remain provenance only: study selection no longer filters by them.

FSRS separately tracks each user's memory difficulty, stability and review dates. These values must not be confused with the global word score.

## New-word curriculum

Only nouns and verbs enter the vocabulary deck. Grammar/function words belong in separate grammar activities.

The default personal target is 15/100, adjustable in Settings and saved to the user's account. New words are selected deterministically, using these priorities within each word class:

1. Unseen words in the comfortable band: target minus 8 through target plus 3.
2. Easier unseen gaps below that band.
3. Harder unseen words if the earlier bands are exhausted.

Within a band, choose scores closest to target minus 3, then higher corpus frequency, then stable word ID. Alternate available nouns and verbs. Only active, scored words are introduced; words already reserved in unfinished sessions are not allocated again. Previously studied words remain reviewable even if a later vocabulary import retires them.

After a completed learning session of at least 10 answers, raise the target by 2 when recall is at least 85%; lower it by 2 below 60%; otherwise keep it. Clamp to 0-100. Review-only sessions do not change the target. This is a conservative application heuristic, not an FSRS feature or a scientifically validated proficiency test. A saved session keeps its original words when settings change.

## Retrieval and review

Nodo uses the official Go FSRS implementation with default parameters. The two self-assessment choices mean:

- **Still learning**: FSRS Again (rating 1).
- **I remembered**: FSRS Good (rating 3).

The learner should attempt recall before revealing, then rate what they remembered, not what they just read. The revealed card includes its meaning, a usage note and an example when available. The full word lesson provides deeper explanations. After rating, the card stays visible so the learner can read and say the example before continuing.

A learning session contains up to 20 saved cards: up to 10 due reviews, with new words filling the remaining places. If fewer new words exist, more due reviews fill the session. Reviews and new words alternate. Due reviews prioritize repeated misses, then the earliest due date. Every card is visibly tagged New or Review.

A separate review session contains up to 20 previously answered words: due cards first, then other words still being learned. It never introduces an unseen word. This allows optional early practice; the next scheduled review remains the recommended time. Repeated misses receive higher priority within those groups.

Sessions resume their remaining cards after leaving or refreshing. Allocation alone does not create learning progress. Answers and review logs are saved transactionally; repeated identical submissions are idempotent. Concurrent starts for one user resume the same unfinished session of that mode.

## Honest progress

- **Learning words**: answered at least once, but not yet meeting the learned criteria.
- **Learned words**: FSRS Review state, at least three successful recalls, and a scheduled interval of at least seven days.
- A failed recall can move a learned word back into Learning.
- Learned does not mean permanent mastery: due learned words still appear in reviews.
- **Practiced in 24 hours** counts distinct answered words, not opened cards.
- The session summary includes saved answers from before a pause.

The learned threshold is an application convention for a useful library, not a guarantee of retention. Both libraries are searchable and independent of sessions.

## Loading and failure behavior

Starting a session queues quick meanings for all its cards, with four concurrent requests. A separate queue prepares teaching lessons for the current and next three words, one at a time. The current word moves to the front of pending work. Quick meanings do not wait behind LLM requests. The card appears immediately; reveal uses the prefetched result, with a retry if the source fails. A first uncached external request can still take time.

Valid PostgreSQL lesson-cache hits skip both external services. Full dictionary evidence loads on demand. Successful dictionary entries are also cached in memory for one hour, with a bounded size. Concurrent requests for the same word share work; a canceled browser request does not cancel useful shared generation. Failures and incomplete lessons are not cached. A lesson outage does not prevent rating a word whose dictionary meaning is available.

## References

- [FSRS algorithm](https://github.com/open-spaced-repetition/awesome-fsrs/wiki/The-Algorithm) and [official Go implementation](https://github.com/open-spaced-repetition/go-fsrs).
- [SuperMemo's twenty rules](https://www.supermemo.com/en/blog/twenty-rules-of-formulating-knowledge): understand first, build on basics and keep individual recall tasks small. These inform the UI; they do not validate Nodo's numeric score thresholds.
- [Vocabulary provenance and limitations](data-sources.md).
