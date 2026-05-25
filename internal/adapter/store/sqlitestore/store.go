package sqlitestore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	path       string
	importPath string
	mu         sync.Mutex
}

func NewSQLiteStore(path string, importPath string) *SQLiteStore {
	return &SQLiteStore{path: path, importPath: importPath}
}

func (s *SQLiteStore) Ensure() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := createSQLiteSchema(db); err != nil {
		return err
	}
	return s.migrateInitialState(db)
}

func (s *SQLiteStore) Read() (DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dbConn, err := s.open()
	if err != nil {
		return DB{}, err
	}
	defer dbConn.Close()

	db, err := readSQLiteTables(dbConn)
	if err != nil {
		return DB{}, err
	}
	if migrateShape(&db) {
		if err := writeSQLiteTables(dbConn, db); err != nil {
			return DB{}, err
		}
	}
	return db, nil
}

func (s *SQLiteStore) Write(db DB) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dbConn, err := s.open()
	if err != nil {
		return err
	}
	defer dbConn.Close()

	return writeSQLiteTables(dbConn, db)
}

func (s *SQLiteStore) Driver() string {
	return "sqlite"
}

func (s *SQLiteStore) Location() string {
	return s.path
}

func (s *SQLiteStore) open() (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (s *SQLiteStore) migrateInitialState(db *sql.DB) error {
	var exists int
	if err := db.QueryRow("SELECT COUNT(1) FROM schema_migrations WHERE version = 1").Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}

	initialState := emptyDB()
	if s.importPath != "" {
		bytes, err := os.ReadFile(s.importPath)
		if err == nil {
			if err := json.Unmarshal(bytes, &initialState); err != nil {
				return err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	migrateShape(&initialState)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := clearSQLiteEntityTables(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := insertSQLiteState(tx, initialState); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)",
		1,
		"entity_tables",
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func createSQLiteSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  password_salt TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
  token_hash TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS wallets (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  name TEXT NOT NULL,
  currency TEXT NOT NULL,
  balance_initial REAL NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS categories (
  id TEXT PRIMARY KEY,
  user_id TEXT,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  icon TEXT NOT NULL,
  color TEXT NOT NULL,
  is_default INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS recurring_transactions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  wallet_id TEXT NOT NULL,
  category_id TEXT NOT NULL,
  type TEXT NOT NULL,
  amount REAL NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  frequency TEXT NOT NULL,
  next_run_at TEXT NOT NULL,
  next_run_date TEXT NOT NULL,
  active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE RESTRICT,
  FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS transactions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  wallet_id TEXT NOT NULL,
  category_id TEXT NOT NULL,
  type TEXT NOT NULL,
  amount REAL NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  transaction_date TEXT NOT NULL,
  source_recurring_id TEXT,
  recurring_run_at TEXT,
  recurring_run_date TEXT,
  sync_status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE RESTRICT,
  FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT,
  FOREIGN KEY (source_recurring_id) REFERENCES recurring_transactions(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS budgets (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  category_id TEXT NOT NULL,
  amount_limit REAL NOT NULL,
  period TEXT NOT NULL,
  start_date TEXT NOT NULL,
  end_date TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS notification_rules (
  id TEXT PRIMARY KEY,
  user_id TEXT,
  payload TEXT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_recurring_run
  ON transactions(source_recurring_id, recurring_run_at)
  WHERE source_recurring_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_transactions_user_month
  ON transactions(user_id, transaction_date, deleted_at);
CREATE INDEX IF NOT EXISTS idx_budgets_user_month
  ON budgets(user_id, start_date, deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_budgets_unique_month_category
  ON budgets(user_id, start_date, category_id)
  WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recurring_user_due
  ON recurring_transactions(user_id, active, deleted_at, next_run_at);
CREATE INDEX IF NOT EXISTS idx_sessions_expiry
  ON sessions(expires_at);
`)
	return err
}

func writeSQLiteTables(db *sql.DB, state DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := clearSQLiteEntityTables(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := insertSQLiteState(tx, state); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func clearSQLiteEntityTables(tx *sql.Tx) error {
	statements := []string{
		"DELETE FROM notification_rules",
		"DELETE FROM transactions",
		"DELETE FROM budgets",
		"DELETE FROM recurring_transactions",
		"DELETE FROM wallets",
		"DELETE FROM categories",
		"DELETE FROM sessions",
		"DELETE FROM users",
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteState(tx *sql.Tx, db DB) error {
	if err := insertSQLiteUsers(tx, db.Users); err != nil {
		return err
	}
	if err := insertSQLiteCategories(tx, db.Categories); err != nil {
		return err
	}
	if err := insertSQLiteWallets(tx, db.Wallets); err != nil {
		return err
	}
	if err := insertSQLiteRecurringTransactions(tx, db.RecurringTransactions); err != nil {
		return err
	}
	if err := insertSQLiteTransactions(tx, db.Transactions); err != nil {
		return err
	}
	if err := insertSQLiteBudgets(tx, db.Budgets); err != nil {
		return err
	}
	if err := insertSQLiteSessions(tx, db.Sessions); err != nil {
		return err
	}
	return insertSQLiteNotificationRules(tx, db.NotificationRules)
}

func insertSQLiteUsers(tx *sql.Tx, users []User) error {
	for _, user := range users {
		if err := insertSQLiteUser(tx, user); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteSessions(tx *sql.Tx, sessions []Session) error {
	for _, session := range sessions {
		if err := insertSQLiteSession(tx, session); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteWallets(tx *sql.Tx, wallets []Wallet) error {
	for _, wallet := range wallets {
		if err := insertSQLiteWallet(tx, wallet); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteCategories(tx *sql.Tx, categories []Category) error {
	stmt, err := tx.Prepare(`INSERT INTO categories (id, user_id, name, type, icon, color, is_default) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, category := range categories {
		if _, err := stmt.Exec(category.ID, nullableString(category.UserID), category.Name, category.Type, category.Icon, category.Color, boolInt(category.IsDefault)); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteRecurringTransactions(tx *sql.Tx, recurringTransactions []RecurringTransaction) error {
	for _, recurring := range recurringTransactions {
		if err := insertSQLiteRecurringTransaction(tx, recurring); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteTransactions(tx *sql.Tx, transactions []Transaction) error {
	for _, transaction := range transactions {
		if err := insertSQLiteTransaction(tx, transaction); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteBudgets(tx *sql.Tx, budgets []Budget) error {
	for _, budget := range budgets {
		if err := insertSQLiteBudget(tx, budget); err != nil {
			return err
		}
	}
	return nil
}

func insertSQLiteNotificationRules(tx *sql.Tx, rules []json.RawMessage) error {
	stmt, err := tx.Prepare(`INSERT INTO notification_rules (id, user_id, payload) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, rule := range rules {
		var payload map[string]any
		_ = json.Unmarshal(rule, &payload)
		id, _ := payload["id"].(string)
		if id == "" {
			id = "notification-rule-" + uuid()
		}
		userID, _ := payload["userId"].(string)
		if userID == "" {
			if _, err := stmt.Exec(id, nil, string(rule)); err != nil {
				return err
			}
			continue
		}
		if _, err := stmt.Exec(id, userID, string(rule)); err != nil {
			return err
		}
	}
	return nil
}

func readSQLiteTables(db *sql.DB) (DB, error) {
	users, err := readSQLiteUsers(db)
	if err != nil {
		return DB{}, err
	}
	sessions, err := readSQLiteSessions(db)
	if err != nil {
		return DB{}, err
	}
	wallets, err := readSQLiteWallets(db)
	if err != nil {
		return DB{}, err
	}
	categories, err := readSQLiteCategories(db)
	if err != nil {
		return DB{}, err
	}
	transactions, err := readSQLiteTransactions(db)
	if err != nil {
		return DB{}, err
	}
	budgets, err := readSQLiteBudgets(db)
	if err != nil {
		return DB{}, err
	}
	recurringTransactions, err := readSQLiteRecurringTransactions(db)
	if err != nil {
		return DB{}, err
	}
	notificationRules, err := readSQLiteNotificationRules(db)
	if err != nil {
		return DB{}, err
	}
	return DB{
		Users:                 users,
		Sessions:              sessions,
		Wallets:               wallets,
		Categories:            categories,
		Transactions:          transactions,
		Budgets:               budgets,
		RecurringTransactions: recurringTransactions,
		NotificationRules:     notificationRules,
	}, nil
}

func readSQLiteUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`SELECT id, email, name, password_hash, password_salt, created_at, updated_at FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []User{}
	for rows.Next() {
		var item User
		if err := rows.Scan(&item.ID, &item.Email, &item.Name, &item.PasswordHash, &item.PasswordSalt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func readSQLiteSessions(db *sql.DB) ([]Session, error) {
	rows, err := db.Query(`SELECT token_hash, user_id, created_at, expires_at FROM sessions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Session{}
	for rows.Next() {
		var item Session
		if err := rows.Scan(&item.TokenHash, &item.UserID, &item.CreatedAt, &item.ExpiresAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func readSQLiteWallets(db *sql.DB) ([]Wallet, error) {
	rows, err := db.Query(`SELECT id, user_id, name, currency, balance_initial, created_at FROM wallets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Wallet{}
	for rows.Next() {
		var item Wallet
		if err := rows.Scan(&item.ID, &item.UserID, &item.Name, &item.Currency, &item.BalanceInitial, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func readSQLiteCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query(`SELECT id, user_id, name, type, icon, color, is_default FROM categories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Category{}
	for rows.Next() {
		var item Category
		var userID sql.NullString
		var isDefault int
		if err := rows.Scan(&item.ID, &userID, &item.Name, &item.Type, &item.Icon, &item.Color, &isDefault); err != nil {
			return nil, err
		}
		item.UserID = stringPtrFromNull(userID)
		item.IsDefault = isDefault != 0
		items = append(items, item)
	}
	return items, rows.Err()
}

func readSQLiteTransactions(db *sql.DB) ([]Transaction, error) {
	rows, err := db.Query(`
SELECT id, user_id, wallet_id, category_id, type, amount, note, transaction_date,
source_recurring_id, recurring_run_at, recurring_run_date, sync_status, created_at, updated_at, deleted_at
FROM transactions`)
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

func readSQLiteBudgets(db *sql.DB) ([]Budget, error) {
	rows, err := db.Query(`SELECT id, user_id, category_id, amount_limit, period, start_date, end_date, created_at, updated_at, deleted_at FROM budgets`)
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

func readSQLiteRecurringTransactions(db *sql.DB) ([]RecurringTransaction, error) {
	rows, err := db.Query(`
SELECT id, user_id, wallet_id, category_id, type, amount, note, frequency,
next_run_at, next_run_date, active, created_at, updated_at, deleted_at
FROM recurring_transactions`)
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

func readSQLiteNotificationRules(db *sql.DB) ([]json.RawMessage, error) {
	rows, err := db.Query(`SELECT payload FROM notification_rules`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []json.RawMessage{}
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		items = append(items, json.RawMessage(payload))
	}
	return items, rows.Err()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableString(value *string) any {
	if value == nil || *value == "" {
		return nil
	}
	return *value
}

func stringPtrFromNull(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
