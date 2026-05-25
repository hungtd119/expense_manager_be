package httpapi

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parseMonth(w http.ResponseWriter, r *http.Request, requestID string) (MonthBounds, bool) {
	bounds, ok := monthBounds(r.URL.Query().Get("month"))
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Thang can co dinh dang YYYY-MM.", map[string]any{"field": "month"}, requestID)
	}
	return bounds, ok
}

func monthBounds(month string) (MonthBounds, bool) {
	parts := strings.Split(month, "-")
	if len(parts) != 2 || len(parts[0]) != 4 || len(parts[1]) != 2 {
		return MonthBounds{}, false
	}
	year, errY := strconv.Atoi(parts[0])
	monthNumber, errM := strconv.Atoi(parts[1])
	if errY != nil || errM != nil || monthNumber < 1 || monthNumber > 12 {
		return MonthBounds{}, false
	}
	start := time.Date(year, time.Month(monthNumber), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return MonthBounds{StartDate: start.Format("2006-01-02"), EndDate: end.Format("2006-01-02")}, true
}

func isDateOnly(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func isDateTimeLocal(value string) bool {
	_, err := time.Parse("2006-01-02T15:04", value)
	return err == nil
}

func localDateTimeNow() string {
	return time.Now().Format("2006-01-02T15:04")
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func filterTransactions(items []map[string]any, r *http.Request) ([]map[string]any, map[string]any, bool) {
	query := r.URL.Query()
	typeValue := query.Get("type")
	if typeValue != "" && typeValue != "income" && typeValue != "expense" {
		return nil, nil, false
	}
	categoryID := query.Get("categoryId")
	walletID := query.Get("walletId")
	q := strings.ToLower(strings.TrimSpace(query.Get("q")))
	var result []map[string]any
	for _, item := range items {
		if typeValue != "" && item["type"] != typeValue {
			continue
		}
		if categoryID != "" && item["categoryId"] != categoryID {
			continue
		}
		if walletID != "" && item["walletId"] != walletID {
			continue
		}
		if q != "" {
			haystack := strings.ToLower(fmt.Sprintf("%s %s %s", item["note"], item["categoryName"], item["walletName"]))
			if !strings.Contains(haystack, q) {
				continue
			}
		}
		result = append(result, item)
	}
	return result, map[string]any{"type": nilOrString(typeValue), "categoryId": nilOrString(categoryID), "walletId": nilOrString(walletID), "q": nilOrString(q)}, true
}

func parsePagination(r *http.Request) (int, int, bool) {
	page := intQuery(r, "page", 1)
	pageSize := intQuery(r, "pageSize", 50)
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return 0, 0, false
	}
	return page, pageSize, true
}

func paginate(items []map[string]any, page int, pageSize int) ([]map[string]any, map[string]any) {
	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}
	return items[start:end], map[string]any{"page": page, "pageSize": pageSize, "total": total, "totalPages": totalPages}
}

func intQuery(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return -1
	}
	return value
}
