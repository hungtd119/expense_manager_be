package usecase

import (
	"strings"

	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/platform"
	"expense-manager-mvp/internal/store"
)

type TransactionService struct {
	store store.Store
	clock platform.Clock
	ids   platform.IDGenerator
}

func NewTransactionService(store store.Store) TransactionService {
	return TransactionService{store: store, clock: platform.SystemClock{}, ids: platform.CryptoIDGenerator{}}
}

func (s TransactionService) List(userID string, bounds domain.MonthBounds) ([]domain.Transaction, error) {
	if _, err := s.store.ProcessDueRecurring(userID, localDateTimeNow()); err != nil {
		return nil, err
	}
	return s.store.ListTransactionsForMonth(userID, bounds)
}

func (s TransactionService) Create(db *domain.DB, userID string, body map[string]any) (domain.Transaction, error) {
	value, err := validateTransaction(db, userID, body, nil)
	if err != nil {
		return domain.Transaction{}, err
	}
	now := platform.NowISO(s.clock)
	tx := domain.Transaction{
		ID:              s.ids.UUID(),
		UserID:          userID,
		WalletID:        value.WalletID,
		CategoryID:      value.CategoryID,
		Type:            value.Type,
		Amount:          value.Amount,
		Note:            value.Note,
		TransactionDate: value.TransactionDate,
		SyncStatus:      "synced",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	return tx, s.store.CreateTransaction(tx)
}

func (s TransactionService) Update(db *domain.DB, userID string, current domain.Transaction, body map[string]any) (domain.Transaction, error) {
	value, err := validateTransaction(db, userID, body, &current)
	if err != nil {
		return domain.Transaction{}, err
	}
	updated := current
	updated.Type = value.Type
	updated.Amount = value.Amount
	updated.WalletID = value.WalletID
	updated.CategoryID = value.CategoryID
	updated.TransactionDate = value.TransactionDate
	updated.Note = value.Note
	updated.SyncStatus = "synced"
	updated.UpdatedAt = platform.NowISO(s.clock)
	return updated, s.store.UpdateTransaction(updated)
}

func (s TransactionService) Delete(userID string, id string) error {
	return s.store.SoftDeleteTransaction(userID, id, platform.NowISO(s.clock))
}

type transactionValue struct {
	Type            string
	Amount          float64
	WalletID        string
	CategoryID      string
	TransactionDate string
	Note            string
}

func validateTransaction(db *domain.DB, userID string, body map[string]any, existing *domain.Transaction) (transactionValue, error) {
	value := transactionValue{
		Type:            valueOr(body, "type", existingTransactionString(existing, "type")),
		Amount:          numberOr(body, "amount", existingTransactionAmount(existing)),
		WalletID:        valueOr(body, "walletId", existingTransactionString(existing, "walletId")),
		CategoryID:      valueOr(body, "categoryId", existingTransactionString(existing, "categoryId")),
		TransactionDate: valueOr(body, "transactionDate", existingTransactionString(existing, "transactionDate")),
		Note:            strings.TrimSpace(valueOr(body, "note", existingTransactionString(existing, "note"))),
	}
	if value.Type != "income" && value.Type != "expense" {
		return transactionValue{}, domain.ErrInvalidInput
	}
	if !isFinitePositive(value.Amount) {
		return transactionValue{}, domain.ErrInvalidInput
	}
	if !isDateOnly(value.TransactionDate) {
		return transactionValue{}, domain.ErrInvalidInput
	}
	if !walletExists(db, userID, value.WalletID) {
		return transactionValue{}, domain.ErrInvalidInput
	}
	if !categoryValid(db, userID, value.CategoryID, value.Type) {
		return transactionValue{}, domain.ErrInvalidInput
	}
	return value, nil
}
