package store

import (
	"time"

	"expense-manager-mvp/internal/domain"
)

type Store interface {
	Ensure() error
	Read() (domain.DB, error)
	Write(domain.DB) error
	Driver() string
	Location() string
	ListTransactionsForMonth(userID string, bounds domain.MonthBounds) ([]domain.Transaction, error)
	CreateTransaction(domain.Transaction) error
	UpdateTransaction(domain.Transaction) error
	SoftDeleteTransaction(userID string, id string, deletedAt string) error
	ListBudgetsForMonth(userID string, bounds domain.MonthBounds) ([]domain.Budget, error)
	CreateBudget(domain.Budget) error
	UpdateBudget(domain.Budget) error
	SoftDeleteBudget(userID string, id string, deletedAt string) error
	ListRecurring(userID string) ([]domain.RecurringTransaction, error)
	CreateRecurring(domain.RecurringTransaction) error
	UpdateRecurring(domain.RecurringTransaction) error
	SoftDeleteRecurring(userID string, id string, deletedAt string) error
	ProcessDueRecurring(userID string, untilAt string) (domain.RecurringResult, error)
	FindUserByEmail(email string) (domain.User, error)
	CreateUserWithWalletAndSession(user domain.User, wallet domain.Wallet, session domain.Session) error
	CreateSession(domain.Session) error
	DeleteSession(tokenHash string) error
	DeleteExpiredSessions(now time.Time) (bool, error)
}
