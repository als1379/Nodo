# Nodo API

Base REST API for Nodo, written in Go with PostgreSQL authentication.

## Run locally

Requirements: Docker. Copy the environment template:

```sh
cp .env.example .env
docker compose up --build
```

The frontend is available at `http://localhost:5173`; it proxies API calls to the Go container over Docker's internal network. For backend-only development, load the variables from `.env` and run Go directly:

```powershell
Get-Content .env | ForEach-Object {
  $name, $value = $_ -split '=', 2
  Set-Item -Path "Env:$name" -Value $value
}
go run ./cmd/api
```

Database migrations run automatically at startup. The API listens on `http://localhost:8080` by default.

## Authentication API

Register:

```sh
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"a-secure-password"}'
```

Login accepts `login` and `password` at `POST /v1/auth/login`. Registration and login return a signed HS256 JWT. Send it on authenticated requests:

```sh
curl http://localhost:8080/v1/auth/me \
  -H "Authorization: Bearer YOUR_TOKEN"
```

`POST /v1/auth/logout` invalidates the current token. `GET /health` checks service and database availability.

The OpenAPI 3.1 specification is available at `GET /openapi.json`, or through the frontend proxy at `GET /api/openapi.json`.

## Dictionary API

Look up an Italian word with an authenticated request:

```sh
curl http://localhost:8080/v1/dictionary/parlare \
  -H "Authorization: Bearer YOUR_TOKEN"
```

The dictionary module fetches structured lexical evidence from WiktAPI/Wiktionary, then asks the OpenRouter teaching agent to produce a focused learner lesson with grammar rules, useful forms, conjugation tables, patterns, and examples. Set `OPENROUTER_API_KEY` and `OPENROUTER_MODEL` in `.env` to enable lesson generation.

Generated lessons are cached in PostgreSQL by normalized Italian word. Cache hits return immediately, even if the external dictionary is unavailable. Add `?view=preview` for a quick meaning without an LLM call, or `?view=entry` for the complete dictionary. Duplicate lookups share work; failed lessons are never cached.

## Vocabulary flashcards

Start or resume a saved learning session:

```sh
curl -X POST http://localhost:8080/v1/flashcards/sessions \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"mode":"learn"}'
```

Use `"mode":"review"` for previously practiced learning words and due learned words. Reviews never add unseen vocabulary. Both modes save and resume progress.

`GET /v1/flashcards/overview` returns learning, learned, due and last-24-hour practice counts. `PUT /v1/flashcards/settings` saves `{"difficulty_target":15}` for future sessions. `GET /v1/flashcards/words?status=learning` (or `learned`) opens the corresponding library.

New vocabulary follows a personal difficulty target across the full noun/verb inventory. Source CEFR labels remain import metadata only. Answers and FSRS scheduling state are persisted per user. See [docs/learning-policy.md](docs/learning-policy.md) for selection, reviews, learned-word criteria and loading behavior.

Import or refresh the word list:

```powershell
$env:DATABASE_URL = "postgres://nodo:nodo@localhost:5432/nodo?sslmode=disable"
go run ./cmd/import-words
```

See [docs/data-sources.md](docs/data-sources.md) for provenance, attribution, and licensing notes.

Score all active words with human Italian age-of-acquisition ratings and a reproducible prediction model for unmatched words:

```powershell
python -m venv .venv-difficulty
.\.venv-difficulty\Scripts\python.exe -m pip install -r tools/requirements-difficulty.txt
$env:DATABASE_URL = "postgres://nodo:nodo@localhost:5432/nodo?sslmode=disable"
.\.venv-difficulty\Scripts\python.exe tools/score_words.py
```

## Commands

```sh
go test ./...
go vet ./...
docker compose down
```

Database integration tests use a temporary schema and clean it up automatically:

```powershell
$env:TEST_DATABASE_URL = "postgres://nodo:nodo@localhost:5432/nodo?sslmode=disable"
go test ./internal/flashcard -v
```


The browser smoke test uses real authentication and study APIs with a temporary account, and fixture dictionary responses to avoid LLM charges. It removes the account afterwards. Defaults match this local setup; override the URL/browser for other environments:

```powershell
npm install --prefix .cache/browser playwright
$env:TEST_FRONTEND_URL = "http://localhost:5175"
$env:TEST_BROWSER_PATH = "C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe"
node tools/test_frontend.cjs
```
