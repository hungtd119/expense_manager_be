package httpapi

import (
	"math"
	"sort"
)

func budgetsForMonth(db *DB, userID string, bounds MonthBounds) []Budget {
	result := []Budget{}
	for _, budget := range db.Budgets {
		if budget.UserID == userID && budget.DeletedAt == nil && budget.Period == "monthly" && budget.StartDate == bounds.StartDate {
			result = append(result, budget)
		}
	}
	sortBudgets(result)
	return result
}

func sortBudgets(items []Budget) {
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt < items[j].CreatedAt })
}

func validateBudget(db *DB, userID string, body map[string]any, bounds MonthBounds, existing *Budget) (map[string]any, string) {
	categoryID := valueOr(body, "categoryId", existingBudgetString(existing, "categoryId"))
	amountLimit := numberOr(body, "amountLimit", existingBudgetAmount(existing))
	if !categoryValid(db, userID, categoryID, "expense") {
		return nil, "Danh muc ngan sach phai la danh muc chi."
	}
	if amountLimit <= 0 || math.IsNaN(amountLimit) || math.IsInf(amountLimit, 0) {
		return nil, "Han muc ngan sach phai lon hon 0."
	}
	return map[string]any{"categoryId": categoryID, "amountLimit": amountLimit, "period": "monthly", "startDate": bounds.StartDate, "endDate": bounds.EndDate}, ""
}

func sanitizeBudget(budget Budget, db *DB, transactions []Transaction) map[string]any {
	category := categoryByID(db, budget.CategoryID)
	spent := 0.0
	for _, tx := range transactions {
		if tx.Type == "expense" && tx.CategoryID == budget.CategoryID && tx.DeletedAt == nil && tx.TransactionDate >= budget.StartDate && tx.TransactionDate < budget.EndDate {
			spent += tx.Amount
		}
	}
	percentUsed := 0.0
	if budget.AmountLimit > 0 {
		percentUsed = round1(spent / budget.AmountLimit * 100)
	}
	remaining := budget.AmountLimit - spent
	alertLevel := "ok"
	alertMessage := "Dang trong gioi han."
	if percentUsed >= 100 {
		alertLevel = "exceeded"
		alertMessage = "Da vuot ngan sach."
	} else if percentUsed >= 80 {
		alertLevel = "warning"
		alertMessage = "Sap cham nguong ngan sach."
	}
	return map[string]any{"id": budget.ID, "userId": budget.UserID, "categoryId": budget.CategoryID, "categoryName": category.Name, "categoryColor": category.Color, "amountLimit": budget.AmountLimit, "period": budget.Period, "startDate": budget.StartDate, "endDate": budget.EndDate, "spent": spent, "remaining": remaining, "percentUsed": percentUsed, "alertLevel": alertLevel, "alertMessage": alertMessage, "createdAt": budget.CreatedAt, "updatedAt": budget.UpdatedAt, "deletedAt": budget.DeletedAt}
}
