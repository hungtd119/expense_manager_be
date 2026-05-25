package usecase

import (
	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/platform"
	"expense-manager-mvp/internal/store"
)

type BudgetService struct {
	store store.Store
	clock platform.Clock
	ids   platform.IDGenerator
}

func NewBudgetService(store store.Store) BudgetService {
	return BudgetService{store: store, clock: platform.SystemClock{}, ids: platform.CryptoIDGenerator{}}
}

func (s BudgetService) List(userID string, bounds domain.MonthBounds) ([]domain.Budget, []domain.Transaction, error) {
	if _, err := s.store.ProcessDueRecurring(userID, localDateTimeNow()); err != nil {
		return nil, nil, err
	}
	transactions, err := s.store.ListTransactionsForMonth(userID, bounds)
	if err != nil {
		return nil, nil, err
	}
	budgets, err := s.store.ListBudgetsForMonth(userID, bounds)
	if err != nil {
		return nil, nil, err
	}
	return budgets, transactions, nil
}

func (s BudgetService) Create(db *domain.DB, userID string, bounds domain.MonthBounds, body map[string]any) (domain.Budget, []domain.Transaction, error) {
	value, err := validateBudget(db, userID, body, bounds, nil)
	if err != nil {
		return domain.Budget{}, nil, err
	}
	now := platform.NowISO(s.clock)
	budget := domain.Budget{ID: s.ids.UUID(), UserID: userID, CategoryID: value.CategoryID, AmountLimit: value.AmountLimit, Period: "monthly", StartDate: bounds.StartDate, EndDate: bounds.EndDate, CreatedAt: now, UpdatedAt: now}
	if err := s.store.CreateBudget(budget); err != nil {
		return domain.Budget{}, nil, err
	}
	transactions, err := s.store.ListTransactionsForMonth(userID, bounds)
	return budget, transactions, err
}

func (s BudgetService) Update(db *domain.DB, userID string, current domain.Budget, body map[string]any) (domain.Budget, []domain.Transaction, error) {
	bounds := domain.MonthBounds{StartDate: current.StartDate, EndDate: current.EndDate}
	value, err := validateBudget(db, userID, body, bounds, &current)
	if err != nil {
		return domain.Budget{}, nil, err
	}
	updated := current
	updated.CategoryID = value.CategoryID
	updated.AmountLimit = value.AmountLimit
	updated.UpdatedAt = platform.NowISO(s.clock)
	if err := s.store.UpdateBudget(updated); err != nil {
		return domain.Budget{}, nil, err
	}
	transactions, err := s.store.ListTransactionsForMonth(userID, bounds)
	return updated, transactions, err
}

func (s BudgetService) Delete(userID string, id string) error {
	return s.store.SoftDeleteBudget(userID, id, platform.NowISO(s.clock))
}

type budgetValue struct {
	CategoryID  string
	AmountLimit float64
}

func validateBudget(db *domain.DB, userID string, body map[string]any, bounds domain.MonthBounds, existing *domain.Budget) (budgetValue, error) {
	categoryID := valueOr(body, "categoryId", existingBudgetString(existing, "categoryId"))
	amountLimit := numberOr(body, "amountLimit", existingBudgetAmount(existing))
	if !categoryValid(db, userID, categoryID, "expense") {
		return budgetValue{}, domain.ErrInvalidInput
	}
	if !isFinitePositive(amountLimit) {
		return budgetValue{}, domain.ErrInvalidInput
	}
	return budgetValue{CategoryID: categoryID, AmountLimit: amountLimit}, nil
}

func existingBudgetString(budget *domain.Budget, field string) string {
	if budget == nil {
		return ""
	}
	if field == "categoryId" {
		return budget.CategoryID
	}
	return ""
}

func existingBudgetAmount(budget *domain.Budget) float64 {
	if budget == nil {
		return 0
	}
	return budget.AmountLimit
}
