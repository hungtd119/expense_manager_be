package httpapi

import (
	"net/http"

	"expense-manager-mvp/internal/usecase"
)

func (s *server) Me(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := currentUser(&db, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Chua dang nhap.", nil, requestID)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": sanitizeUser(user), "requestId": requestID})
}

func (s *server) Categories(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	categories := usecase.NewReferenceService().CategoriesForUser(&db, user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"categories": sanitizeCategories(categories), "requestId": requestID})
}

func (s *server) Wallets(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	wallets := usecase.NewReferenceService().WalletsForUser(&db, user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"wallets": sanitizeWallets(wallets), "requestId": requestID})
}
