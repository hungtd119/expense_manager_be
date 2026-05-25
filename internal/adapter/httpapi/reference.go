package httpapi

func userCategories(db *DB, userID string) []map[string]any {
	var result []map[string]any
	for _, category := range db.Categories {
		if category.UserID == nil || *category.UserID == userID {
			result = append(result, sanitizeCategory(category))
		}
	}
	return result
}

func sanitizeCategories(categories []Category) []map[string]any {
	result := make([]map[string]any, 0, len(categories))
	for _, category := range categories {
		result = append(result, sanitizeCategory(category))
	}
	return result
}

func sanitizeCategory(category Category) map[string]any {
	return map[string]any{"id": category.ID, "userId": category.UserID, "name": category.Name, "type": category.Type, "icon": category.Icon, "color": category.Color, "isDefault": category.IsDefault}
}

func userWallets(db *DB, userID string) []map[string]any {
	var result []map[string]any
	for _, wallet := range db.Wallets {
		if wallet.UserID == userID {
			result = append(result, sanitizeWallet(wallet))
		}
	}
	return result
}

func sanitizeWallets(wallets []Wallet) []map[string]any {
	result := make([]map[string]any, 0, len(wallets))
	for _, wallet := range wallets {
		result = append(result, sanitizeWallet(wallet))
	}
	return result
}

func sanitizeWallet(wallet Wallet) map[string]any {
	return map[string]any{"id": wallet.ID, "userId": wallet.UserID, "name": wallet.Name, "currency": wallet.Currency, "balanceInitial": wallet.BalanceInitial, "createdAt": wallet.CreatedAt}
}

func categoryByID(db *DB, id string) Category {
	for _, category := range db.Categories {
		if category.ID == id {
			return category
		}
	}
	return Category{Name: "Khong ro", Color: "#657084"}
}

func walletByID(db *DB, id string) Wallet {
	for _, wallet := range db.Wallets {
		if wallet.ID == id {
			return wallet
		}
	}
	return Wallet{Name: ""}
}

func walletExists(db *DB, userID string, walletID string) bool {
	for _, wallet := range db.Wallets {
		if wallet.ID == walletID && wallet.UserID == userID {
			return true
		}
	}
	return false
}

func categoryValid(db *DB, userID string, categoryID string, typeValue string) bool {
	for _, category := range db.Categories {
		if category.ID == categoryID && category.Type == typeValue && (category.UserID == nil || *category.UserID == userID) {
			return true
		}
	}
	return false
}
