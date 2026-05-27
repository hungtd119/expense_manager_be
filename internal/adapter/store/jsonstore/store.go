package jsonstore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"expense-manager-mvp/internal/adapter/store/shared"
	"expense-manager-mvp/internal/domain"
)

type Store struct {
	path string
	mu   sync.Mutex
}

func New(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Ensure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(s.path); errors.Is(err, os.ErrNotExist) {
		return s.writeLocked(shared.EmptyDB())
	}
	return nil
}

func (s *Store) Read() (domain.DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bytes, err := os.ReadFile(s.path)
	if err != nil {
		return domain.DB{}, err
	}
	var db domain.DB
	if err := json.Unmarshal(bytes, &db); err != nil {
		return domain.DB{}, err
	}
	if shared.MigrateShape(&db) {
		if err := s.writeLocked(db); err != nil {
			return domain.DB{}, err
		}
	}
	return db, nil
}

func (s *Store) Write(db domain.DB) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeLocked(db)
}

func (s *Store) Driver() string {
	return "json"
}

func (s *Store) Location() string {
	return s.path
}

func (s *Store) ListTransactionsForMonth(userID string, bounds domain.MonthBounds) ([]domain.Transaction, error) {
	db, err := s.Read()
	if err != nil {
		return nil, err
	}
	return shared.TransactionsForMonth(&db, userID, bounds), nil
}

func (s *Store) CreateTransaction(tx domain.Transaction) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	db.Transactions = append(db.Transactions, tx)
	return s.Write(db)
}

func (s *Store) UpdateTransaction(tx domain.Transaction) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	for i, item := range db.Transactions {
		if item.ID == tx.ID && item.UserID == tx.UserID && item.DeletedAt == nil {
			db.Transactions[i] = tx
			return s.Write(db)
		}
	}
	return domain.ErrNotFound
}

func (s *Store) SoftDeleteTransaction(userID string, id string, deletedAt string) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	for i, item := range db.Transactions {
		if item.ID == id && item.UserID == userID && item.DeletedAt == nil {
			db.Transactions[i].DeletedAt = &deletedAt
			db.Transactions[i].UpdatedAt = deletedAt
			db.Transactions[i].SyncStatus = "synced"
			return s.Write(db)
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
	db, err := s.Read()
	if err != nil {
		return err
	}
	for _, item := range db.Budgets {
		if item.UserID == budget.UserID && item.DeletedAt == nil && item.Period == "monthly" && item.StartDate == budget.StartDate && item.CategoryID == budget.CategoryID {
			return domain.ErrAlreadyExists
		}
	}
	db.Budgets = append(db.Budgets, budget)
	return s.Write(db)
}

func (s *Store) UpdateBudget(budget domain.Budget) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	for _, item := range db.Budgets {
		if item.ID != budget.ID && item.UserID == budget.UserID && item.DeletedAt == nil && item.Period == "monthly" && item.StartDate == budget.StartDate && item.CategoryID == budget.CategoryID {
			return domain.ErrAlreadyExists
		}
	}
	for i, item := range db.Budgets {
		if item.ID == budget.ID && item.UserID == budget.UserID && item.DeletedAt == nil {
			db.Budgets[i] = budget
			return s.Write(db)
		}
	}
	return domain.ErrNotFound
}

func (s *Store) SoftDeleteBudget(userID string, id string, deletedAt string) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	for i, item := range db.Budgets {
		if item.ID == id && item.UserID == userID && item.DeletedAt == nil {
			db.Budgets[i].DeletedAt = &deletedAt
			db.Budgets[i].UpdatedAt = deletedAt
			return s.Write(db)
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
	db, err := s.Read()
	if err != nil {
		return err
	}
	db.RecurringTransactions = append(db.RecurringTransactions, recurring)
	return s.Write(db)
}

func (s *Store) UpdateRecurring(recurring domain.RecurringTransaction) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	for i, item := range db.RecurringTransactions {
		if item.ID == recurring.ID && item.UserID == recurring.UserID && item.DeletedAt == nil {
			db.RecurringTransactions[i] = recurring
			return s.Write(db)
		}
	}
	return domain.ErrNotFound
}

func (s *Store) SoftDeleteRecurring(userID string, id string, deletedAt string) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	for i, item := range db.RecurringTransactions {
		if item.ID == id && item.UserID == userID && item.DeletedAt == nil {
			db.RecurringTransactions[i].Active = false
			db.RecurringTransactions[i].DeletedAt = &deletedAt
			db.RecurringTransactions[i].UpdatedAt = deletedAt
			return s.Write(db)
		}
	}
	return domain.ErrNotFound
}

