package usecase

import (
	"strings"

	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/platform"
	"expense-manager-mvp/internal/store"
)

type RecurringService struct {
	store store.Store
	clock platform.Clock
	ids   platform.IDGenerator
}

func NewRecurringService(store store.Store) RecurringService {
	return RecurringService{store: store, clock: platform.SystemClock{}, ids: platform.CryptoIDGenerator{}}
}

func (s RecurringService) List(userID string) ([]domain.RecurringTransaction, domain.RecurringResult, error) {
	result, err := s.store.ProcessDueRecurring(userID, localDateTimeNow())
	if err != nil {
		return nil, domain.RecurringResult{}, err
	}
	items, err := s.store.ListRecurring(userID)
	return items, result, err
}

func (s RecurringService) Create(db *domain.DB, userID string, body map[string]any) (domain.RecurringTransaction, domain.RecurringResult, error) {
	value, err := validateRecurring(db, userID, body, nil)
	if err != nil {
		return domain.RecurringTransaction{}, domain.RecurringResult{}, err
	}
	now := platform.NowISO(s.clock)
	recurring := domain.RecurringTransaction{
		ID:          s.ids.UUID(),
		UserID:      userID,
		WalletID:    value.WalletID,
		CategoryID:  value.CategoryID,
		Type:        value.Type,
		Amount:      value.Amount,
		Note:        value.Note,
		Frequency:   value.Frequency,
		NextRunAt:   value.NextRunAt,
		NextRunDate: value.NextRunDate,
		Active:      value.Active,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.CreateRecurring(recurring); err != nil {
		return domain.RecurringTransaction{}, domain.RecurringResult{}, err
	}
	result, err := s.store.ProcessDueRecurring(userID, localDateTimeNow())
	return recurring, result, err
}

func (s RecurringService) Update(db *domain.DB, userID string, current domain.RecurringTransaction, body map[string]any) (domain.RecurringTransaction, domain.RecurringResult, error) {
	value, err := validateRecurring(db, userID, body, &current)
	if err != nil {
		return domain.RecurringTransaction{}, domain.RecurringResult{}, err
	}
	updated := current
	updated.Type = value.Type
	updated.Amount = value.Amount
	updated.WalletID = value.WalletID
	updated.CategoryID = value.CategoryID
	updated.Note = value.Note
	updated.Frequency = value.Frequency
	updated.NextRunAt = value.NextRunAt
	updated.NextRunDate = value.NextRunDate
	updated.Active = value.Active
	updated.UpdatedAt = platform.NowISO(s.clock)
	if err := s.store.UpdateRecurring(updated); err != nil {
		return domain.RecurringTransaction{}, domain.RecurringResult{}, err
	}
	result, err := s.store.ProcessDueRecurring(userID, localDateTimeNow())
	return updated, result, err
}

func (s RecurringService) Delete(userID string, id string) error {
	return s.store.SoftDeleteRecurring(userID, id, platform.NowISO(s.clock))
}

type recurringValue struct {
	Type        string
	Amount      float64
	WalletID    string
	CategoryID  string
	Note        string
	Frequency   string
	NextRunAt   string
	NextRunDate string
	Active      bool
}

func validateRecurring(db *domain.DB, userID string, body map[string]any, existing *domain.RecurringTransaction) (recurringValue, error) {
	value := recurringValue{
		Type:       valueOr(body, "type", existingRecurringString(existing, "type")),
		Amount:     numberOr(body, "amount", existingRecurringAmount(existing)),
		WalletID:   valueOr(body, "walletId", existingRecurringString(existing, "walletId")),
		CategoryID: valueOr(body, "categoryId", existingRecurringString(existing, "categoryId")),
		Note:       strings.TrimSpace(valueOr(body, "note", existingRecurringString(existing, "note"))),
		Frequency:  valueOr(body, "frequency", existingRecurringString(existing, "frequency")),
		NextRunAt:  valueOr(body, "nextRunAt", existingRecurringString(existing, "nextRunAt")),
		Active:     true,
	}
	if value.NextRunAt == "" {
		value.NextRunAt = valueOr(body, "nextRunDate", existingRecurringString(existing, "nextRunDate"))
	}
	if existing != nil {
		value.Active = existing.Active
	}
	if raw, ok := body["active"].(bool); ok {
		value.Active = raw
	}
	if value.Type != "income" && value.Type != "expense" {
		return recurringValue{}, domain.ErrInvalidInput
	}
	if !isFinitePositive(value.Amount) {
		return recurringValue{}, domain.ErrInvalidInput
	}
	if value.Frequency != "daily" && value.Frequency != "weekly" && value.Frequency != "monthly" {
		return recurringValue{}, domain.ErrInvalidInput
	}
	if !isDateTimeLocal(value.NextRunAt) {
		return recurringValue{}, domain.ErrInvalidInput
	}
	if !walletExists(db, userID, value.WalletID) {
		return recurringValue{}, domain.ErrInvalidInput
	}
	if !categoryValid(db, userID, value.CategoryID, value.Type) {
		return recurringValue{}, domain.ErrInvalidInput
	}
	value.NextRunDate = value.NextRunAt[:10]
	return value, nil
}

func existingRecurringString(recurring *domain.RecurringTransaction, field string) string {
	if recurring == nil {
		return ""
	}
	switch field {
	case "type":
		return recurring.Type
	case "walletId":
		return recurring.WalletID
	case "categoryId":
		return recurring.CategoryID
	case "note":
		return recurring.Note
	case "frequency":
		return recurring.Frequency
	case "nextRunAt":
		return recurring.NextRunAt
	case "nextRunDate":
		return recurring.NextRunDate
	default:
		return ""
	}
}

func existingRecurringAmount(recurring *domain.RecurringTransaction) float64 {
	if recurring == nil {
		return 0
	}
	return recurring.Amount
}
