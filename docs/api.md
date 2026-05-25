# API

Base URL khi chay local:

```text
http://localhost:3000/api
```

API routing hien nam trong Go Gin adapter `internal/adapter/httpapi/`; business rules nam trong `internal/usecase/`.
Chay cac lenh tu thu muc `backend/`; static frontend nam tai `../frontend/public/`.

```bash
go run ./cmd/server
```

## Response contract

Moi response thanh cong co the kem `requestId` de trace log/debug.

Response loi giu `error` de tuong thich frontend hien tai, dong thoi co shape chuan hon:

```json
{
  "ok": false,
  "error": "Email khong hop le.",
  "code": "VALIDATION_ERROR",
  "message": "Email khong hop le.",
  "details": {
    "field": "email"
  },
  "requestId": "..."
}
```

Body JSON bi gioi han 1MB. Cac list endpoint co the tra `meta`.

## POST /api/auth/register

Request:

```json
{
  "name": "Hung",
  "email": "hung@example.com",
  "password": "secret123"
}
```

Response:

```json
{
  "token": "...",
  "user": {
    "id": "...",
    "name": "Hung",
    "email": "hung@example.com"
  }
}
```

## POST /api/auth/login

Request:

```json
{
  "email": "hung@example.com",
  "password": "secret123"
}
```

## POST /api/auth/logout

Header:

```text
Authorization: Bearer <token>
```

## GET /api/me

Header:

```text
Authorization: Bearer <token>
```

Response:

```json
{
  "user": {
    "id": "...",
    "name": "Hung",
    "email": "hung@example.com"
  }
}
```

## GET /api/categories

Header:

```text
Authorization: Bearer <token>
```

Response:

```json
{
  "categories": [
    {
      "id": "...",
      "name": "An uong",
      "type": "expense",
      "icon": "utensils",
      "color": "#ef4444",
      "isDefault": true
    }
  ]
}
```

## GET /api/wallets

Header:

```text
Authorization: Bearer <token>
```

## GET /api/transactions?month=2026-05

Header:

```text
Authorization: Bearer <token>
```

Query:

| Param | Note |
| --- | --- |
| month | Bat buoc, format `YYYY-MM` |
| page | Tuy chon, mac dinh `1` |
| pageSize | Tuy chon, mac dinh `50`, toi da `100` |
| type | Tuy chon: `income` hoac `expense` |
| categoryId | Tuy chon |
| walletId | Tuy chon |
| q | Tuy chon, tim trong note/category/wallet |

Response:

```json
{
  "transactions": [],
  "meta": {
    "page": 1,
    "pageSize": 50,
    "total": 0,
    "totalPages": 1,
    "filters": {
      "type": null,
      "categoryId": null,
      "walletId": null,
      "q": null
    }
  },
  "requestId": "..."
}
```

## GET /api/dashboard?month=2026-05

Header:

```text
Authorization: Bearer <token>
```

Response:

```json
{
  "month": "2026-05",
  "dashboard": {
    "totals": {
      "income": 12000000,
      "expense": 3500000,
      "balance": 8500000,
      "savingsRate": 70.8
    },
    "counts": {
      "all": 12,
      "income": 1,
      "expense": 11
    },
    "expenseByCategory": [],
    "topCategory": null,
    "averageExpense": 318182,
    "insight": "An uong la danh muc chi nhieu nhat, chiem 35.2% tong chi."
  }
}
```

## POST /api/transactions

Header:

```text
Authorization: Bearer <token>
```

Request:

```json
{
  "walletId": "...",
  "categoryId": "...",
  "type": "expense",
  "amount": 150000,
  "note": "Ca phe sang",
  "transactionDate": "2026-05-21"
}
```

## PUT /api/transactions/:id

Cap nhat giao dich cua user hien tai.

## DELETE /api/transactions/:id

Xoa mem giao dich bang `deletedAt`.

## GET /api/budgets?month=2026-05

Header:

```text
Authorization: Bearer <token>
```

Response:

```json
{
  "budgets": [
    {
      "id": "...",
      "categoryName": "An uong",
      "amountLimit": 2000000,
      "spent": 1200000,
      "remaining": 800000,
      "percentUsed": 60,
      "alertLevel": "ok",
      "alertMessage": "Dang trong gioi han."
    }
  ],
  "meta": {
    "total": 1
  },
  "requestId": "..."
}
```

## POST /api/budgets?month=2026-05

Request:

```json
{
  "categoryId": "...",
  "amountLimit": 2000000
}
```

Chi tao ngan sach cho danh muc chi. Moi danh muc chi chi co mot ngan sach trong mot thang.

## PUT /api/budgets/:id

Cap nhat danh muc hoac han muc ngan sach.

## DELETE /api/budgets/:id

Xoa mem ngan sach bang `deletedAt`.

## GET /api/recurring-transactions

Header:

```text
Authorization: Bearer <token>
```

Response:

```json
{
  "recurringTransactions": [
    {
      "id": "...",
      "type": "expense",
      "amount": 3000000,
      "frequency": "monthly",
      "nextRunAt": "2026-06-01T11:35",
      "nextRunDate": "2026-06-01",
      "active": true
    }
  ],
  "generatedCount": 0,
  "meta": {
    "total": 1
  },
  "requestId": "..."
}
```

Endpoint nay se tu sinh cac giao dich dinh ky da den han truoc khi tra ve danh sach.

## POST /api/recurring-transactions

Request:

```json
{
  "walletId": "...",
  "categoryId": "...",
  "type": "expense",
  "amount": 3000000,
  "note": "Tien nha",
  "frequency": "monthly",
  "nextRunAt": "2026-05-01T11:35",
  "active": true
}
```

## PUT /api/recurring-transactions/:id

Cap nhat khoan dinh ky. Neu `nextRunAt` da den han, he thong se sinh giao dich tuong ung.

## DELETE /api/recurring-transactions/:id

Tat va xoa mem khoan dinh ky. Cac giao dich da sinh truoc do duoc giu lai.
