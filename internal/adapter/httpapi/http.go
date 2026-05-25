package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, target any, requestID string) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			writeError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Body JSON vuot qua gioi han 1MB.", nil, requestID)
			return false
		}
		if errors.Is(err, io.EOF) {
			return true
		}
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Body JSON khong hop le.", nil, requestID)
		return false
	}
	return true
}

// WriteJSON ghi JSON response theo API contract.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.Header().Set("cache-control", "no-store")
	if asMap, ok := body.(map[string]any); ok {
		if requestID, ok := asMap["requestId"].(string); ok {
			w.Header().Set("x-request-id", requestID)
		}
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// WriteError ghi JSON error response theo API contract.
func WriteError(w http.ResponseWriter, status int, code string, message string, details any, requestID string) {
	WriteJSON(w, status, APIError{OK: false, Error: message, Code: code, Message: message, Details: details, RequestID: requestID})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	WriteJSON(w, status, body)
}

func writeError(w http.ResponseWriter, status int, code string, message string, details any, requestID string) {
	WriteError(w, status, code, message, details, requestID)
}
