package usecase

import (
	"errors"
	"strings"
	"time"

	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/platform"
	"expense-manager-mvp/internal/store"
)

type AuthService struct {
	store          store.Store
	clock          platform.Clock
	ids            platform.IDGenerator
	passwordPolicy PasswordPolicy
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	Token string
	User  domain.User
}

func NewAuthService(store store.Store) AuthService {
	return NewAuthServiceWithPolicy(store, DefaultPasswordPolicy())
}

func NewAuthServiceWithPolicy(store store.Store, policy PasswordPolicy) AuthService {
	return AuthService{
		store:          store,
		clock:          platform.SystemClock{},
		ids:            platform.CryptoIDGenerator{},
		passwordPolicy: policy,
	}
}

func (s AuthService) Register(input RegisterInput) (AuthResult, error) {
	email := normalizeEmail(input.Email)
	name := strings.TrimSpace(input.Name)
	password := input.Password
	if name == "" || email == "" {
		return AuthResult{}, domain.ErrInvalidInput
	}
	if !validEmail(email) {
		return AuthResult{}, domain.ErrInvalidInput
	}
	if err := s.passwordPolicy.Validate(password); err != nil {
		return AuthResult{}, err
	}
	if existing, err := s.store.FindUserByEmail(email); err == nil && existing.ID != "" {
		return AuthResult{}, domain.ErrAlreadyExists
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return AuthResult{}, err
	}

	now := platform.NowISO(s.clock)
	hashValue, salt := hashPassword(password, "")
	user := domain.User{ID: s.ids.UUID(), Email: email, Name: name, PasswordHash: hashValue, PasswordSalt: salt, CreatedAt: now, UpdatedAt: now}
	wallet := domain.Wallet{ID: s.ids.UUID(), UserID: user.ID, Name: "Vi chinh", Currency: "VND", BalanceInitial: 0, CreatedAt: now}
	token, session := s.buildSession(user.ID)
	if err := s.store.CreateUserWithWalletAndSession(user, wallet, session); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Token: token, User: user}, nil
}

func (s AuthService) Login(input LoginInput) (AuthResult, error) {
	email := normalizeEmail(input.Email)
	user, err := s.store.FindUserByEmail(email)
	if errors.Is(err, domain.ErrNotFound) {
		return AuthResult{}, domain.ErrUnauthorized
	}
	if err != nil {
		return AuthResult{}, err
	}
	hashValue, _ := hashPassword(input.Password, user.PasswordSalt)
	if hashValue != user.PasswordHash {
		return AuthResult{}, domain.ErrUnauthorized
	}
	token, session := s.buildSession(user.ID)
	if err := s.store.CreateSession(session); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Token: token, User: user}, nil
}

func (s AuthService) Logout(token string) error {
	if token == "" {
		return nil
	}
	return s.store.DeleteSession(sha256Hex(token))
}

func (s AuthService) PruneExpiredSessions() error {
	_, err := s.store.DeleteExpiredSessions(s.clock.Now())
	return err
}

func (s AuthService) buildSession(userID string) (string, domain.Session) {
	token := s.ids.TokenHex(32)
	now := s.clock.Now().UTC()
	return token, domain.Session{
		TokenHash: sha256Hex(token),
		UserID:    userID,
		CreatedAt: now.Format(time.RFC3339Nano),
		ExpiresAt: now.Add(30 * 24 * time.Hour).Format(time.RFC3339Nano),
	}
}
