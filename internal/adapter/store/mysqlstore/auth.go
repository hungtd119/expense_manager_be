package mysqlstore

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

func (s *MySQLStore) FindUserByEmail(email string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return User{}, err
	}
	defer db.Close()

	return findMySQLUserByEmail(db, email)
}

func (s *MySQLStore) CreateUserWithWalletAndSession(user User, wallet Wallet, session Session) error {
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
	if err := insertMySQLUser(tx, user); err != nil {
		_ = tx.Rollback()
		if isUniqueConstraint(err) {
			return errAlreadyExists
		}
		return err
	}
	if err := insertMySQLWallet(tx, wallet); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := insertMySQLSession(tx, session); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *MySQLStore) CreateSession(session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	return insertMySQLSession(db, session)
}

func (s *MySQLStore) DeleteSession(tokenHash string) error {
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

func (s *MySQLStore) DeleteExpiredSessions(now time.Time) (bool, error) {
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

func findMySQLUserByEmail(db *sql.DB, email string) (User, error) {
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

func insertMySQLUser(db execer, user User) error {
	updatedAt := user.UpdatedAt
	if updatedAt == "" {
		updatedAt = user.CreatedAt
	}
	_, err := db.Exec(`
INSERT INTO users (id, email, name, password_hash, password_salt, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`, user.ID, user.Email, user.Name, user.PasswordHash, user.PasswordSalt, user.CreatedAt, updatedAt)
	return err
}

func insertMySQLWallet(db execer, wallet Wallet) error {
	_, err := db.Exec(`
INSERT INTO wallets (id, user_id, name, currency, balance_initial, created_at)
VALUES (?, ?, ?, ?, ?, ?)`, wallet.ID, wallet.UserID, wallet.Name, wallet.Currency, wallet.BalanceInitial, wallet.CreatedAt)
	return err
}

func insertMySQLSession(db execer, session Session) error {
	_, err := db.Exec(`
INSERT INTO sessions (token_hash, user_id, created_at, expires_at)
VALUES (?, ?, ?, ?)`, session.TokenHash, session.UserID, session.CreatedAt, session.ExpiresAt)
	return err
}

func isUniqueConstraint(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Error 1062") || strings.Contains(err.Error(), "Duplicate entry"))
}
