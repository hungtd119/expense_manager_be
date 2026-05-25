package mysqlstore

import (
	"database/sql"
)

func (s *MySQLStore) ListBudgetsForMonth(userID string, bounds MonthBounds) ([]Budget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
SELECT id, user_id, category_id, amount_limit, period, start_date, end_date, created_at, updated_at, deleted_at
FROM budgets
WHERE user_id = ? AND deleted_at IS NULL AND period = 'monthly' AND start_date = ?
ORDER BY created_at ASC`, userID, bounds.StartDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Budget{}
	for rows.Next() {
		item, err := scanBudget(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MySQLStore) CreateBudget(budget Budget) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := insertMySQLBudget(db, budget); err != nil {
		if isUniqueConstraint(err) {
			return errAlreadyExists
		}
		return err
	}
	return nil
}

func (s *MySQLStore) UpdateBudget(budget Budget) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Exec(`
UPDATE budgets
SET category_id = ?, amount_limit = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		budget.CategoryID,
		budget.AmountLimit,
		budget.UpdatedAt,
		budget.ID,
		budget.UserID,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return errAlreadyExists
		}
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

func (s *MySQLStore) SoftDeleteBudget(userID string, id string, deletedAt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Exec(`
UPDATE budgets
SET deleted_at = ?, updated_at = ?
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

func insertMySQLBudget(db execer, budget Budget) error {
	updatedAt := budget.UpdatedAt
	if updatedAt == "" {
		updatedAt = budget.CreatedAt
	}
	_, err := db.Exec(`
INSERT INTO budgets (
  id, user_id, category_id, amount_limit, period, start_date, end_date, created_at, updated_at, deleted_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		budget.ID,
		budget.UserID,
		budget.CategoryID,
		budget.AmountLimit,
		budget.Period,
		budget.StartDate,
		budget.EndDate,
		budget.CreatedAt,
		updatedAt,
		nullableString(budget.DeletedAt),
	)
	return err
}

func scanBudget(scanner budgetScanner) (Budget, error) {
	var item Budget
	var deletedAt sql.NullString
	if err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.CategoryID,
		&item.AmountLimit,
		&item.Period,
		&item.StartDate,
		&item.EndDate,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
	); err != nil {
		return Budget{}, err
	}
	item.DeletedAt = stringPtrFromNull(deletedAt)
	return item, nil
}

type budgetScanner interface {
	Scan(dest ...any) error
}
