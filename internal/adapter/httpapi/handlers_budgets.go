package httpapi

import (
	"errors"
	"net/http"

	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/usecase"
)

func (s *server) ListBudgets(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	bounds, ok := parseMonth(w, r, requestID)
	if !ok {
		return
	}
	budgetItems, transactions, err := usecase.NewBudgetService(s.store).List(user.ID, bounds)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong doc duoc ngan sach.", nil, requestID)
		return
	}
	budgets := []map[string]any{}
	for _, budget := range budgetItems {
		budgets = append(budgets, sanitizeBudget(budget, &db, transactions))
	}
	writeJSON(w, http.StatusOK, map[string]any{"budgets": budgets, "meta": map[string]any{"total": len(budgets)}, "requestId": requestID})
}

func (s *server) CreateBudget(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	bounds, ok := parseMonth(w, r, requestID)
	if !ok {
		return
	}
	var body map[string]any
	if !decodeJSON(w, r, &body, requestID) {
		return
	}
	budget, transactions, err := usecase.NewBudgetService(s.store).Create(&db, user.ID, bounds, body)
	if errors.Is(err, domain.ErrInvalidInput) {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Ngan sach khong hop le.", nil, requestID)
		return
	}
	if err != nil {
		writeBudgetMutationError(w, err, requestID)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"budget": sanitizeBudget(budget, &db, transactions), "requestId": requestID})
}

func (s *server) BudgetByID(w http.ResponseWriter, r *http.Request, db DB, id string, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	index := -1
	for i, budget := range db.Budgets {
		if budget.ID == id && budget.UserID == user.ID && budget.DeletedAt == nil {
			index = i
			break
		}
	}
	if index < 0 {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Ngan sach khong ton tai.", nil, requestID)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var body map[string]any
		if !decodeJSON(w, r, &body, requestID) {
			return
		}
		current := db.Budgets[index]
		updated, transactions, err := usecase.NewBudgetService(s.store).Update(&db, user.ID, current, body)
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Ngan sach khong hop le.", nil, requestID)
			return
		}
		if err != nil {
			writeBudgetMutationError(w, err, requestID)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"budget": sanitizeBudget(updated, &db, transactions), "requestId": requestID})
	case http.MethodDelete:
		if err := usecase.NewBudgetService(s.store).Delete(user.ID, id); err != nil {
			writeBudgetMutationError(w, err, requestID)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "requestId": requestID})
	default:
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Endpoint khong ton tai.", nil, requestID)
	}
}

func writeBudgetMutationError(w http.ResponseWriter, err error, requestID string) {
	if errors.Is(err, errAlreadyExists) {
		writeError(w, http.StatusConflict, "BUDGET_EXISTS", "Danh muc nay da co ngan sach trong thang.", nil, requestID)
		return
	}
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Ngan sach khong ton tai.", nil, requestID)
		return
	}
	writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong ghi duoc du lieu.", nil, requestID)
}
