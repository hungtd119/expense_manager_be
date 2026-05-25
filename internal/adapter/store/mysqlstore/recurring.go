package mysqlstore

import (
	"database/sql"
)

func (s *MySQLStore) ListRecurring(userID string) ([]RecurringTransaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
SELECT id, user_id, wallet_id, category_id, type, amount, note, frequency,
next_run_at, next_run_date, active, created_at, updated_at, deleted_at
FROM recurring_transactions
WHERE user_id = ? AND deleted_at IS NULL
ORDER BY active DESC, next_run_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []RecurringTransaction{}
	for rows.Next() {
		item, err := scanRecurring(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MySQLStore) CreateRecurring(recurring RecurringTransaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	return insertMySQLRecurringTransaction(db, recurring)
}

func (s *MySQLStore) UpdateRecurring(recurring RecurringTransaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Exec(`
UPDATE recurring_transactions
SET wallet_id = ?, category_id = ?, type = ?, amount = ?, note = ?, frequency = ?,
    next_run_at = ?, next_run_date = ?, active = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		recurring.WalletID,
		recurring.CategoryID,
		recurring.Type,
		recurring.Amount,
		recurring.Note,
		recurring.Frequency,
		recurring.NextRunAt,
		recurring.NextRunDate,
		boolInt(recurring.Active),
		recurring.UpdatedAt,
		recurring.ID,
		recurring.UserID,
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

func (s *MySQLStore) SoftDeleteRecurring(userID string, id string, deletedAt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Exec(`
UPDATE recurring_transactions
SET active = 0, deleted_at = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL`, deletedAt, deletedAt, id, userID)
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

func (s *MySQLStore) ProcessDueRecurring(userID string, untilAt string) (RecurringResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return RecurringResult{}, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return RecurringResult{}, err
	}
	result, err := processDueRecurringSQL(tx, userID, untilAt)
	if err != nil {
		_ = tx.Rollback()
		return RecurringResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return RecurringResult{}, err
	}
	return result, nil
}

func processDueRecurringSQL(tx *sql.Tx, userID string, untilAt string) (RecurringResult, error) {
	rows, err := tx.Query(`
SELECT id, user_id, wallet_id, category_id, type, amount, note, frequency,
next_run_at, next_run_date, active, created_at, updated_at, deleted_at
FROM recurring_transactions
WHERE user_id = ? AND active = 1 AND deleted_at IS NULL AND next_run_at <= ?
ORDER BY next_run_at ASC`, userID, untilAt)
	if err != nil {
		return RecurringResult{}, err
	}
	defer rows.Close()

	recurringItems := []RecurringTransaction{}
	for rows.Next() {
		item, err := scanRecurring(rows)
		if err != nil {
			return RecurringResult{}, err
		}
		recurringItems = append(recurringItems, item)
	}
	if err := rows.Err(); err != nil {
		return RecurringResult{}, err
	}

	result := RecurringResult{}
	now := nowISO()
	for _, recurring := range recurringItems {
		runAt := recurring.NextRunAt
		guard := 0
		for runAt != "" && runAt <= untilAt && guard < 36 {
			runDate := runAt[:10]
			exists, err := recurringTransactionExists(tx, userID, recurring.ID, runAt)
			if err != nil {
				return RecurringResult{}, err
			}
			if !exists {
				sourceID := recurring.ID
				runAtCopy := runAt
				runDateCopy := runDate
				generated := Transaction{
					ID:                uuid(),
					UserID:            userID,
					WalletID:          recurring.WalletID,
					CategoryID:        recurring.CategoryID,
					Type:              recurring.Type,
					Amount:            recurring.Amount,
					Note:              recurring.Note,
					TransactionDate:   runDate,
					SourceRecurringID: &sourceID,
					RecurringRunAt:    &runAtCopy,
					RecurringRunDate:  &runDateCopy,
					SyncStatus:        "synced",
					CreatedAt:         now,
					UpdatedAt:         now,
				}
				if err := insertMySQLTransaction(tx, generated); err != nil {
					if !isUniqueConstraint(err) {
						return RecurringResult{}, err
					}
				} else {
					result.GeneratedCount++
					result.Changed = true
				}
			}
			runAt = nextRunAt(runAt, recurring.Frequency)
			guard++
		}
		if runAt != "" && runAt != recurring.NextRunAt {
			if _, err := tx.Exec(`
UPDATE recurring_transactions
SET next_run_at = ?, next_run_date = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL`, runAt, runAt[:10], now, recurring.ID, userID); err != nil {
				return RecurringResult{}, err
			}
			result.Changed = true
		}
	}
	return result, nil
}

func recurringTransactionExists(tx *sql.Tx, userID string, recurringID string, runAt string) (bool, error) {
	var count int
	err := tx.QueryRow(`
SELECT COUNT(1)
FROM transactions
WHERE user_id = ? AND source_recurring_id = ? AND recurring_run_at = ? AND deleted_at IS NULL`, userID, recurringID, runAt).Scan(&count)
	return count > 0, err
}

func insertMySQLRecurringTransaction(db execer, recurring RecurringTransaction) error {
	nextRunAt := recurring.NextRunAt
	if nextRunAt == "" && recurring.NextRunDate != "" {
		nextRunAt = recurring.NextRunDate + "T00:00"
	}
	nextRunDate := recurring.NextRunDate
	if nextRunDate == "" && len(nextRunAt) >= 10 {
		nextRunDate = nextRunAt[:10]
	}
	updatedAt := recurring.UpdatedAt
	if updatedAt == "" {
		updatedAt = recurring.CreatedAt
	}
	_, err := db.Exec(`
INSERT INTO recurring_transactions (
  id, user_id, wallet_id, category_id, type, amount, note, frequency,
  next_run_at, next_run_date, active, created_at, updated_at, deleted_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		recurring.ID,
		recurring.UserID,
		recurring.WalletID,
		recurring.CategoryID,
		recurring.Type,
		recurring.Amount,
		recurring.Note,
		recurring.Frequency,
		nextRunAt,
		nextRunDate,
		boolInt(recurring.Active),
		recurring.CreatedAt,
		updatedAt,
		nullableString(recurring.DeletedAt),
	)
	return err
}

func scanRecurring(scanner recurringScanner) (RecurringTransaction, error) {
	var item RecurringTransaction
	var active int
	var deletedAt sql.NullString
	if err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.WalletID,
		&item.CategoryID,
		&item.Type,
		&item.Amount,
		&item.Note,
		&item.Frequency,
		&item.NextRunAt,
		&item.NextRunDate,
		&active,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
	); err != nil {
		return RecurringTransaction{}, err
	}
	item.Active = active != 0
	item.DeletedAt = stringPtrFromNull(deletedAt)
	return item, nil
}

type recurringScanner interface {
	Scan(dest ...any) error
}
