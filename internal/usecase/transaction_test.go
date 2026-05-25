package usecase

import (
	"errors"
	"testing"

	"expense-manager-mvp/internal/domain"
)

func TestTransactionCRUD(t *testing.T) {
	h := newTestHarness(t)
	user, _ := registerUser(t, h, "tx@example.com")
	db := h.readDB(t)
	wallet := walletForUser(&db, user.ID)
	category := defaultExpenseCategory(&db)

	svc := h.transactions()
	tx, err := svc.Create(&db, user.ID, map[string]any{
		"type":            "expense",
		"amount":          120000,
		"walletId":        wallet.ID,
		"categoryId":      category.ID,
		"transactionDate": "2026-05-10",
		"note":            "An trua",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if tx.Amount != 120000 || tx.Type != "expense" {
		t.Fatalf("unexpected tx: %+v", tx)
	}

	items, err := svc.List(user.ID, may2026)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].ID != tx.ID {
		t.Fatalf("expected one transaction, got %+v", items)
	}

	updated, err := svc.Update(&db, user.ID, tx, map[string]any{"amount": 150000, "note": "Cap nhat"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Amount != 150000 || updated.Note != "Cap nhat" {
		t.Fatalf("unexpected updated tx: %+v", updated)
	}

	if err := svc.Delete(user.ID, tx.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	items, err = svc.List(user.ID, may2026)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no transactions after delete, got %d", len(items))
	}
}

func TestTransactionCreateValidation(t *testing.T) {
	h := newTestHarness(t)
	user, _ := registerUser(t, h, "invalid-tx@example.com")
	db := h.readDB(t)
	wallet := walletForUser(&db, user.ID)

	_, err := h.transactions().Create(&db, user.ID, map[string]any{
		"type":            "expense",
		"amount":          -1,
		"walletId":        wallet.ID,
		"categoryId":      "missing-category",
		"transactionDate": "2026-05-10",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestTransactionDeleteNotFound(t *testing.T) {
	h := newTestHarness(t)
	user, _ := registerUser(t, h, "delete-missing@example.com")
	err := h.transactions().Delete(user.ID, "missing-id")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
