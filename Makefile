.PHONY: run test fmt db-up db-down

run:
	go run ./cmd/api

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go')

db-up:
	docker compose up -d postgres

db-down:
	docker compose down
