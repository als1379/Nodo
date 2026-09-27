package quiz

const wordQuestionSystemPrompt = `Create a concise English vocabulary multiple-choice question.
Return only valid JSON with these fields:
- word: a string
- options: an array of exactly four short meanings
- correct_option: the zero-based index of the correct option, from 0 to 3
Exactly one option must be correct.`

const wordQuestionUserPrompt = "Create one word-meaning question."
