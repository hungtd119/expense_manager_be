package httpapi

import (
	"net/http"

	"expense-manager-mvp/internal/usecase"
)

func (s *server) Dashboard(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	user, ok := requireUser(w, r, &db, requestID)
	if !ok {
		return
	}
	bounds, ok := parseMonth(w, r, requestID)
	if !ok {
		return
	}
	transactions, err := usecase.NewDashboardService(s.store).TransactionsForDashboard(user.ID, bounds)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong doc duoc giao dich.", nil, requestID)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"month": r.URL.Query().Get("month"), "dashboard": dashboardSummary(&db, transactions), "requestId": requestID})
}
