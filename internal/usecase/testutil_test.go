package usecase

import (
	"fmt"
	"testing"
	"time"

	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/platform"
	"expense-manager-mvp/internal/store"
	"expense-manager-mvp/internal/store/memstore"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type seqIDs struct {
	n int
}

func (g *seqIDs) UUID() string {
	g.n++
	return fmt.Sprintf("test-id-%d", g.n)
}

func (g *seqIDs) TokenHex(size int) string {
	g.n++
	return fmt.Sprintf("test-token-%d-%d", g.n, size)
}

type testHarness struct {
	store store.Store
	clock platform.Clock
	ids   platform.IDGenerator
}

func newTestHarness(t *testing.T) testHarness {
	t.Helper()
	st := memstore.New()
	if err := st.Ensure(); err != nil {
		t.Fatalf("ensure memstore: %v", err)
	}
	return testHarness{
		store: st,
		clock: fixedClock{now: time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)},
		ids:   &seqIDs{},
	}
}

func (h testHarness) auth() AuthService {
	return AuthService{store: h.store, clock: h.clock, ids: h.ids}
}

func (h testHarness) transactions() TransactionService {
	return TransactionService{store: h.store, clock: h.clock, ids: h.ids}
}

func (h testHarness) budgets() BudgetService {
	return BudgetService{store: h.store, clock: h.clock, ids: h.ids}
}

func (h testHarness) recurring() RecurringService {
	return RecurringService{store: h.store, clock: h.clock, ids: h.ids}
}

func (h testHarness) readDB(t *testing.T) domain.DB {
	t.Helper()
	db, err := h.store.Read()
	if err != nil {
		t.Fatalf("read db: %v", err)
	}
	return db
}

func registerUser(t *testing.T, h testHarness, email string) (domain.User, string) {
	t.Helper()
	result, err := h.auth().Register(RegisterInput{
		Name:     "Test User",
		Email:    email,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if result.Token == "" || result.User.ID == "" {
		t.Fatal("register returned empty token or user")
	}
	return result.User, result.Token
}

func walletForUser(db *domain.DB, userID string) domain.Wallet {
	for _, wallet := range db.Wallets {
		if wallet.UserID == userID {
			return wallet
		}
	}
	panic("wallet not found for user " + userID)
}

func defaultExpenseCategory(db *domain.DB) domain.Category {
	for _, category := range db.Categories {
		if category.Type == "expense" && category.IsDefault {
			return category
		}
	}
	panic("default expense category not found")
}

var may2026 = domain.MonthBounds{StartDate: "2026-05-01", EndDate: "2026-06-01"}