func (s *Store) ProcessDueRecurring(userID string, untilAt string) (domain.RecurringResult, error) {
	db, err := s.Read()
	if err != nil {
		return domain.RecurringResult{}, err
	}
	result := shared.ProcessDueRecurring(&db, userID, untilAt)
	if result.Changed {
		return result, s.Write(db)
	}
	return result, nil
}

func (s *Store) FindUserByEmail(email string) (domain.User, error) {
	db, err := s.Read()
	if err != nil {
		return domain.User{}, err
	}
	for _, user := range db.Users {
		if user.Email == email {
			return user, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (s *Store) CreateUserWithWalletAndSession(user domain.User, wallet domain.Wallet, session domain.Session) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	for _, item := range db.Users {
		if item.Email == user.Email {
			return domain.ErrAlreadyExists
		}
	}
	db.Users = append(db.Users, user)
	db.Wallets = append(db.Wallets, wallet)
	db.Sessions = append(db.Sessions, session)
	return s.Write(db)
}

func (s *Store) CreateSession(session domain.Session) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	db.Sessions = append(db.Sessions, session)
	return s.Write(db)
}

func (s *Store) DeleteSession(tokenHash string) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	next := db.Sessions[:0]
	changed := false
	for _, session := range db.Sessions {
		if session.TokenHash == tokenHash {
			changed = true
			continue
		}
		next = append(next, session)
	}
	if !changed {
		return nil
	}
	db.Sessions = next
	return s.Write(db)
}

func (s *Store) DeleteExpiredSessions(now time.Time) (bool, error) {
	db, err := s.Read()
	if err != nil {
		return false, err
	}
	next := db.Sessions[:0]
	changed := false
	for _, session := range db.Sessions {
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
	db.Sessions = next
	return true, s.Write(db)
}

func (s *Store) CreateWallet(wallet domain.Wallet) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	db.Wallets = append(db.Wallets, wallet)
	return s.Write(db)
}

func (s *Store) UpdateWallet(wallet domain.Wallet) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	for i, item := range db.Wallets {
		if item.ID == wallet.ID && item.UserID == wallet.UserID {
			db.Wallets[i] = wallet
			return s.Write(db)
		}
	}
	return domain.ErrNotFound
}

func (s *Store) DeleteWallet(userID string, id string) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	for _, tx := range db.Transactions {
		if tx.WalletID == id && tx.DeletedAt == nil {
			return errors.New("cannot delete wallet containing transactions")
		}
	}
	for _, rt := range db.RecurringTransactions {
		if rt.WalletID == id && rt.DeletedAt == nil {
			return errors.New("cannot delete wallet linked to recurring transactions")
		}
	}
	next := db.Wallets[:0]
	found := false
	for _, item := range db.Wallets {
		if item.ID == id && item.UserID == userID {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return domain.ErrNotFound
	}
	db.Wallets = next
	return s.Write(db)
}

func (s *Store) CreateCategory(category domain.Category) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	db.Categories = append(db.Categories, category)
	return s.Write(db)
}

func (s *Store) UpdateCategory(category domain.Category) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	for i, item := range db.Categories {
		if item.ID == category.ID && item.UserID != nil && *item.UserID == *category.UserID {
			db.Categories[i] = category
			return s.Write(db)
		}
	}
	return domain.ErrNotFound
}

func (s *Store) DeleteCategory(userID string, id string) error {
	db, err := s.Read()
	if err != nil {
		return err
	}
	for _, item := range db.Categories {
		if item.ID == id {
			if item.UserID == nil {
				return errors.New("cannot delete system default category")
			}
			if *item.UserID != userID {
				return domain.ErrNotFound
			}
		}
	}
	for _, tx := range db.Transactions {
		if tx.CategoryID == id && tx.DeletedAt == nil {
			return errors.New("cannot delete category containing transactions")
		}
	}
	for _, b := range db.Budgets {
		if b.CategoryID == id && b.DeletedAt == nil {
			return errors.New("cannot delete category containing budgets")
		}
	}
	for _, rt := range db.RecurringTransactions {
		if rt.CategoryID == id && rt.DeletedAt == nil {
			return errors.New("cannot delete category linked to recurring transactions")
		}
	}
	next := db.Categories[:0]
	found := false
	for _, item := range db.Categories {
		if item.ID == id && item.UserID != nil && *item.UserID == userID {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return domain.ErrNotFound
	}
	db.Categories = next
	return s.Write(db)
}

func (s *Store) writeLocked(db domain.DB) error {
	bytes, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, bytes, 0o644)
}
