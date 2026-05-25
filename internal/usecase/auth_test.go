package usecase

import (
	"errors"
	"testing"

	"expense-manager-mvp/internal/domain"
)

func TestAuthRegisterLoginLogout(t *testing.T) {
	h := newTestHarness(t)
	auth := h.auth()

	result, err := auth.Register(RegisterInput{
		Name:     "Nguyen Van A",
		Email:    "user@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if result.User.Email != "user@example.com" || result.Token == "" {
		t.Fatalf("unexpected register result: %+v", result)
	}

	_, err = auth.Register(RegisterInput{
		Name:     "Other",
		Email:    "user@example.com",
		Password: "password123",
	})
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	login, err := auth.Login(LoginInput{Email: "user@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if login.User.ID != result.User.ID || login.Token == "" {
		t.Fatalf("unexpected login result: %+v", login)
	}

	if err := auth.Logout(login.Token); err != nil {
		t.Fatalf("logout: %v", err)
	}
}

func TestAuthRegisterValidation(t *testing.T) {
	h := newTestHarness(t)
	_, err := h.auth().Register(RegisterInput{Name: "", Email: "bad", Password: "short"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAuthLoginInvalidCredentials(t *testing.T) {
	h := newTestHarness(t)
	registerUser(t, h, "known@example.com")

	_, err := h.auth().Login(LoginInput{Email: "known@example.com", Password: "wrong-pass"})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}

	_, err = h.auth().Login(LoginInput{Email: "missing@example.com", Password: "password123"})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for missing user, got %v", err)
	}
}

func TestAuthPruneExpiredSessions(t *testing.T) {
	h := newTestHarness(t)
	if err := h.auth().PruneExpiredSessions(); err != nil {
		t.Fatalf("prune sessions: %v", err)
	}
}
