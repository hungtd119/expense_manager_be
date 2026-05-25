package httpapi

import (
	"errors"
	"net/http"

	"expense-manager-mvp/internal/domain"
	"expense-manager-mvp/internal/usecase"
)

func (s *server) Register(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	var body map[string]any
	if !decodeJSON(w, r, &body, requestID) {
		return
	}
	result, err := s.authService().Register(usecase.RegisterInput{
		Name:     str(body["name"]),
		Email:    str(body["email"]),
		Password: str(body["password"]),
	})
	if errors.Is(err, domain.ErrInvalidInput) {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Vui long nhap ten, email hop le va mat khau toi thieu 8 ki tu (gom chu va so).", map[string]any{"fields": []string{"name", "email", "password"}}, requestID)
		return
	}
	if errors.Is(err, domain.ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "EMAIL_EXISTS", "Email da ton tai.", map[string]any{"field": "email"}, requestID)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong ghi duoc du lieu.", nil, requestID)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": result.Token, "user": sanitizeUser(result.User), "requestId": requestID})
}

func (s *server) Login(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	var body map[string]any
	if !decodeJSON(w, r, &body, requestID) {
		return
	}
	result, err := s.authService().Login(usecase.LoginInput{Email: str(body["email"]), Password: str(body["password"])})
	if errors.Is(err, domain.ErrUnauthorized) {
		writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email hoac mat khau khong dung.", nil, requestID)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORE_ERROR", "Khong doc duoc du lieu.", nil, requestID)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": result.Token, "user": sanitizeUser(result.User), "requestId": requestID})
}

func (s *server) Logout(w http.ResponseWriter, r *http.Request, db DB, requestID string) {
	_ = s.authService().Logout(bearerToken(r))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "requestId": requestID})
}
