# Nodo API

Base REST API for Nodo, written in Go with PostgreSQL authentication.

## Run locally

Requirements: Go 1.26+ and Docker.

```sh
cp .env.example .env
docker compose up -d postgres
```

Load the variables from `.env`, then start the server. In PowerShell:

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

Login uses the same JSON shape at `POST /v1/auth/login`. Both endpoints return an opaque session token. Send it on authenticated requests:

```sh
curl http://localhost:8080/v1/auth/me \
  -H "Authorization: Bearer YOUR_TOKEN"
```

`POST /v1/auth/logout` invalidates the current token. `GET /health` checks service and database availability.

## Commands

```sh
go test ./...
go vet ./...
docker compose down
```
