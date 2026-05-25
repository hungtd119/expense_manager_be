package sqlitestore

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

func (s *SQLiteStore) FindUserByEmail(email string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return User{}, err
	}
	defer db.Close()

	return findSQLiteUserByEmail(db, email)
}

func (s *SQLiteStore) CreateUserWithWalletAndSession(user User, wallet Wallet, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := insertSQLiteUser(tx, user); err != nil {
		_ = tx.Rollback()
		if isUniqueConstraint(err) {
			return errAlreadyExists
		}
		return err
	}
	if err := insertSQLiteWallet(tx, wallet); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := insertSQLiteSession(tx, session); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) CreateSession(session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	return insertSQLiteSession(db, session)
}

func (s *SQLiteStore) DeleteSession(tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM sessions WHERE token_hash = ?", tokenHash)
	return err
}

func (s *SQLiteStore) DeleteExpiredSessions(now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return false, err
	}
	defer db.Close()

	result, err := db.Exec("DELETE FROM sessions WHERE expires_at < ?", now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func findSQLiteUserByEmail(db *sql.DB, email string) (User, error) {
	var user User
	err := db.QueryRow(`
SELECT id, email, name, password_hash, password_salt, created_at, updated_at
FROM users
WHERE email = ?`, email).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.PasswordSalt, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, errNotFound
	}
	return user, err
}

func insertSQLiteUser(db execer, user User) error {
	updatedAt := user.UpdatedAt
	if updatedAt == "" {
		updatedAt = user.CreatedAt
	}
	_, err := db.Exec(`
INSERT INTO users (id, email, name, password_hash, password_salt, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`, user.ID, user.Email, user.Name, user.PasswordHash, user.PasswordSalt, user.CreatedAt, updatedAt)
	return err
}

func insertSQLiteWallet(db execer, wallet Wallet) error {
	_, err := db.Exec(`
INSERT INTO wallets (id, user_id, name, currency, balance_initial, created_at)
VALUES (?, ?, ?, ?, ?, ?)`, wallet.ID, wallet.UserID, wallet.Name, wallet.Currency, wallet.BalanceInitial, wallet.CreatedAt)
	return err
}

func insertSQLiteSession(db execer, session Session) error {
	_, err := db.Exec(`
INSERT INTO sessions (token_hash, user_id, created_at, expires_at)
VALUES (?, ?, ?, ?)`, session.TokenHash, session.UserID, session.CreatedAt, session.ExpiresAt)
	return err
}

func isUniqueConstraint(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "constraint failed") || strings.Contains(err.Error(), "UNIQUE constraint failed"))
}
