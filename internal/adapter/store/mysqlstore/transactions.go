package mysqlstore

import (
	"database/sql"
	"sort"
)

func (s *MySQLStore) ListTransactionsForMonth(userID string, bounds MonthBounds) ([]Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
SELECT id, user_id, wallet_id, category_id, type, amount, note, transaction_date,
source_recurring_id, recurring_run_at, recurring_run_date, sync_status, created_at, updated_at, deleted_at
FROM transactions
WHERE user_id = ? AND deleted_at IS NULL AND transaction_date >= ? AND transaction_date < ?
ORDER BY transaction_date DESC, created_at DESC`, userID, bounds.StartDate, bounds.EndDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Transaction{}
	for rows.Next() {
		item, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MySQLStore) CreateTransaction(tx Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	return insertMySQLTransaction(db, tx)
}

func (s *MySQLStore) UpdateTransaction(tx Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Exec(`
UPDATE transactions
SET wallet_id = ?, category_id = ?, type = ?, amount = ?, note = ?, transaction_date = ?,
    source_recurring_id = ?, recurring_run_at = ?, recurring_run_date = ?, sync_status = ?,
    updated_at = ?, deleted_at = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		tx.WalletID,
		tx.CategoryID,
		tx.Type,
		tx.Amount,
		tx.Note,
		tx.TransactionDate,
		nullableString(tx.SourceRecurringID),
		nullableString(tx.RecurringRunAt),
		nullableString(tx.RecurringRunDate),
		tx.SyncStatus,
		tx.UpdatedAt,
		nullableString(tx.DeletedAt),
		tx.ID,
		tx.UserID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound
	}
	return nil
}

func (s *MySQLStore) SoftDeleteTransaction(userID string, id string, deletedAt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Exec(`
UPDATE transactions
SET deleted_at = ?, updated_at = ?, sync_status = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL`, deletedAt, deletedAt, "synced", id, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errNotFound
	}
	return nil
}

func insertMySQLTransaction(db execer, tx Transaction) error {
	syncStatus := tx.SyncStatus
	if syncStatus == "" {
		syncStatus = "synced"
	}
	updatedAt := tx.UpdatedAt
	if updatedAt == "" {
		updatedAt = tx.CreatedAt
	}
	_, err := db.Exec(`
INSERT INTO transactions (
  id, user_id, wallet_id, category_id, type, amount, note, transaction_date,
  source_recurring_id, recurring_run_at, recurring_run_date, sync_status,
  created_at, updated_at, deleted_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tx.ID,
		tx.UserID,
		tx.WalletID,
		tx.CategoryID,
		tx.Type,
		tx.Amount,
		tx.Note,
		tx.TransactionDate,
		nullableString(tx.SourceRecurringID),
		nullableString(tx.RecurringRunAt),
		nullableString(tx.RecurringRunDate),
		syncStatus,
		tx.CreatedAt,
		updatedAt,
		nullableString(tx.DeletedAt),
	)
	return err
}

func scanTransaction(scanner transactionScanner) (Transaction, error) {
	var item Transaction
	var sourceRecurringID, recurringRunAt, recurringRunDate, deletedAt sql.NullString
	if err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.WalletID,
		&item.CategoryID,
		&item.Type,
		&item.Amount,
		&item.Note,
		&item.TransactionDate,
		&sourceRecurringID,
		&recurringRunAt,
		&recurringRunDate,
		&item.SyncStatus,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
	); err != nil {
		return Transaction{}, err
	}
	item.SourceRecurringID = stringPtrFromNull(sourceRecurringID)
	item.RecurringRunAt = stringPtrFromNull(recurringRunAt)
	item.RecurringRunDate = stringPtrFromNull(recurringRunDate)
	item.DeletedAt = stringPtrFromNull(deletedAt)
	return item, nil
}

func sortTransactions(items []Transaction) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].TransactionDate == items[j].TransactionDate {
			return items[i].CreatedAt > items[j].CreatedAt
		}
		return items[i].TransactionDate > items[j].TransactionDate
	})
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

type transactionScanner interface {
	Scan(dest ...any) error
}
