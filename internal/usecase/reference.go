package usecase

import "expense-manager-mvp/internal/domain"

type ReferenceService struct{}

func NewReferenceService() ReferenceService {
	return ReferenceService{}
}

func (ReferenceService) CategoriesForUser(db *domain.DB, userID string) []domain.Category {
	result := []domain.Category{}
	for _, category := range db.Categories {
		if category.UserID == nil || *category.UserID == userID {
			result = append(result, category)
		}
	}
	return result
}

func (ReferenceService) WalletsForUser(db *domain.DB, userID string) []domain.Wallet {
	result := []domain.Wallet{}
	for _, wallet := range db.Wallets {
		if wallet.UserID == userID {
			result = append(result, wallet)
		}
	}
	return result
}
