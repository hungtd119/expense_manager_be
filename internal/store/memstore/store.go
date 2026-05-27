package memstore

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"expense-manager-mvp/internal/adapter/store/shared"
	"expense-manager-mvp/internal/domain"
)

// Store la in-memory implementation cua store.Store cho unit test usecase.
type Store struct {
	mu sync.Mutex
	db domain.DB
}

func New() *Store {
	return &Store{db: shared.EmptyDB()}
}

func NewWithDB(db domain.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ensure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	shared.MigrateShape(&s.db)
	return nil
}

func (s *Store) Read() (domain.DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneDB(s.db), nil
}

func (s *Store) Write(db domain.DB) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db = cloneDB(db)
	return nil
}

func (s *Store) Driver() string  { return "memory" }
func (s *Store) Location() string { return "memory" }

func (s *Store) ListTransactionsForMonth(userID string, bounds domain.MonthBounds) ([]domain.Transaction, error) {
	db, err := s.Read()
	if err != nil {
		return nil, err
	}
	return shared.TransactionsForMonth(&db, userID, bounds), nil
}

func (s *Store) CreateTransaction(tx domain.Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Transactions = append(s.db.Transactions, tx)
	return nil
}

func (s *Store) UpdateTransaction(tx domain.Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.db.Transactions {
		if item.ID == tx.ID && item.UserID == tx.UserID && item.DeletedAt == nil {
			s.db.Transactions[i] = tx
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *Store) SoftDeleteTransaction(userID string, id string, deletedAt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.db.Transactions {
		if item.ID == id && item.UserID == userID && item.DeletedAt == nil {
			s.db.Transactions[i].DeletedAt = &deletedAt
			s.db.Transactions[i].UpdatedAt = deletedAt
			s.db.Transactions[i].SyncStatus = "synced"
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *Store) ListBudgetsForMonth(userID string, bounds domain.MonthBounds) ([]domain.Budget, error) {
	db, err := s.Read()
	if err != nil {
		return nil, err
	}
	return shared.BudgetsForMonth(&db, userID, bounds), nil
}

func (s *Store) CreateBudget(budget domain.Budget) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.db.Budgets {
		if item.UserID == budget.UserID && item.DeletedAt == nil && item.Period == "monthly" && item.StartDate == budget.StartDate && item.CategoryID == budget.CategoryID {
			return domain.ErrAlreadyExists
		}
	}
	s.db.Budgets = append(s.db.Budgets, budget)
	return nil
}

func (s *Store) UpdateBudget(budget domain.Budget) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.db.Budgets {
		if item.ID != budget.ID && item.UserID == budget.UserID && item.DeletedAt == nil && item.Period == "monthly" && item.StartDate == budget.StartDate && item.CategoryID == budget.CategoryID {
			return domain.ErrAlreadyExists
		}
	}
	for i, item := range s.db.Budgets {
		if item.ID == budget.ID && item.UserID == budget.UserID && item.DeletedAt == nil {
			s.db.Budgets[i] = budget
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *Store) SoftDeleteBudget(userID string, id string, deletedAt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.db.Budgets {
		if item.ID == id && item.UserID == userID && item.DeletedAt == nil {
			s.db.Budgets[i].DeletedAt = &deletedAt
			s.db.Budgets[i].UpdatedAt = deletedAt
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *Store) ListRecurring(userID string) ([]domain.RecurringTransaction, error) {
	db, err := s.Read()
	if err != nil {
		return nil, err
	}
	return shared.RecurringForUser(&db, userID), nil
}

func (s *Store) CreateRecurring(recurring domain.RecurringTransaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.RecurringTransactions = append(s.db.RecurringTransactions, recurring)
	return nil
}

func (s *Store) UpdateRecurring(recurring domain.RecurringTransaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.db.RecurringTransactions {
		if item.ID == recurring.ID && item.UserID == recurring.UserID && item.DeletedAt == nil {
			s.db.RecurringTransactions[i] = recurring
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *Store) SoftDeleteRecurring(userID string, id string, deletedAt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.db.RecurringTransactions {
		if item.ID == id && item.UserID == userID && item.DeletedAt == nil {
			s.db.RecurringTransactions[i].Active = false
			s.db.RecurringTransactions[i].DeletedAt = &deletedAt
			s.db.RecurringTransactions[i].UpdatedAt = deletedAt
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *Store) ProcessDueRecurring(userID string, untilAt string) (domain.RecurringResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := shared.ProcessDueRecurring(&s.db, userID, untilAt)
	return result, nil
}

func (s *Store) FindUserByEmail(email string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, user := range s.db.Users {
		if user.Email == email {
			return user, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (s *Store) CreateUserWithWalletAndSession(user domain.User, wallet domain.Wallet, session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.db.Users {
		if item.Email == user.Email {
			return domain.ErrAlreadyExists
		}
	}
	s.db.Users = append(s.db.Users, user)
	s.db.Wallets = append(s.db.Wallets, wallet)
	s.db.Sessions = append(s.db.Sessions, session)
	return nil
}

func (s *Store) CreateSession(session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Sessions = append(s.db.Sessions, session)
	return nil
}

func (s *Store) DeleteSession(tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.db.Sessions[:0]
	changed := false
	for _, session := range s.db.Sessions {
		if session.TokenHash == tokenHash {
			changed = true
			continue
		}
		next = append(next, session)
	}
	if !changed {
		return nil
	}
	s.db.Sessions = next
	return nil
}

func (s *Store) DeleteExpiredSessions(now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.db.Sessions[:0]
	changed := false
	for _, session := range s.db.Sessions {
		expiresAt, err := shared.ParseSessionTime(session.ExpiresAt)
		if err != nil || expiresAt.Before(now) {
			changed = true
			continue
		}
		next = append(next, session)
	}
	if !changed {
		return false, nil
	}
	s.db.Sessions = next
	return true, nil
}

func cloneDB(db domain.DB) domain.DB {
	bytes, err := json.Marshal(db)
	if err != nil {
		panic(err)
	}
	var out domain.DB
	if err := json.Unmarshal(bytes, &out); err != nil {
		panic(err)
	}
	return out
}

func (s *Store) CreateWallet(wallet domain.Wallet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Wallets = append(s.db.Wallets, wallet)
	return nil
}

func (s *Store) UpdateWallet(wallet domain.Wallet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.db.Wallets {
		if item.ID == wallet.ID && item.UserID == wallet.UserID {
			s.db.Wallets[i] = wallet
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *Store) DeleteWallet(userID string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, tx := range s.db.Transactions {
		if tx.WalletID == id && tx.DeletedAt == nil {
			return errors.New("cannot delete wallet containing transactions")
		}
	}
	for _, rt := range s.db.RecurringTransactions {
		if rt.WalletID == id && rt.DeletedAt == nil {
			return errors.New("cannot delete wallet linked to recurring transactions")
		}
	}
	next := s.db.Wallets[:0]
	found := false
	for _, item := range s.db.Wallets {
		if item.ID == id && item.UserID == userID {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return domain.ErrNotFound
	}
	s.db.Wallets = next
	return nil
}

func (s *Store) CreateCategory(category domain.Category) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Categories = append(s.db.Categories, category)
	return nil
}

func (s *Store) UpdateCategory(category domain.Category) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.db.Categories {
		if item.ID == category.ID && item.UserID != nil && *item.UserID == *category.UserID {
			s.db.Categories[i] = category
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *Store) DeleteCategory(userID string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.db.Categories {
		if item.ID == id {
			if item.UserID == nil {
				return errors.New("cannot delete system default category")
			}
			if *item.UserID != userID {
				return domain.ErrNotFound
			}
		}
	}
	for _, tx := range s.db.Transactions {
		if tx.CategoryID == id && tx.DeletedAt == nil {
			return errors.New("cannot delete category containing transactions")
		}
	}
	for _, b := range s.db.Budgets {
		if b.CategoryID == id && b.DeletedAt == nil {
			return errors.New("cannot delete category containing budgets")
		}
	}
	for _, rt := range s.db.RecurringTransactions {
		if rt.CategoryID == id && rt.DeletedAt == nil {
			return errors.New("cannot delete category linked to recurring transactions")
		}
	}
	next := s.db.Categories[:0]
	found := false
	for _, item := range s.db.Categories {
		if item.ID == id && item.UserID != nil && *item.UserID == userID {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return domain.ErrNotFound
	}
	s.db.Categories = next
	return nil
}

