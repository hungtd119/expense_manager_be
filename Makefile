.PHONY: dev dev-json test test-api test-usecase test-config

dev:
	go run ./cmd/server

dev-json:
	STORE_DRIVER=json DATA_FILE=data/go-app.db.json go run ./cmd/server

test:
	go test ./...

test-api:
	go test ./internal/adapter/httpapi/...

test-usecase:
	go test ./internal/usecase/...

test-config:
	go test ./internal/platform/config/...
