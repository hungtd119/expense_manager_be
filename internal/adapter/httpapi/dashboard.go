package httpapi

import (
	"fmt"
	"math"
	"sort"
	"time"
)

func dashboardSummary(db *DB, transactions []Transaction, bounds MonthBounds) map[string]any {
	incomeTotal := 0.0
	expenseTotal := 0.0
	expenseCount := 0
	categoryMap := map[string]map[string]any{}
	for _, tx := range transactions {
		if tx.Type == "income" {
			incomeTotal += tx.Amount
		}
		if tx.Type == "expense" {
			expenseTotal += tx.Amount
			expenseCount++
			category := categoryByID(db, tx.CategoryID)
			item, ok := categoryMap[tx.CategoryID]
			if !ok {
				item = map[string]any{"categoryId": tx.CategoryID, "categoryName": category.Name, "categoryColor": category.Color, "amount": 0.0, "count": 0, "percent": 0.0}
				categoryMap[tx.CategoryID] = item
			}
			item["amount"] = item["amount"].(float64) + tx.Amount
			item["count"] = item["count"].(int) + 1
		}
	}
	balance := incomeTotal - expenseTotal
	savingsRate := 0.0
	if incomeTotal > 0 {
		savingsRate = round1(balance / incomeTotal * 100)
	}
	expenseByCategory := make([]map[string]any, 0, len(categoryMap))
	for _, item := range categoryMap {
		if expenseTotal > 0 {
			item["percent"] = round1(item["amount"].(float64) / expenseTotal * 100)
		}
		expenseByCategory = append(expenseByCategory, item)
	}
	sort.Slice(expenseByCategory, func(i, j int) bool {
		return expenseByCategory[i]["amount"].(float64) > expenseByCategory[j]["amount"].(float64)
	})
	var topCategory any
	insight := "Chua co du lieu chi tieu trong thang nay."
	if len(expenseByCategory) > 0 {
		topCategory = expenseByCategory[0]
		insight = fmt.Sprintf("%s la danh muc chi nhieu nhat, chiem %.1f%% tong chi.", expenseByCategory[0]["categoryName"], expenseByCategory[0]["percent"])
	}
	if incomeTotal > 0 && balance < 0 {
		insight = "Thang nay chi tieu dang vuot thu nhap."
	}
	averageExpense := 0.0
	if expenseCount > 0 {
		averageExpense = math.Round(expenseTotal / float64(expenseCount))
	}
	incomeCount := 0
	for _, tx := range transactions {
		if tx.Type == "income" {
			incomeCount++
		}
	}

	// Generate daily breakdown
	var dailyBreakdown []map[string]any
	start, err1 := time.Parse("2006-01-02", bounds.StartDate)
	end, err2 := time.Parse("2006-01-02", bounds.EndDate)
	if err1 == nil && err2 == nil {
		type amountPair struct {
			Income  float64
			Expense float64
		}
		dailyAmounts := make(map[string]amountPair)
		for _, tx := range transactions {
			dateStr := tx.TransactionDate // YYYY-MM-DD
			pair := dailyAmounts[dateStr]
			if tx.Type == "income" {
				pair.Income += tx.Amount
			} else if tx.Type == "expense" {
				pair.Expense += tx.Amount
			}
			dailyAmounts[dateStr] = pair
		}

		for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			pair := dailyAmounts[dateStr]
			dailyBreakdown = append(dailyBreakdown, map[string]any{
				"date":    dateStr,
				"income":  pair.Income,
				"expense": pair.Expense,
			})
		}
	} else {
		dailyBreakdown = []map[string]any{}
	}

	return map[string]any{
		"totals": map[string]any{
			"income":      incomeTotal,
			"expense":     expenseTotal,
			"balance":     balance,
			"savingsRate": savingsRate,
		},
		"counts": map[string]any{
			"all":     len(transactions),
			"income":  incomeCount,
			"expense": expenseCount,
		},
		"expenseByCategory": expenseByCategory,
		"topCategory":       topCategory,
		"averageExpense":    averageExpense,
		"insight":           insight,
		"dailyBreakdown":    dailyBreakdown,
	}
}
