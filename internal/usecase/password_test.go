package usecase

import (
	"errors"
	"testing"

	"expense-manager-mvp/internal/domain"
)

func TestPasswordPolicy(t *testing.T) {
	policy := DefaultPasswordPolicy()
	if err := policy.Validate("password123"); err != nil {
		t.Fatalf("expected valid password: %v", err)
	}
	if err := policy.Validate("short1"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid short password: %v", err)
	}
	if err := policy.Validate("12345678"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid without letter: %v", err)
	}
	if err := policy.Validate("abcdefgh"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid without digit: %v", err)
	}
}
