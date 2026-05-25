package usecase

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"expense-manager-mvp/internal/domain"
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

func isDateOnly(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func isDateTimeLocal(value string) bool {
	_, err := time.ParseInLocation("2006-01-02T15:04", value, time.Local)
	return err == nil && len(strings.TrimSpace(value)) == len("2006-01-02T15:04")
}

func existingTransactionString(tx *domain.Transaction, field string) string {
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

func existingTransactionAmount(tx *domain.Transaction) float64 {
	if tx == nil {
		return 0
	}
	return tx.Amount
}
