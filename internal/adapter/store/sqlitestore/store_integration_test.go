package sqlitestore

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStoreRepositories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.sqlite")
	store := NewSQLiteStore(path, "")
	if err := store.Ensure(); err != nil {
		t.Fatalf("ensure sqlite store: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	user := User{ID: uuid(), Email: "repo@example.com", Name: "Repo Tester", PasswordHash: "hash", PasswordSalt: "salt", CreatedAt: now, UpdatedAt: now}
	wallet := Wallet{ID: uuid(), UserID: user.ID, Name: "Vi chinh", Currency: "VND", CreatedAt: now}
	session := Session{TokenHash: "token-hash", UserID: user.ID, CreatedAt: now, ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	if err := store.CreateUserWithWalletAndSession(user, wallet, session); err != nil {
		t.Fatalf("create user/wallet/session: %v", err)
	}
	found, err := store.FindUserByEmail(user.Email)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if found.ID != user.ID {
		t.Fatalf("found wrong user: %s", found.ID)
	}

	db, err := store.Read()
	if err != nil {
		t.Fatalf("read db: %v", err)
	}
	var expenseCategory Category
	for _, category := range db.Categories {
		if category.Type == "expense" {
			expenseCategory = category
			break
		}
	}
	if expenseCategory.ID == "" {
		t.Fatal("missing default expense category")
	}

	tx := Transaction{ID: uuid(), UserID: user.ID, WalletID: wallet.ID, CategoryID: expenseCategory.ID, Type: "expense", Amount: 120000, Note: "Repo tx", TransactionDate: "2026-05-22", SyncStatus: "synced", CreatedAt: now, UpdatedAt: now}
	if err := store.CreateTransaction(tx); err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	transactions, err := store.ListTransactionsForMonth(user.ID, MonthBounds{StartDate: "2026-05-01", EndDate: "2026-06-01"})
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(transactions))
	}

	budget := Budget{ID: uuid(), UserID: user.ID, CategoryID: expenseCategory.ID, AmountLimit: 500000, Period: "monthly", StartDate: "2026-05-01", EndDate: "2026-06-01", CreatedAt: now, UpdatedAt: now}
	if err := store.CreateBudget(budget); err != nil {
		t.Fatalf("create budget: %v", err)
	}
	budgets, err := store.ListBudgetsForMonth(user.ID, MonthBounds{StartDate: "2026-05-01", EndDate: "2026-06-01"})
	if err != nil {
		t.Fatalf("list budgets: %v", err)
	}
	if len(budgets) != 1 {
		t.Fatalf("expected 1 budget, got %d", len(budgets))
	}

	recurring := RecurringTransaction{ID: uuid(), UserID: user.ID, WalletID: wallet.ID, CategoryID: expenseCategory.ID, Type: "expense", Amount: 45000, Note: "Repo recurring", Frequency: "daily", NextRunAt: "2026-05-22T10:00", NextRunDate: "2026-05-22", Active: true, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateRecurring(recurring); err != nil {
		t.Fatalf("create recurring: %v", err)
	}
	result, err := store.ProcessDueRecurring(user.ID, "2026-05-22T10:00")
	if err != nil {
		t.Fatalf("process due recurring: %v", err)
	}
	if result.GeneratedCount != 1 {
		t.Fatalf("expected 1 generated transaction, got %d", result.GeneratedCount)
	}
	transactions, err = store.ListTransactionsForMonth(user.ID, MonthBounds{StartDate: "2026-05-01", EndDate: "2026-06-01"})
	if err != nil {
		t.Fatalf("list transactions after recurring: %v", err)
	}
	if len(transactions) != 2 {
		t.Fatalf("expected 2 transactions after recurring, got %d", len(transactions))
	}
}
