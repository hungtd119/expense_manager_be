package mysqlstore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLStore struct {
	dsn        string
	importPath string
	mu         sync.Mutex
}

func NewMySQLStore(dsn string, importPath string) *MySQLStore {
	return &MySQLStore{dsn: dsn, importPath: importPath}
}

func (s *MySQLStore) Ensure() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := createMySQLSchema(db); err != nil {
		return err
	}
	return s.migrateInitialState(db)
}

func (s *MySQLStore) Read() (DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dbConn, err := s.open()
	if err != nil {
		return DB{}, err
	}
	defer dbConn.Close()

	db, err := readMySQLTables(dbConn)
	if err != nil {
		return DB{}, err
	}
	if migrateShape(&db) {
		if err := writeMySQLTables(dbConn, db); err != nil {
			return DB{}, err
		}
	}
	return db, nil
}

func (s *MySQLStore) Write(db DB) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dbConn, err := s.open()
	if err != nil {
		return err
	}
	defer dbConn.Close()

	return writeMySQLTables(dbConn, db)
}

func (s *MySQLStore) Driver() string {
	return "mysql"
}

func (s *MySQLStore) Location() string {
	return s.dsn
}

func (s *MySQLStore) open() (*sql.DB, error) {
	db, err := sql.Open("mysql", s.dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (s *MySQLStore) migrateInitialState(db *sql.DB) error {
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
	if err := clearMySQLEntityTables(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := insertMySQLState(tx, initialState); err != nil {
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

func createMySQLSchema(db *sql.DB) error {
	return execMySQLStatements(db, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INT PRIMARY KEY,
  name VARCHAR(191) NOT NULL,
  applied_at VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS users (
  id VARCHAR(64) PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  password_salt VARCHAR(255) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sessions (
  token_hash VARCHAR(128) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  expires_at VARCHAR(64) NOT NULL,
  INDEX idx_sessions_expiry (expires_at),
  CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS wallets (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  currency VARCHAR(16) NOT NULL,
  balance_initial DOUBLE NOT NULL DEFAULT 0,
  created_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_wallets_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS categories (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64),
  name VARCHAR(255) NOT NULL,
  type VARCHAR(32) NOT NULL,
  icon VARCHAR(64) NOT NULL,
  color VARCHAR(32) NOT NULL,
  is_default TINYINT(1) NOT NULL DEFAULT 0,
  CONSTRAINT fk_categories_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS recurring_transactions (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  wallet_id VARCHAR(64) NOT NULL,
  category_id VARCHAR(64) NOT NULL,
  type VARCHAR(32) NOT NULL,
  amount DOUBLE NOT NULL,
  note TEXT NOT NULL,
  frequency VARCHAR(32) NOT NULL,
  next_run_at VARCHAR(64) NOT NULL,
  next_run_date VARCHAR(16) NOT NULL,
  active TINYINT(1) NOT NULL DEFAULT 1,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  deleted_at VARCHAR(64),
  INDEX idx_recurring_user_due (user_id, active, deleted_at, next_run_at),
  CONSTRAINT fk_recurring_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_recurring_wallet FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE RESTRICT,
  CONSTRAINT fk_recurring_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS transactions (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  wallet_id VARCHAR(64) NOT NULL,
  category_id VARCHAR(64) NOT NULL,
  type VARCHAR(32) NOT NULL,
  amount DOUBLE NOT NULL,
  note TEXT NOT NULL,
  transaction_date VARCHAR(16) NOT NULL,
  source_recurring_id VARCHAR(64),
  recurring_run_at VARCHAR(64),
  recurring_run_date VARCHAR(16),
  sync_status VARCHAR(32) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  deleted_at VARCHAR(64),
  active_source_recurring_id VARCHAR(64) GENERATED ALWAYS AS (IF(deleted_at IS NULL, source_recurring_id, NULL)) STORED,
  active_recurring_run_at VARCHAR(64) GENERATED ALWAYS AS (IF(deleted_at IS NULL, recurring_run_at, NULL)) STORED,
  UNIQUE KEY idx_transactions_recurring_run (active_source_recurring_id, active_recurring_run_at),
  INDEX idx_transactions_user_month (user_id, transaction_date, deleted_at),
  CONSTRAINT fk_transactions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_transactions_wallet FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE RESTRICT,
  CONSTRAINT fk_transactions_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT,
  CONSTRAINT fk_transactions_recurring FOREIGN KEY (source_recurring_id) REFERENCES recurring_transactions(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS budgets (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  category_id VARCHAR(64) NOT NULL,
  amount_limit DOUBLE NOT NULL,
  period VARCHAR(32) NOT NULL,
  start_date VARCHAR(16) NOT NULL,
  end_date VARCHAR(16) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  deleted_at VARCHAR(64),
  active_user_id VARCHAR(64) GENERATED ALWAYS AS (IF(deleted_at IS NULL, user_id, NULL)) STORED,
  active_start_date VARCHAR(16) GENERATED ALWAYS AS (IF(deleted_at IS NULL, start_date, NULL)) STORED,
  active_category_id VARCHAR(64) GENERATED ALWAYS AS (IF(deleted_at IS NULL, category_id, NULL)) STORED,
  UNIQUE KEY idx_budgets_unique_month_category (active_user_id, active_start_date, active_category_id),
  INDEX idx_budgets_user_month (user_id, start_date, deleted_at),
  CONSTRAINT fk_budgets_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
  CONSTRAINT fk_budgets_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS notification_rules (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64),
  payload TEXT NOT NULL,
  CONSTRAINT fk_notification_rules_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`)
}

func execMySQLStatements(db *sql.DB, statements string) error {
	for _, statement := range strings.Split(statements, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func writeMySQLTables(db *sql.DB, state DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := clearMySQLEntityTables(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := insertMySQLState(tx, state); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func clearMySQLEntityTables(tx *sql.Tx) error {
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

func insertMySQLState(tx *sql.Tx, db DB) error {
	if err := insertMySQLUsers(tx, db.Users); err != nil {
		return err
	}
	if err := insertMySQLCategories(tx, db.Categories); err != nil {
		return err
	}
	if err := insertMySQLWallets(tx, db.Wallets); err != nil {
		return err
	}
	if err := insertMySQLRecurringTransactions(tx, db.RecurringTransactions); err != nil {
		return err
	}
	if err := insertMySQLTransactions(tx, db.Transactions); err != nil {
		return err
	}
	if err := insertMySQLBudgets(tx, db.Budgets); err != nil {
		return err
	}
	if err := insertMySQLSessions(tx, db.Sessions); err != nil {
		return err
	}
	return insertMySQLNotificationRules(tx, db.NotificationRules)
}

func insertMySQLUsers(tx *sql.Tx, users []User) error {
	for _, user := range users {
		if err := insertMySQLUser(tx, user); err != nil {
			return err
		}
	}
	return nil
}

func insertMySQLSessions(tx *sql.Tx, sessions []Session) error {
	for _, session := range sessions {
		if err := insertMySQLSession(tx, session); err != nil {
			return err
		}
	}
	return nil
}

func insertMySQLWallets(tx *sql.Tx, wallets []Wallet) error {
	for _, wallet := range wallets {
		if err := insertMySQLWallet(tx, wallet); err != nil {
			return err
		}
	}
	return nil
}

func insertMySQLCategories(tx *sql.Tx, categories []Category) error {
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

func insertMySQLRecurringTransactions(tx *sql.Tx, recurringTransactions []RecurringTransaction) error {
	for _, recurring := range recurringTransactions {
		if err := insertMySQLRecurringTransaction(tx, recurring); err != nil {
			return err
		}
	}
	return nil
}

func insertMySQLTransactions(tx *sql.Tx, transactions []Transaction) error {
	for _, transaction := range transactions {
		if err := insertMySQLTransaction(tx, transaction); err != nil {
			return err
		}
	}
	return nil
}

func insertMySQLBudgets(tx *sql.Tx, budgets []Budget) error {
	for _, budget := range budgets {
		if err := insertMySQLBudget(tx, budget); err != nil {
			return err
		}
	}
	return nil
}

func insertMySQLNotificationRules(tx *sql.Tx, rules []json.RawMessage) error {
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

func readMySQLTables(db *sql.DB) (DB, error) {
	users, err := readMySQLUsers(db)
	if err != nil {
		return DB{}, err
	}
	sessions, err := readMySQLSessions(db)
	if err != nil {
		return DB{}, err
	}
	wallets, err := readMySQLWallets(db)
	if err != nil {
		return DB{}, err
	}
	categories, err := readMySQLCategories(db)
	if err != nil {
		return DB{}, err
	}
	transactions, err := readMySQLTransactions(db)
	if err != nil {
		return DB{}, err
	}
	budgets, err := readMySQLBudgets(db)
	if err != nil {
		return DB{}, err
	}
	recurringTransactions, err := readMySQLRecurringTransactions(db)
	if err != nil {
		return DB{}, err
	}
	notificationRules, err := readMySQLNotificationRules(db)
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

func readMySQLUsers(db *sql.DB) ([]User, error) {
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

func readMySQLSessions(db *sql.DB) ([]Session, error) {
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

func readMySQLWallets(db *sql.DB) ([]Wallet, error) {
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

func readMySQLCategories(db *sql.DB) ([]Category, error) {
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

func readMySQLTransactions(db *sql.DB) ([]Transaction, error) {
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

func readMySQLBudgets(db *sql.DB) ([]Budget, error) {
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

func readMySQLRecurringTransactions(db *sql.DB) ([]RecurringTransaction, error) {
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

func readMySQLNotificationRules(db *sql.DB) ([]json.RawMessage, error) {
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
