package httpapi

import (
	"errors"
	"net/http"

	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/usecase"
)

func (s *server) ListTransactions(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	bounds, ok := parseMonth(w, r, requestID)
	if !ok {
		return
	}
	transactions, err := usecase.NewTransactionService(s.store).List(user.ID, bounds)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong doc duoc giao dich.", nil, requestID)
		return
	}
	items := sanitizeTransactions(transactions, &db)
	filtered, filters, filterOK := filterTransactions(items, r)
	if !filterOK {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "type chi chap nhan income hoac expense.", nil, requestID)
		return
	}
	page, pageSize, pageOK := parsePagination(r)
	if !pageOK {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "page phai >= 1 va pageSize phai nam trong khoang 1-100.", nil, requestID)
		return
	}
	paged, meta := paginate(filtered, page, pageSize)
	meta["filters"] = filters
	writeJSON(w, http.StatusOK, map[string]any{"transactions": paged, "meta": meta, "requestId": requestID})
}

func (s *server) CreateTransaction(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	var body map[string]any
	if !decodeJSON(w, r, &body, requestID) {
		return
	}
	tx, err := usecase.NewTransactionService(s.store).Create(&db, user.ID, body)
	if errors.Is(err, domain.ErrInvalidInput) {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Giao dich khong hop le.", nil, requestID)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong ghi duoc du lieu.", nil, requestID)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"transaction": sanitizeTransaction(tx, &db), "requestId": requestID})
}

func (s *server) TransactionByID(w http.ResponseWriter, r *http.Request, db DB, id string, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	index := -1
	for i, tx := range db.Transactions {
		if tx.ID == id && tx.UserID == user.ID && tx.DeletedAt == nil {
			index = i
			break
		}
	}
	if index < 0 {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Giao dich khong ton tai.", nil, requestID)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var body map[string]any
		if !decodeJSON(w, r, &body, requestID) {
			return
		}
		current := db.Transactions[index]
		updated, err := usecase.NewTransactionService(s.store).Update(&db, user.ID, current, body)
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Giao dich khong hop le.", nil, requestID)
			return
		}
		if err != nil {
			writeStoreMutationError(w, err, requestID)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"transaction": sanitizeTransaction(updated, &db), "requestId": requestID})
	case http.MethodDelete:
		if err := usecase.NewTransactionService(s.store).Delete(user.ID, id); err != nil {
			writeStoreMutationError(w, err, requestID)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "requestId": requestID})
	default:
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Endpoint khong ton tai.", nil, requestID)
	}
}

func writeStoreMutationError(w http.ResponseWriter, err error, requestID string) {
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Giao dich khong ton tai.", nil, requestID)
		return
	}
	writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong ghi duoc du lieu.", nil, requestID)
}
