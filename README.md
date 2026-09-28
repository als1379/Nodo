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

## Commands

```sh
go test ./...
go vet ./...
docker compose down
```
