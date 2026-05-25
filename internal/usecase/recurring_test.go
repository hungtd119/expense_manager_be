package usecase

import (
	"testing"
	"time"
)

func TestRecurringProcessDueGeneratesTransactionOnce(t *testing.T) {
	h := newTestHarness(t)
	user, _ := registerUser(t, h, "recurring@example.com")
	db := h.readDB(t)
	wallet := walletForUser(&db, user.ID)
	category := defaultExpenseCategory(&db)
	svc := h.recurring()

	pastRun := time.Now().Add(-1 * time.Minute).Format("2006-01-02T15:04")
	recurring, result, err := svc.Create(&db, user.ID, map[string]any{
		"type":       "expense",
		"amount":     50000,
		"walletId":   wallet.ID,
		"categoryId": category.ID,
		"frequency":  "daily",
		"nextRunAt":  pastRun,
		"note":       "Tien dien",
	})
	if err != nil {
		t.Fatalf("create recurring: %v", err)
	}
	if result.GeneratedCount < 1 {
		t.Fatalf("expected generated transactions on create, got %+v", result)
	}

	countLinked := func() int {
		t.Helper()
		transactions, err := h.transactions().List(user.ID, may2026)
		if err != nil {
			t.Fatalf("list transactions: %v", err)
		}
		n := 0
		for _, tx := range transactions {
			if tx.SourceRecurringID != nil && *tx.SourceRecurringID == recurring.ID {
				n++
			}
		}
		return n
	}

	afterCreate := countLinked()
	if afterCreate < 1 {
		t.Fatal("expected at least one generated transaction")
	}

	_, _, err = svc.List(user.ID)
	if err != nil {
		t.Fatalf("list recurring: %v", err)
	}
	if countLinked() != afterCreate {
		t.Fatalf("expected no duplicate generation for same runAt, count changed %d -> %d", afterCreate, countLinked())
	}

	items, _, err := svc.List(user.ID)
	if err != nil {
		t.Fatalf("list recurring again: %v", err)
	}
	if len(items) != 1 || items[0].ID != recurring.ID {
		t.Fatalf("unexpected recurring list: %+v", items)
	}
	if items[0].NextRunAt <= pastRun {
		t.Fatalf("expected nextRunAt advanced past %q, got %q", pastRun, items[0].NextRunAt)
	}
}

func TestRecurringDelete(t *testing.T) {
	h := newTestHarness(t)
	user, _ := registerUser(t, h, "recurring-delete@example.com")
	db := h.readDB(t)
	wallet := walletForUser(&db, user.ID)
	category := defaultExpenseCategory(&db)

	recurring, _, err := h.recurring().Create(&db, user.ID, map[string]any{
		"type":       "expense",
		"amount":     30000,
		"walletId":   wallet.ID,
		"categoryId": category.ID,
		"frequency":  "weekly",
		"nextRunAt":  time.Now().Add(48 * time.Hour).Format("2006-01-02T15:04"),
	})
	if err != nil {
		t.Fatalf("create recurring: %v", err)
	}
	if err := h.recurring().Delete(user.ID, recurring.ID); err != nil {
		t.Fatalf("delete recurring: %v", err)
	}
	items, _, err := h.recurring().List(user.ID)
	if err != nil {
		t.Fatalf("list recurring: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no active recurring after delete, got %+v", items)
	}
}
