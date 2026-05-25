package sqlitestore

import (
	"expense-manager-mvp/internal/adapter/store/shared"
	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/platform"
)

type DB = domain.DB
type User = domain.User
type Session = domain.Session
type Wallet = domain.Wallet
type Category = domain.Category
type Transaction = domain.Transaction
type Budget = domain.Budget
type RecurringTransaction = domain.RecurringTransaction
type MonthBounds = domain.MonthBounds
type RecurringResult = domain.RecurringResult

var (
	errNotFound      = domain.ErrNotFound
	errAlreadyExists = domain.ErrAlreadyExists
)

func emptyDB() DB {
	return shared.EmptyDB()
}

func migrateShape(db *DB) bool {
	return shared.MigrateShape(db)
}

func nowISO() string {
	return platform.NowISO(platform.SystemClock{})
}

func uuid() string {
	return platform.CryptoIDGenerator{}.UUID()
}

func nextRunAt(currentRunAt string, frequency string) string {
	return shared.NextRunAt(currentRunAt, frequency)
}
