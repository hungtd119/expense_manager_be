package domain

import "encoding/json"

type DB struct {
	Users                 []User                 `json:"users"`
	Sessions              []Session              `json:"sessions"`
	Wallets               []Wallet               `json:"wallets"`
	Categories            []Category             `json:"categories"`
	Transactions          []Transaction          `json:"transactions"`
	Budgets               []Budget               `json:"budgets"`
	RecurringTransactions []RecurringTransaction `json:"recurringTransactions"`
	NotificationRules     []json.RawMessage      `json:"notificationRules"`
}

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"passwordHash"`
	PasswordSalt string `json:"passwordSalt"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type Session struct {
	TokenHash string `json:"tokenHash"`
	UserID    string `json:"userId"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
}

type Wallet struct {
	ID             string  `json:"id"`
	UserID         string  `json:"userId"`
	Name           string  `json:"name"`
	Currency       string  `json:"currency"`
	BalanceInitial float64 `json:"balanceInitial"`
	CreatedAt      string  `json:"createdAt"`
}

type Category struct {
	ID        string  `json:"id"`
	UserID    *string `json:"userId"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Icon      string  `json:"icon"`
	Color     string  `json:"color"`
	IsDefault bool    `json:"isDefault"`
}

type Transaction struct {
	ID                string  `json:"id"`
	UserID            string  `json:"userId"`
	WalletID          string  `json:"walletId"`
	CategoryID        string  `json:"categoryId"`
	Type              string  `json:"type"`
	Amount            float64 `json:"amount"`
	Note              string  `json:"note"`
	TransactionDate   string  `json:"transactionDate"`
	SourceRecurringID *string `json:"sourceRecurringId"`
	RecurringRunAt    *string `json:"recurringRunAt"`
	RecurringRunDate  *string `json:"recurringRunDate"`
	SyncStatus        string  `json:"syncStatus"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
	DeletedAt         *string `json:"deletedAt"`
}

type Budget struct {
	ID          string  `json:"id"`
	UserID      string  `json:"userId"`
	CategoryID  string  `json:"categoryId"`
	AmountLimit float64 `json:"amountLimit"`
	Period      string  `json:"period"`
	StartDate   string  `json:"startDate"`
	EndDate     string  `json:"endDate"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	DeletedAt   *string `json:"deletedAt"`
}

type RecurringTransaction struct {
	ID          string  `json:"id"`
	UserID      string  `json:"userId"`
	WalletID    string  `json:"walletId"`
	CategoryID  string  `json:"categoryId"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Note        string  `json:"note"`
	Frequency   string  `json:"frequency"`
	NextRunAt   string  `json:"nextRunAt"`
	NextRunDate string  `json:"nextRunDate"`
	Active      bool    `json:"active"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	DeletedAt   *string `json:"deletedAt"`
}

type MonthBounds struct {
	StartDate string
	EndDate   string
}

type RecurringResult struct {
	GeneratedCount int
	Changed        bool
}
