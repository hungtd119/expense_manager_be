package httpapi

import (
	"math"
	"sort"
	"strings"
	"time"
)

func recurringForUser(db *DB, userID string) []RecurringTransaction {
	result := []RecurringTransaction{}
	for _, recurring := range db.RecurringTransactions {
		if recurring.UserID == userID && recurring.DeletedAt == nil {
			result = append(result, recurring)
		}
	}
	sortRecurring(result)
	return result
}

func sortRecurring(items []RecurringTransaction) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Active != items[j].Active {
			return items[i].Active
		}
		return items[i].NextRunAt < items[j].NextRunAt
	})
}

func validateRecurring(db *DB, userID string, body map[string]any, existing *RecurringTransaction) (map[string]any, string) {
	typeValue := valueOr(body, "type", existingRecurringString(existing, "type"))
	amount := numberOr(body, "amount", existingRecurringAmount(existing))
	walletID := valueOr(body, "walletId", existingRecurringString(existing, "walletId"))
	categoryID := valueOr(body, "categoryId", existingRecurringString(existing, "categoryId"))
	note := strings.TrimSpace(valueOr(body, "note", existingRecurringString(existing, "note")))
	frequency := valueOr(body, "frequency", existingRecurringString(existing, "frequency"))
	nextRunAt := valueOr(body, "nextRunAt", existingRecurringString(existing, "nextRunAt"))
	if nextRunAt == "" {
		nextRunAt = valueOr(body, "nextRunDate", existingRecurringString(existing, "nextRunDate"))
	}
	active := true
	if existing != nil {
		active = existing.Active
	}
	if raw, ok := body["active"].(bool); ok {
		active = raw
	}
	if typeValue != "income" && typeValue != "expense" {
		return nil, "Loai giao dich dinh ky khong hop le."
	}
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return nil, "So tien dinh ky phai lon hon 0."
	}
	if frequency != "daily" && frequency != "weekly" && frequency != "monthly" {
		return nil, "Tan suat dinh ky khong hop le."
	}
	if !isDateTimeLocal(nextRunAt) {
		return nil, "Thoi diem chay tiep theo khong hop le."
	}
	if !walletExists(db, userID, walletID) {
		return nil, "Vi khong ton tai."
	}
	if !categoryValid(db, userID, categoryID, typeValue) {
		return nil, "Danh muc khong hop le voi loai giao dich dinh ky."
	}
	return map[string]any{"type": typeValue, "amount": amount, "walletId": walletID, "categoryId": categoryID, "note": note, "frequency": frequency, "nextRunAt": nextRunAt, "nextRunDate": nextRunAt[:10], "active": active}, ""
}

func sanitizeRecurring(recurring RecurringTransaction, db *DB) map[string]any {
	category := categoryByID(db, recurring.CategoryID)
	wallet := walletByID(db, recurring.WalletID)
	return map[string]any{"id": recurring.ID, "userId": recurring.UserID, "walletId": recurring.WalletID, "walletName": wallet.Name, "categoryId": recurring.CategoryID, "categoryName": category.Name, "categoryColor": category.Color, "type": recurring.Type, "amount": recurring.Amount, "note": recurring.Note, "frequency": recurring.Frequency, "nextRunAt": recurring.NextRunAt, "nextRunDate": recurring.NextRunDate, "active": recurring.Active, "createdAt": recurring.CreatedAt, "updatedAt": recurring.UpdatedAt, "deletedAt": recurring.DeletedAt}
}

func processDueRecurring(db *DB, userID string, untilAt string) RecurringResult {
	result := RecurringResult{}
	now := nowISO()
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
				db.Transactions = append(db.Transactions, Transaction{ID: uuid(), UserID: userID, WalletID: recurring.WalletID, CategoryID: recurring.CategoryID, Type: recurring.Type, Amount: recurring.Amount, Note: recurring.Note, TransactionDate: runDate, SourceRecurringID: &sourceID, RecurringRunAt: &runAtCopy, RecurringRunDate: &runDateCopy, SyncStatus: "synced", CreatedAt: now, UpdatedAt: now})
				result.GeneratedCount++
				result.Changed = true
			}
			runAt = nextRunAt(runAt, recurring.Frequency)
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

func nextRunAt(currentRunAt string, frequency string) string {
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
