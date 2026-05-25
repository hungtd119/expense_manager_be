package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"

	"expense-manager-mvp/internal/domain"
)

var (
	errNotFound      = domain.ErrNotFound
	errAlreadyExists = domain.ErrAlreadyExists
)

func valueOr(body map[string]any, key string, fallback string) string {
	if value, ok := body[key]; ok {
		return str(value)
	}
	return fallback
}

func numberOr(body map[string]any, key string, fallback float64) float64 {
	if value, ok := body[key]; ok {
		switch typed := value.(type) {
		case float64:
			return typed
		case int:
			return float64(typed)
		case string:
			parsed, _ := strconv.ParseFloat(typed, 64)
			return parsed
		default:
			return math.NaN()
		}
	}
	return fallback
}

func existingString(tx *Transaction, field string) string {
	if tx == nil {
		return ""
	}
	switch field {
	case "type":
		return tx.Type
	case "walletId":
		return tx.WalletID
	case "categoryId":
		return tx.CategoryID
	case "transactionDate":
		return tx.TransactionDate
	case "note":
		return tx.Note
	default:
		return ""
	}
}

func existingAmount(tx *Transaction) float64 {
	if tx == nil {
		return 0
	}
	return tx.Amount
}

func existingBudgetString(budget *Budget, field string) string {
	if budget == nil {
		return ""
	}
	if field == "categoryId" {
		return budget.CategoryID
	}
	return ""
}

func existingBudgetAmount(budget *Budget) float64 {
	if budget == nil {
		return 0
	}
	return budget.AmountLimit
}

func existingRecurringString(recurring *RecurringTransaction, field string) string {
	if recurring == nil {
		return ""
	}
	switch field {
	case "type":
		return recurring.Type
	case "walletId":
		return recurring.WalletID
	case "categoryId":
		return recurring.CategoryID
	case "note":
		return recurring.Note
	case "frequency":
		return recurring.Frequency
	case "nextRunAt":
		return recurring.NextRunAt
	case "nextRunDate":
		return recurring.NextRunDate
	default:
		return ""
	}
}

func existingRecurringAmount(recurring *RecurringTransaction) float64 {
	if recurring == nil {
		return 0
	}
	return recurring.Amount
}

func str(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func nilOrString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}

func randomHex(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func uuid() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:])
}
