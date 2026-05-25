package httpapi

import (
	"math"
	"sort"
	"strings"
)

func transactionsForMonth(db *DB, userID string, bounds MonthBounds) []Transaction {
	var result []Transaction
	for _, tx := range db.Transactions {
		if tx.UserID == userID && tx.DeletedAt == nil && tx.TransactionDate >= bounds.StartDate && tx.TransactionDate < bounds.EndDate {
			result = append(result, tx)
		}
	}
	sortTransactions(result)
	return result
}

func sortTransactions(items []Transaction) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].TransactionDate == items[j].TransactionDate {
			return items[i].CreatedAt > items[j].CreatedAt
		}
		return items[i].TransactionDate > items[j].TransactionDate
	})
}

func sanitizeTransactions(transactions []Transaction, db *DB) []map[string]any {
	result := make([]map[string]any, 0, len(transactions))
	for _, tx := range transactions {
		result = append(result, sanitizeTransaction(tx, db))
	}
	return result
}

func sanitizeTransaction(tx Transaction, db *DB) map[string]any {
	category := categoryByID(db, tx.CategoryID)
	wallet := walletByID(db, tx.WalletID)
	return map[string]any{
		"id": tx.ID, "userId": tx.UserID, "walletId": tx.WalletID, "walletName": wallet.Name,
		"categoryId": tx.CategoryID, "categoryName": category.Name, "categoryColor": category.Color,
		"type": tx.Type, "amount": tx.Amount, "note": tx.Note, "transactionDate": tx.TransactionDate,
		"sourceRecurringId": tx.SourceRecurringID, "recurringRunAt": tx.RecurringRunAt, "recurringRunDate": tx.RecurringRunDate,
		"syncStatus": tx.SyncStatus, "createdAt": tx.CreatedAt, "updatedAt": tx.UpdatedAt, "deletedAt": tx.DeletedAt,
	}
}

func validateTransaction(db *DB, userID string, body map[string]any, existing *Transaction) (map[string]any, string) {
	typeValue := valueOr(body, "type", existingString(existing, "type"))
	amount := numberOr(body, "amount", existingAmount(existing))
	walletID := valueOr(body, "walletId", existingString(existing, "walletId"))
	categoryID := valueOr(body, "categoryId", existingString(existing, "categoryId"))
	transactionDate := valueOr(body, "transactionDate", existingString(existing, "transactionDate"))
	note := strings.TrimSpace(valueOr(body, "note", existingString(existing, "note")))
	if typeValue != "income" && typeValue != "expense" {
		return nil, "Loai giao dich khong hop le."
	}
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return nil, "So tien phai lon hon 0."
	}
	if !isDateOnly(transactionDate) {
		return nil, "Ngay giao dich khong hop le."
	}
	if !walletExists(db, userID, walletID) {
		return nil, "Vi khong ton tai."
	}
	if !categoryValid(db, userID, categoryID, typeValue) {
		return nil, "Danh muc khong hop le voi loai giao dich."
	}
	return map[string]any{"type": typeValue, "amount": amount, "walletId": walletID, "categoryId": categoryID, "transactionDate": transactionDate, "note": note}, ""
}
