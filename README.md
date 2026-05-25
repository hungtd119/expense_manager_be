# Expense Manager Backend

Backend hien tai chi dung Go + Gin. Static frontend nam tai `../frontend/public/` va duoc Go server serve truc tiep.

## Chay local

Tat ca lenh backend chay trong thu muc `backend/`:

```bash
cd backend
```

Chay backend Go voi SQLite mac dinh:

```bash
go run ./cmd/server
```

Hoac dung Makefile:

```bash
make dev
```

Mo trinh duyet tai:

```text
http://localhost:3000
```

Mac dinh app dung SQLite tai:

```text
data/go-app.sqlite
```

Neu can chay JSON store fallback:

```bash
STORE_DRIVER=json DATA_FILE=data/go-app.db.json go run ./cmd/server
```

## Kiem tra

```bash
go test ./...
```

Hoac:

```bash
make test
```

## Cau truc backend

- `cmd/server/main.go`: entrypoint process.
- `internal/app/`: bootstrap config, store, HTTP server.
- `internal/adapter/httpapi/`: Gin router, handlers, middleware, presenter va API tests.
- `internal/usecase/`: business logic cho auth, transaction, budget, recurring, dashboard, reference.
- `internal/domain/`: entity types va domain errors.
- `internal/store/`: store contract va in-memory store cho tests.
- `internal/adapter/store/sqlitestore/`: SQLite repository implementation.
- `internal/adapter/store/jsonstore/`: JSON fallback implementation.
- `migrations/001_init_sqlite.sql`: SQLite schema reference.
- `../frontend/public/`: frontend static files duoc backend serve.

## Feature hien co

- Dang ky, dang nhap, dang xuat.
- Vi mac dinh va danh muc thu/chi mac dinh.
- Them, sua, xoa mem giao dich thu/chi.
- Danh sach giao dich theo thang, filter va pagination.
- Dashboard thang: tong thu, tong chi, con lai, ty le tiet kiem, breakdown category.
- Ngan sach thang theo danh muc chi, canh bao 80% va 100%.
- Khoan thu/chi dinh ky, tu sinh giao dich den han theo `nextRunAt`.
- API error response co `code`, `message`, `details`, `requestId`.
