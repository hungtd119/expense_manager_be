package httpapi

import (
	"errors"
	"net/http"

	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/usecase"
)

func (s *server) ListRecurring(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	recurringItems, result, err := usecase.NewRecurringService(s.store).List(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong doc duoc danh sach dinh ky.", nil, requestID)
		return
	}
	items := []map[string]any{}
	for _, recurring := range recurringItems {
		items = append(items, sanitizeRecurring(recurring, &db))
	}
	writeJSON(w, http.StatusOK, map[string]any{"recurringTransactions": items, "generatedCount": result.GeneratedCount, "meta": map[string]any{"total": len(items)}, "requestId": requestID})
}

func (s *server) CreateRecurring(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	var body map[string]any
	if !decodeJSON(w, r, &body, requestID) {
		return
	}
	recurring, result, err := usecase.NewRecurringService(s.store).Create(&db, user.ID, body)
	if errors.Is(err, domain.ErrInvalidInput) {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Khoan dinh ky khong hop le.", nil, requestID)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong ghi duoc du lieu.", nil, requestID)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"recurringTransaction": sanitizeRecurring(recurring, &db), "generatedCount": result.GeneratedCount, "requestId": requestID})
}

func (s *server) RecurringByID(w http.ResponseWriter, r *http.Request, db DB, id string, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	index := -1
	for i, recurring := range db.RecurringTransactions {
		if recurring.ID == id && recurring.UserID == user.ID && recurring.DeletedAt == nil {
			index = i
			break
		}
	}
	if index < 0 {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Khoan dinh ky khong ton tai.", nil, requestID)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var body map[string]any
		if !decodeJSON(w, r, &body, requestID) {
			return
		}
		current := db.RecurringTransactions[index]
		updated, result, err := usecase.NewRecurringService(s.store).Update(&db, user.ID, current, body)
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Khoan dinh ky khong hop le.", nil, requestID)
			return
		}
		if err != nil {
			writeRecurringMutationError(w, err, requestID)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"recurringTransaction": sanitizeRecurring(updated, &db), "generatedCount": result.GeneratedCount, "requestId": requestID})
	case http.MethodDelete:
		if err := usecase.NewRecurringService(s.store).Delete(user.ID, id); err != nil {
			writeRecurringMutationError(w, err, requestID)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "requestId": requestID})
	default:
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Endpoint khong ton tai.", nil, requestID)
	}
}

func writeRecurringMutationError(w http.ResponseWriter, err error, requestID string) {
	if err == errNotFound {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Khoan dinh ky khong ton tai.", nil, requestID)
		return
	}
	writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong ghi duoc du lieu.", nil, requestID)
}
