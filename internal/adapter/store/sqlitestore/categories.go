package sqlitestore

import (
	"database/sql"
	"errors"
)

func (s *SQLiteStore) CreateCategory(category Category) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := insertSQLiteCategory(db, category); err != nil {
		if isUniqueConstraint(err) {
			return errAlreadyExists
		}
		return err
	}
	return nil
}

func (s *SQLiteStore) UpdateCategory(category Category) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	var existingUserID sql.NullString
	err = db.QueryRow(`SELECT user_id FROM categories WHERE id = ?`, category.ID).Scan(&existingUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if !existingUserID.Valid {
		return errors.New("cannot update system default category")
	}
	if category.UserID == nil || existingUserID.String != *category.UserID {
		return errNotFound
	}

	result, err := db.Exec(`
UPDATE categories
SET name = ?, icon = ?, color = ?
WHERE id = ? AND user_id = ?`,
		category.Name,
		category.Icon,
		category.Color,
		category.ID,
		category.UserID,
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

func (s *SQLiteStore) DeleteCategory(userID string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	var existingUserID sql.NullString
	err = db.QueryRow(`SELECT user_id FROM categories WHERE id = ?`, id).Scan(&existingUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if !existingUserID.Valid {
		return errors.New("cannot delete system default category")
	}
	if existingUserID.String != userID {
		return errNotFound
	}

	var hasTx int
	err = db.QueryRow(`SELECT COUNT(1) FROM transactions WHERE category_id = ? AND deleted_at IS NULL`, id).Scan(&hasTx)
	if err != nil {
		return err
	}
	if hasTx > 0 {
		return errors.New("cannot delete category containing transactions")
	}

	var hasBudget int
	err = db.QueryRow(`SELECT COUNT(1) FROM budgets WHERE category_id = ? AND deleted_at IS NULL`, id).Scan(&hasBudget)
	if err != nil {
		return err
	}
	if hasBudget > 0 {
		return errors.New("cannot delete category containing budgets")
	}

	var hasRecurring int
	err = db.QueryRow(`SELECT COUNT(1) FROM recurring_transactions WHERE category_id = ? AND deleted_at IS NULL`, id).Scan(&hasRecurring)
	if err != nil {
		return err
	}
	if hasRecurring > 0 {
		return errors.New("cannot delete category linked to recurring transactions")
	}

	result, err := db.Exec(`DELETE FROM categories WHERE id = ? AND user_id = ?`, id, userID)
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

func insertSQLiteCategory(db execer, category Category) error {
	_, err := db.Exec(`
INSERT INTO categories (
  id, user_id, name, type, icon, color, is_default
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		category.ID,
		nullableString(category.UserID),
		category.Name,
		category.Type,
		category.Icon,
		category.Color,
		boolInt(category.IsDefault),
	)
	return err
}
