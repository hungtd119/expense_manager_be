package httpapi

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
)

func (s *server) exportTransactions(c *gin.Context) {
	s.ExportTransactions(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) ExportTransactions(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	month := r.URL.Query().Get("month")

	var transactions []Transaction

	if month != "" {
		bounds, ok := monthBounds(month)
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Thang can co dinh dang YYYY-MM.", map[string]any{"field": "month"}, requestID)
			return
		}
		// Filter transactions for this month and this user
		for _, tx := range db.Transactions {
			if tx.UserID == user.ID && tx.DeletedAt == nil && tx.TransactionDate >= bounds.StartDate && tx.TransactionDate < bounds.EndDate {
				transactions = append(transactions, tx)
			}
		}
	} else {
		// All time transactions for this user
		for _, tx := range db.Transactions {
			if tx.UserID == user.ID && tx.DeletedAt == nil {
				transactions = append(transactions, tx)
			}
		}
	}

	// Sort transactions: newest first
	sort.Slice(transactions, func(i, j int) bool {
		if transactions[i].TransactionDate == transactions[j].TransactionDate {
			return transactions[i].CreatedAt > transactions[j].CreatedAt
		}
		return transactions[i].TransactionDate > transactions[j].TransactionDate
	})

	if format == "json" {
		// Output JSON format
		sanitized := make([]map[string]any, 0, len(transactions))
		for _, tx := range transactions {
			sanitized = append(sanitized, sanitizeTransaction(tx, &db))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="transactions.json"`)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(sanitized)
		return
	}

	// Default/CSV format
	filename := "transactions.csv"
	if month != "" {
		filename = fmt.Sprintf("transactions-%s.csv", month)
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	// Write BOM for Excel UTF-8 support
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header row
	_ = writer.Write([]string{"Ngay", "Loai", "Danh muc", "Vi", "So tien", "Ghi chu"})

	for _, tx := range transactions {
		category := categoryByID(&db, tx.CategoryID)
		wallet := walletByID(&db, tx.WalletID)
		
		typeStr := "Thu"
		if tx.Type == "expense" {
			typeStr = "Chi"
		}

		amountStr := fmt.Sprintf("%.0f", tx.Amount) // Format as whole number since it is VND

		row := []string{
			tx.TransactionDate,
			typeStr,
			category.Name,
			wallet.Name,
			amountStr,
			tx.Note,
		}
		_ = writer.Write(row)
	}
}
