# Nodo API

Base REST API for Nodo, written in Go with PostgreSQL authentication.

## Run locally

Requirements: Docker. Copy the environment template and add your OpenRouter key:

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

## Word quiz

Set `OPENROUTER_API_KEY` in your local `.env`. `OPENROUTER_MODEL` defaults to the free-model router `openrouter/free`.

Get a question:

```sh
curl http://localhost:8080/v1/quiz/word
```

Answer using the question ID and a zero-based option number. The response says whether the choice was correct; request the question endpoint again for the next word:

```sh
curl -X POST http://localhost:8080/v1/quiz/word/QUESTION_ID/answer \
  -H "Content-Type: application/json" \
  -d '{"option":0}'
```

Questions and answers are held in memory and disappear when the API restarts.

## Commands

```sh
go test ./...
go vet ./...
docker compose down
```
