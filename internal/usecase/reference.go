package usecase

import (
	"strings"

	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/platform"
	"expense-manager-mvp/internal/store"
)

type ReferenceService struct {
	store store.Store
	clock platform.Clock
	ids   platform.IDGenerator
}

func NewReferenceService(store store.Store) ReferenceService {
	return ReferenceService{
		store: store,
		clock: platform.SystemClock{},
		ids:   platform.CryptoIDGenerator{},
	}
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

func (s ReferenceService) CreateWallet(userID string, body map[string]any) (domain.Wallet, error) {
	name, _ := body["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Wallet{}, domain.ErrInvalidInput
	}
	currency, _ := body["currency"].(string)
	currency = strings.TrimSpace(currency)
	if currency == "" {
		currency = "VND"
	}
	balanceInitial, _ := body["balanceInitial"].(float64)

	now := platform.NowISO(s.clock)
	wallet := domain.Wallet{
		ID:             s.ids.UUID(),
		UserID:         userID,
		Name:           name,
		Currency:       currency,
		BalanceInitial: balanceInitial,
		CreatedAt:      now,
	}
	return wallet, s.store.CreateWallet(wallet)
}

func (s ReferenceService) UpdateWallet(userID string, id string, body map[string]any) (domain.Wallet, error) {
	name, _ := body["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Wallet{}, domain.ErrInvalidInput
	}
	currency, _ := body["currency"].(string)
	currency = strings.TrimSpace(currency)
	if currency == "" {
		currency = "VND"
	}
	balanceInitial, _ := body["balanceInitial"].(float64)

	wallet := domain.Wallet{
		ID:             id,
		UserID:         userID,
		Name:           name,
		Currency:       currency,
		BalanceInitial: balanceInitial,
	}
	return wallet, s.store.UpdateWallet(wallet)
}

func (s ReferenceService) DeleteWallet(userID string, id string) error {
	return s.store.DeleteWallet(userID, id)
}

func (s ReferenceService) CreateCategory(userID string, body map[string]any) (domain.Category, error) {
	name, _ := body["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Category{}, domain.ErrInvalidInput
	}
	typeVal, _ := body["type"].(string)
	if typeVal != "income" && typeVal != "expense" {
		return domain.Category{}, domain.ErrInvalidInput
	}
	icon, _ := body["icon"].(string)
	icon = strings.TrimSpace(icon)
	color, _ := body["color"].(string)
	color = strings.TrimSpace(color)

	category := domain.Category{
		ID:        s.ids.UUID(),
		UserID:    &userID,
		Name:      name,
		Type:      typeVal,
		Icon:      icon,
		Color:     color,
		IsDefault: false,
	}
	return category, s.store.CreateCategory(category)
}

func (s ReferenceService) UpdateCategory(userID string, id string, body map[string]any) (domain.Category, error) {
	name, _ := body["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Category{}, domain.ErrInvalidInput
	}
	typeVal, _ := body["type"].(string)
	if typeVal != "income" && typeVal != "expense" {
		return domain.Category{}, domain.ErrInvalidInput
	}
	icon, _ := body["icon"].(string)
	icon = strings.TrimSpace(icon)
	color, _ := body["color"].(string)
	color = strings.TrimSpace(color)

	category := domain.Category{
		ID:        id,
		UserID:    &userID,
		Name:      name,
		Type:      typeVal,
		Icon:      icon,
		Color:     color,
		IsDefault: false,
	}
	return category, s.store.UpdateCategory(category)
}

func (s ReferenceService) DeleteCategory(userID string, id string) error {
	return s.store.DeleteCategory(userID, id)
}
