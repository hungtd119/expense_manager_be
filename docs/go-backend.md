# Go Backend

Backend hien tai chi dung Go + Gin. Khong con Node backend hay script JavaScript trong `backend/`.

## Chay local

Chay cac lenh tu thu muc `backend/`:

```bash
cd backend
go run ./cmd/server
```

Hoac:

```bash
make dev
```

Mo trinh duyet tai:

```text
http://localhost:3000
```

Go backend serve static frontend trong `../frontend/public/`.

## Kiem tra

```bash
go test ./...
```

Hoac:

```bash
make test
```

## Cau truc

```text
cmd/server/main.go
internal/app/                  # Config, store wiring, HTTP server lifecycle
internal/adapter/httpapi/       # Gin router, handlers, middleware, presenters, API tests
internal/adapter/store/jsonstore/
internal/adapter/store/sqlitestore/
internal/adapter/store/shared/
internal/domain/
internal/platform/
internal/store/
internal/usecase/
migrations/001_init_sqlite.sql
```

## HTTP Router

- Dung `github.com/gin-gonic/gin` trong `internal/adapter/httpapi`.
- `cmd/server/main.go` wire `app.Run(httpapi.NewHandler)`.
- Handlers goi usecase services va giu response contract hien tai.
- Static frontend duoc serve qua `PUBLIC_DIR`, mac dinh `../frontend/public`.

## Tests

- `go test ./...`: chay tat ca test Go.
- `go test ./internal/adapter/httpapi/...`: API contract bang `httptest` + `memstore`.
- `go test ./internal/usecase/...`: business logic unit tests.
- `go test ./internal/adapter/store/sqlitestore/...`: SQLite repository integration tests.

## Cau hinh

| Bien | Mac dinh | Mo ta |
|------|----------|-------|
| `PORT` | `3000` | Port HTTP |
| `STORE_DRIVER` | `sqlite` | `sqlite` hoac `json` |
| `DATA_FILE` | `data/go-app.db.json` | JSON fallback path |
| `SQLITE_FILE` | `data/go-app.sqlite` | SQLite path |
| `SQLITE_IMPORT_JSON` | giong `DATA_FILE` | Import lan dau sang SQLite neu co file JSON |
| `PUBLIC_DIR` | `../frontend/public` | Static frontend |
| `CORS_ORIGINS` | `http://localhost:3000,http://127.0.0.1:3000` | Danh sach origin CSV |
| `AUTH_RATE_PER_MINUTE` | `30` | Rate limit register/login theo IP |
| `PASSWORD_MIN_LENGTH` | `8` | Do dai mat khau toi thieu |
| `PASSWORD_REQUIRE_LETTER` | `true` | Mat khau phai co chu cai |
| `PASSWORD_REQUIRE_DIGIT` | `true` | Mat khau phai co chu so |
| `SHUTDOWN_TIMEOUT` | `10s` | Timeout graceful shutdown |
| `READ_HEADER_TIMEOUT` | `5s` | Timeout doc request header |
