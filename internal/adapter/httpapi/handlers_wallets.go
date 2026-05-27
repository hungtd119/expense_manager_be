package httpapi

import (
	"errors"
	"net/http"

	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/usecase"
)

func (s *server) CreateWallet(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	var body map[string]any
	if !decodeJSON(w, r, &body, requestID) {
		return
	}
	wallet, err := usecase.NewReferenceService(s.store).CreateWallet(user.ID, body)
	if errors.Is(err, domain.ErrInvalidInput) {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Vi khong hop le.", nil, requestID)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong ghi duoc du lieu.", nil, requestID)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"wallet": sanitizeWallet(wallet), "requestId": requestID})
}

func (s *server) WalletByID(w http.ResponseWriter, r *http.Request, db DB, id string, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	index := -1
	for i, w := range db.Wallets {
		if w.ID == id && w.UserID == user.ID {
			index = i
			break
		}
	}
	if index < 0 {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Vi khong ton tai.", nil, requestID)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var body map[string]any
		if !decodeJSON(w, r, &body, requestID) {
			return
		}
		updated, err := usecase.NewReferenceService(s.store).UpdateWallet(user.ID, id, body)
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Vi khong hop le.", nil, requestID)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong ghi duoc du lieu.", nil, requestID)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"wallet": sanitizeWallet(updated), "requestId": requestID})
	case http.MethodDelete:
		err := usecase.NewReferenceService(s.store).DeleteWallet(user.ID, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "STORE_ERROR", err.Error(), nil, requestID)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "requestId": requestID})
	default:
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Endpoint khong ton tai.", nil, requestID)
	}
}
