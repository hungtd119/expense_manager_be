package usecase

import (
	"strings"
	"unicode"

	"expense-manager-mvp/internal/domain"
)

// PasswordPolicy mo ta quy tac mat khau khi dang ky.
type PasswordPolicy struct {
	MinLength      int
	RequireLetter  bool
	RequireDigit   bool
}

func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{MinLength: 8, RequireLetter: true, RequireDigit: true}
}

func PasswordPolicyFromConfig(minLength int, requireLetter, requireDigit bool) PasswordPolicy {
	return PasswordPolicy{
		MinLength:     minLength,
		RequireLetter: requireLetter,
		RequireDigit:  requireDigit,
	}
}

func (p PasswordPolicy) Validate(password string) error {
	if len(password) < p.MinLength {
		return domain.ErrInvalidInput
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if p.RequireLetter && !hasLetter {
		return domain.ErrInvalidInput
	}
	if p.RequireDigit && !hasDigit {
		return domain.ErrInvalidInput
	}
	if strings.TrimSpace(password) != password {
		return domain.ErrInvalidInput
	}
	return nil
}
