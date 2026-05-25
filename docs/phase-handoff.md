# Expense Manager - Handoff

Tai lieu nay mo ta trang thai hien tai cua project.

## Nguyen tac lam viec voi user

- Luon tra loi bang tieng Viet.
- Lam theo phase nho, moi phase xong bao cao de user test/confirm.
- User muon test tren mot port duy nhat: `http://localhost:3000`.
- Truoc khi bat server phase moi, can dung server cu dang giu port 3000.
- Neu port 3000 bi chiem:
  ```bash
  lsof -nP -iTCP:3000 -sTCP:LISTEN
  kill <PID>
  ```
- Tai khoan test hien co neu data local con giu:
  ```text
  Email: test@example.com
  Password: password123
  ```

## Tong quan

Project co 2 thu muc chinh:

```text
backend/
frontend/
```

Backend chi dung Go + Gin. Khong con Node backend hay script JavaScript trong `backend/`.

Frontend hien tai la static HTML/CSS/JS trong `frontend/public/`, duoc Go server serve truc tiep.

## Cach chay

Tu repo root:

```bash
cd backend
go run ./cmd/server
```

Hoac:

```bash
cd backend
make dev
```

URL test:

```text
http://localhost:3000
```

Health check:

```bash
curl -s http://localhost:3000/api/health
```

Ket qua mong doi:

```json
{
  "ok": true,
  "storageDriver": "sqlite",
  "storage": "data/go-app.sqlite"
}
```

## Kiem tra

```bash
cd backend
go test ./...
```

Hoac:

```bash
cd backend
make test
```

## Cau truc quan trong

```text
backend/
  Makefile
  go.mod
  go.sum
  cmd/server/main.go
  internal/app/
  internal/adapter/httpapi/
  internal/adapter/store/jsonstore/
  internal/adapter/store/sqlitestore/
  internal/adapter/store/shared/
  internal/domain/
  internal/platform/
  internal/store/
  internal/usecase/
  migrations/001_init_sqlite.sql
  docs/

frontend/
  public/
    index.html
    app.js
    styles.css
```

## Feature hien co

- Dang ky, dang nhap, dang xuat.
- Vi mac dinh.
- Danh muc thu/chi mac dinh.
- Them/sua/xoa mem giao dich.
- Loc giao dich theo thang, type, category, wallet, text search.
- Dashboard thang: tong thu, tong chi, con lai, ti le tiet kiem, breakdown category.
- Ngan sach thang theo danh muc chi.
- Canh bao ngan sach 80% va 100%.
- Khoan thu/chi dinh ky.
- Tu sinh giao dich den han theo `nextRunAt`.
- API response co `requestId`, `code`, `message`, `details`.

## Backend Go

- Gin router nam tai `internal/adapter/httpapi`.
- Usecase layer nam tai `internal/usecase`.
- Domain types/errors nam tai `internal/domain`.
- Store contract nam tai `internal/store`.
- SQLite store chinh nam tai `internal/adapter/store/sqlitestore`.
- JSON store fallback nam tai `internal/adapter/store/jsonstore`.
- Static frontend path mac dinh: `../frontend/public`.

## Cau hinh runtime

| Bien | Mac dinh | Mo ta |
| --- | --- | --- |
| `PORT` | `3000` | Port HTTP |
| `STORE_DRIVER` | `sqlite` | `sqlite` hoac `json` |
| `SQLITE_FILE` | `data/go-app.sqlite` | SQLite DB |
| `DATA_FILE` | `data/go-app.db.json` | JSON fallback/import |
| `SQLITE_IMPORT_JSON` | giong `DATA_FILE` | Import lan dau sang SQLite |
| `PUBLIC_DIR` | `../frontend/public` | Static frontend |
