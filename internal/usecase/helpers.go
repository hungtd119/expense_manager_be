package usecase

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"math"
	"net/mail"
	"strings"
	"time"

	"expense-manager-mvp/internal/domain"
)

func hashPassword(password string, salt string) (string, string) {
	if salt == "" {
		salt = randomHex(16)
	}
	hashValue, err := pbkdf2.Key[hash.Hash](sha512.New, password, []byte(salt), 210000, 64)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(hashValue), salt
}

func randomHex(size int) string {
	bytes := make([]byte, size)
	if _, err := randRead(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

var randRead = func(bytes []byte) (int, error) {
	return rand.Read(bytes)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil && !strings.Contains(email, " ")
}

func sha256Hex(value string) string {
	hashValue := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hashValue[:])
}

func walletExists(db *domain.DB, userID string, walletID string) bool {
	for _, wallet := range db.Wallets {
		if wallet.ID == walletID && wallet.UserID == userID {
			return true
		}
	}
	return false
}

func categoryValid(db *domain.DB, userID string, categoryID string, typeValue string) bool {
	for _, category := range db.Categories {
		if category.ID == categoryID && category.Type == typeValue && (category.UserID == nil || *category.UserID == userID) {
			return true
		}
	}
	return false
}

func isFinitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func localDateTimeNow() string {
	now := time.Now()
	return now.Format("2006-01-02T15:04")
}
