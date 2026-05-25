package httpapi

import "expense-manager-mvp/internal/domain"

type DB = domain.DB
type User = domain.User
type Session = domain.Session
type Wallet = domain.Wallet
type Category = domain.Category
type Transaction = domain.Transaction
type Budget = domain.Budget
type RecurringTransaction = domain.RecurringTransaction

type APIError struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details"`
	RequestID string `json:"requestId"`
}

type MonthBounds = domain.MonthBounds
type RecurringResult = domain.RecurringResult
