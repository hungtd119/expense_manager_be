# Data Schema

App mac dinh dung SQLite entity tables tai `data/go-app.sqlite`. API/UI van nhan shape camelCase nhu cac entity duoi day. Co the chay JSON fallback bang `STORE_DRIVER=json DATA_FILE=data/go-app.db.json`.

Storage logic nam trong Go store adapters tai `backend/internal/adapter/store/`. Static frontend nam tai `frontend/public/`. Neu co file JSON import cu, SQLite migration version 1 se import sang cac table rieng va ghi dau trong `schema_migrations`.

## SQLite tables

| Table | Purpose |
| --- | --- |
| schema_migrations | Theo doi migration da chay |
| users | Tai khoan |
| sessions | Token session da hash |
| wallets | Vi cua user |
| categories | Danh muc mac dinh va danh muc rieng user |
| transactions | Giao dich thu/chi, gom giao dich sinh tu dinh ky |
| budgets | Ngan sach thang |
| recurring_transactions | Khoan thu/chi dinh ky |
| notification_rules | Placeholder cho rule thong bao sau nay |

## SQLite indexes

| Index | Purpose |
| --- | --- |
| idx_transactions_user_month | Lay giao dich theo user/thang |
| idx_transactions_recurring_run | Chong sinh trung giao dich dinh ky theo `sourceRecurringId + recurringRunAt` |
| idx_budgets_user_month | Lay ngan sach theo user/thang |
| idx_budgets_unique_month_category | Moi danh muc chi chi co mot ngan sach active trong thang |
| idx_recurring_user_due | Quet khoan dinh ky den han |
| idx_sessions_expiry | Ho tro don session het han sau nay |

## users

| Field | Type | Note |
| --- | --- | --- |
| id | uuid | Primary key |
| email | string | Unique, lowercase |
| name | string | Display name |
| passwordHash | string | PBKDF2 hash |
| passwordSalt | string | Random salt |
| createdAt | ISO datetime | Created time |
| updatedAt | ISO datetime | Updated time |

## sessions

| Field | Type | Note |
| --- | --- | --- |
| tokenHash | string | SHA-256 token hash |
| userId | uuid | Owner |
| createdAt | ISO datetime | Created time |
| expiresAt | ISO datetime | Expiry time |

## wallets

| Field | Type | Note |
| --- | --- | --- |
| id | uuid | Primary key |
| userId | uuid | Owner |
| name | string | Wallet name |
| currency | string | VND by default |
| balanceInitial | number | Initial balance |
| createdAt | ISO datetime | Created time |

## categories

| Field | Type | Note |
| --- | --- | --- |
| id | uuid | Primary key |
| userId | uuid/null | Null means default category |
| name | string | Category name |
| type | income/expense | Transaction type |
| icon | string | Icon key |
| color | string | Hex color |
| isDefault | boolean | System category |

## Planned entities

- `notificationRules`

## transactions

| Field | Type | Note |
| --- | --- | --- |
| id | uuid | Primary key |
| userId | uuid | Owner |
| walletId | uuid | Wallet |
| categoryId | uuid | Category |
| type | income/expense | Transaction type |
| amount | number | Positive amount |
| note | string | Optional note |
| transactionDate | YYYY-MM-DD | Local date selected by user |
| syncStatus | string | `synced` in Phase 1 |
| createdAt | ISO datetime | Created time |
| updatedAt | ISO datetime | Updated time |
| deletedAt | ISO datetime/null | Soft delete |

## recurringTransactions

| Field | Type | Note |
| --- | --- | --- |
| id | uuid | Primary key |
| userId | uuid | Owner |
| walletId | uuid | Wallet |
| categoryId | uuid | Category |
| type | income/expense | Transaction type |
| amount | number | Positive amount |
| note | string | Optional note copied to generated transactions |
| frequency | daily/weekly/monthly | Recurrence interval |
| nextRunAt | YYYY-MM-DDTHH:mm | Next due local datetime |
| nextRunDate | YYYY-MM-DD | Date portion for compatibility |
| active | boolean | Whether the recurrence is active |
| createdAt | ISO datetime | Created time |
| updatedAt | ISO datetime | Updated time |
| deletedAt | ISO datetime/null | Soft delete |

Generated transactions include:

| Field | Type | Note |
| --- | --- | --- |
| sourceRecurringId | uuid | Recurring transaction id |
| recurringRunAt | YYYY-MM-DDTHH:mm | Due datetime used to generate this transaction |
| recurringRunDate | YYYY-MM-DD | Due date used to generate this transaction |

## budgets

| Field | Type | Note |
| --- | --- | --- |
| id | uuid | Primary key |
| userId | uuid | Owner |
| categoryId | uuid | Expense category |
| amountLimit | number | Monthly budget limit |
| period | string | `monthly` in Phase 3 |
| startDate | YYYY-MM-DD | First day of budget month |
| endDate | YYYY-MM-DD | First day of next month |
| createdAt | ISO datetime | Created time |
| updatedAt | ISO datetime | Updated time |
| deletedAt | ISO datetime/null | Soft delete |
