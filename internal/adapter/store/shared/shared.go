package shared

import (
	"encoding/json"
	"sort"
	"time"

	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/platform"
)

func EmptyDB() domain.DB {
	return domain.DB{
		Users:                 []domain.User{},
		Sessions:              []domain.Session{},
		Wallets:               []domain.Wallet{},
		Categories:            DefaultCategories(),
		Transactions:          []domain.Transaction{},
		Budgets:               []domain.Budget{},
		RecurringTransactions: []domain.RecurringTransaction{},
		NotificationRules:     []json.RawMessage{},
	}
}

func DefaultCategories() []domain.Category {
	ids := platform.CryptoIDGenerator{}
	return []domain.Category{
		{ID: ids.UUID(), Name: "An uong", Type: "expense", Icon: "utensils", Color: "#ef4444", IsDefault: true},
		{ID: ids.UUID(), Name: "Di lai", Type: "expense", Icon: "car", Color: "#f59e0b", IsDefault: true},
		{ID: ids.UUID(), Name: "Nha cua", Type: "expense", Icon: "home", Color: "#10b981", IsDefault: true},
		{ID: ids.UUID(), Name: "Mua sam", Type: "expense", Icon: "shopping-bag", Color: "#8b5cf6", IsDefault: true},
		{ID: ids.UUID(), Name: "Giai tri", Type: "expense", Icon: "film", Color: "#06b6d4", IsDefault: true},
		{ID: ids.UUID(), Name: "Suc khoe", Type: "expense", Icon: "heart-pulse", Color: "#ec4899", IsDefault: true},
		{ID: ids.UUID(), Name: "Hoc tap", Type: "expense", Icon: "book-open", Color: "#3b82f6", IsDefault: true},
		{ID: ids.UUID(), Name: "Luong", Type: "income", Icon: "wallet", Color: "#22c55e", IsDefault: true},
		{ID: ids.UUID(), Name: "Thu nhap khac", Type: "income", Icon: "plus-circle", Color: "#14b8a6", IsDefault: true},
	}
}

func MigrateShape(db *domain.DB) bool {
	changed := false
	for i := range db.RecurringTransactions {
		item := &db.RecurringTransactions[i]
		if item.NextRunAt == "" && item.NextRunDate != "" {
			item.NextRunAt = item.NextRunDate + "T00:00"
			changed = true
		}
	}
	for i := range db.Transactions {
		item := &db.Transactions[i]
		if item.SourceRecurringID != nil && item.RecurringRunAt == nil && item.RecurringRunDate != nil {
			runAt := *item.RecurringRunDate + "T00:00"
			item.RecurringRunAt = &runAt
			changed = true
		}
	}
	return changed
}

func TransactionsForMonth(db *domain.DB, userID string, bounds domain.MonthBounds) []domain.Transaction {
	var result []domain.Transaction
	for _, tx := range db.Transactions {
		if tx.UserID == userID && tx.DeletedAt == nil && tx.TransactionDate >= bounds.StartDate && tx.TransactionDate < bounds.EndDate {
			result = append(result, tx)
		}
	}
	SortTransactions(result)
	return result
}

func SortTransactions(items []domain.Transaction) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].TransactionDate == items[j].TransactionDate {
			return items[i].CreatedAt > items[j].CreatedAt
		}
		return items[i].TransactionDate > items[j].TransactionDate
	})
}

func BudgetsForMonth(db *domain.DB, userID string, bounds domain.MonthBounds) []domain.Budget {
	result := []domain.Budget{}
	for _, budget := range db.Budgets {
		if budget.UserID == userID && budget.DeletedAt == nil && budget.Period == "monthly" && budget.StartDate == bounds.StartDate {
			result = append(result, budget)
		}
	}
	SortBudgets(result)
	return result
}

func SortBudgets(items []domain.Budget) {
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })
}

func RecurringForUser(db *domain.DB, userID string) []domain.RecurringTransaction {
	result := []domain.RecurringTransaction{}
	for _, recurring := range db.RecurringTransactions {
		if recurring.UserID == userID && recurring.DeletedAt == nil {
			result = append(result, recurring)
		}
	}
	SortRecurring(result)
	return result
}

func SortRecurring(items []domain.RecurringTransaction) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Active != items[j].Active {
			return items[i].Active
		}
		return items[i].NextRunAt < items[j].NextRunAt
	})
}

func ProcessDueRecurring(db *domain.DB, userID string, untilAt string) domain.RecurringResult {
	result := domain.RecurringResult{}
	now := platform.NowISO(platform.SystemClock{})
	ids := platform.CryptoIDGenerator{}
	for i := range db.RecurringTransactions {
		recurring := &db.RecurringTransactions[i]
		if recurring.UserID != userID || !recurring.Active || recurring.DeletedAt != nil {
			continue
		}
		runAt := recurring.NextRunAt
		guard := 0
		for runAt != "" && runAt <= untilAt && guard < 36 {
			runDate := runAt[:10]
			alreadyGenerated := false
			for _, tx := range db.Transactions {
				if tx.DeletedAt == nil && tx.UserID == userID && tx.SourceRecurringID != nil && *tx.SourceRecurringID == recurring.ID && tx.RecurringRunAt != nil && *tx.RecurringRunAt == runAt {
					alreadyGenerated = true
					break
				}
			}
			if !alreadyGenerated {
				sourceID := recurring.ID
				runAtCopy := runAt
				runDateCopy := runDate
				db.Transactions = append(db.Transactions, domain.Transaction{ID: ids.UUID(), UserID: userID, WalletID: recurring.WalletID, CategoryID: recurring.CategoryID, Type: recurring.Type, Amount: recurring.Amount, Note: recurring.Note, TransactionDate: runDate, SourceRecurringID: &sourceID, RecurringRunAt: &runAtCopy, RecurringRunDate: &runDateCopy, SyncStatus: "synced", CreatedAt: now, UpdatedAt: now})
				result.GeneratedCount++
				result.Changed = true
			}
			runAt = NextRunAt(runAt, recurring.Frequency)
			guard++
		}
		if runAt != "" && runAt != recurring.NextRunAt {
			recurring.NextRunAt = runAt
			recurring.NextRunDate = runAt[:10]
			recurring.UpdatedAt = now
			result.Changed = true
		}
	}
	return result
}

func NextRunAt(currentRunAt string, frequency string) string {
	date, err := time.ParseInLocation("2006-01-02T15:04", currentRunAt, time.Local)
	if err != nil {
		return ""
	}
	switch frequency {
	case "daily":
		return date.AddDate(0, 0, 1).Format("2006-01-02T15:04")
	case "weekly":
		return date.AddDate(0, 0, 7).Format("2006-01-02T15:04")
	case "monthly":
		day := date.Day()
		next := date.AddDate(0, 1, 0)
		lastDay := time.Date(next.Year(), next.Month()+1, 0, next.Hour(), next.Minute(), 0, 0, time.Local).Day()
		if day > lastDay {
			day = lastDay
		}
		return time.Date(next.Year(), next.Month(), day, next.Hour(), next.Minute(), 0, 0, time.Local).Format("2006-01-02T15:04")
	default:
		return ""
	}
}

func ParseSessionTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, value)
	}
	return parsed, err
}
