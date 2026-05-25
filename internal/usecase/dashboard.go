package usecase

import (
	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/store"
)

type DashboardService struct {
	store store.Store
}

func NewDashboardService(store store.Store) DashboardService {
	return DashboardService{store: store}
}

func (s DashboardService) TransactionsForDashboard(userID string, bounds domain.MonthBounds) ([]domain.Transaction, error) {
	if _, err := s.store.ProcessDueRecurring(userID, localDateTimeNow()); err != nil {
		return nil, err
	}
	return s.store.ListTransactionsForMonth(userID, bounds)
}
