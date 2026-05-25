package usecase

import (
	"errors"
	"testing"

	"expense-manager-mvp/internal/domain"
)

func TestBudgetCRUD(t *testing.T) {
	h := newTestHarness(t)
	user, _ := registerUser(t, h, "budget@example.com")
	db := h.readDB(t)
	category := defaultExpenseCategory(&db)
	svc := h.budgets()

	budget, _, err := svc.Create(&db, user.ID, may2026, map[string]any{
		"categoryId":  category.ID,
		"amountLimit": 500000,
	})
	if err != nil {
		t.Fatalf("create budget: %v", err)
	}
	if budget.AmountLimit != 500000 {
		t.Fatalf("unexpected budget: %+v", budget)
	}

	budgets, _, err := svc.List(user.ID, may2026)
	if err != nil {
		t.Fatalf("list budgets: %v", err)
	}
	if len(budgets) != 1 || budgets[0].ID != budget.ID {
		t.Fatalf("expected one budget, got %+v", budgets)
	}

	_, _, err = svc.Create(&db, user.ID, may2026, map[string]any{
		"categoryId":  category.ID,
		"amountLimit": 600000,
	})
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists for duplicate budget, got %v", err)
	}

	updated, _, err := svc.Update(&db, user.ID, budget, map[string]any{"amountLimit": 700000})
	if err != nil {
		t.Fatalf("update budget: %v", err)
	}
	if updated.AmountLimit != 700000 {
		t.Fatalf("unexpected updated budget: %+v", updated)
	}

	if err := svc.Delete(user.ID, budget.ID); err != nil {
		t.Fatalf("delete budget: %v", err)
	}
	budgets, _, err = svc.List(user.ID, may2026)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(budgets) != 0 {
		t.Fatalf("expected no budgets after delete, got %d", len(budgets))
	}
}

func TestBudgetCreateValidation(t *testing.T) {
	h := newTestHarness(t)
	user, _ := registerUser(t, h, "budget-invalid@example.com")
	db := h.readDB(t)

	_, _, err := h.budgets().Create(&db, user.ID, may2026, map[string]any{
		"categoryId":  "missing",
		"amountLimit": 100,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
