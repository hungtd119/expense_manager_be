package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expense-manager-mvp/internal/platform/config"
	"expense-manager-mvp/internal/store/memstore"
)

func TestHealth(t *testing.T) {
	handler := NewHandler(memstore.New(), config.TestDefaults())
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["ok"] != true || body["storageDriver"] != "memory" {
		t.Fatalf("unexpected health: %+v", body)
	}
	if body["requestId"] == "" {
		t.Fatal("missing requestId")
	}
}

func TestAuthRegisterValidation(t *testing.T) {
	handler := NewHandler(memstore.New(), config.TestDefaults())
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"name":"A","email":"bad","password":"password123"}`))
	req.Header.Set("content-type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["code"] != "VALIDATION_ERROR" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestAPISmokeFlow(t *testing.T) {
	st := memstore.New()
	if err := st.Ensure(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	handler := NewHandler(st, config.TestDefaults())

	email := "httpapi-test@example.com"
	token := register(t, handler, email)

	month := time.Now().Format("2006-01")
	categories := apiGet(t, handler, "/api/categories", token)
	categoryList, _ := categories["categories"].([]any)
	var expenseCatID, incomeCatID string
	for _, item := range categoryList {
		cat, _ := item.(map[string]any)
		if cat["type"] == "expense" {
			expenseCatID, _ = cat["id"].(string)
		}
		if cat["type"] == "income" {
			incomeCatID, _ = cat["id"].(string)
		}
	}
	wallets := apiGet(t, handler, "/api/wallets", token)
	walletList, _ := wallets["wallets"].([]any)
	wallet0, _ := walletList[0].(map[string]any)
	walletID, _ := wallet0["id"].(string)

	today := time.Now().Format("2006-01-02")
	apiPost(t, handler, "/api/transactions", token, map[string]any{
		"walletId": walletID, "categoryId": incomeCatID, "type": "income",
		"amount": 1000000, "note": "Test income", "transactionDate": today,
	})
	expense := apiPost(t, handler, "/api/transactions", token, map[string]any{
		"walletId": walletID, "categoryId": expenseCatID, "type": "expense",
		"amount": 250000, "note": "Test expense", "transactionDate": today,
	})
	expenseTx, _ := expense["transaction"].(map[string]any)
	expenseTxID, _ := expenseTx["id"].(string)

	dashboard := apiGet(t, handler, "/api/dashboard?month="+month, token)
	dash, _ := dashboard["dashboard"].(map[string]any)
	totals, _ := dash["totals"].(map[string]any)
	if totals["income"].(float64) != 1000000 {
		t.Fatalf("dashboard income: %+v", totals)
	}

	budget := apiPost(t, handler, "/api/budgets?month="+month, token, map[string]any{
		"categoryId": expenseCatID, "amountLimit": 300000,
	})
	budgetObj, _ := budget["budget"].(map[string]any)
	if budgetObj["alertLevel"] != "warning" {
		t.Fatalf("expected budget warning, got %+v", budgetObj)
	}

	pastRun := time.Now().Add(-1 * time.Minute).Format("2006-01-02T15:04")
	recurring := apiPost(t, handler, "/api/recurring-transactions", token, map[string]any{
		"walletId": walletID, "categoryId": expenseCatID, "type": "expense", "amount": 50000,
		"note": "Test recurring", "frequency": "daily", "nextRunAt": pastRun, "active": true,
	})
	if recurring["generatedCount"].(float64) < 1 {
		t.Fatalf("expected generated recurring tx: %+v", recurring)
	}

	apiRequest(t, handler, http.MethodDelete, "/api/transactions/"+expenseTxID, token, nil)
	list := apiGet(t, handler, "/api/transactions?month="+month, token)
	txList, _ := list["transactions"].([]any)
	for _, item := range txList {
		tx, _ := item.(map[string]any)
		if tx["id"] == expenseTxID {
			t.Fatal("deleted transaction still visible")
		}
	}
}

func register(t *testing.T, handler http.Handler, email string) string {
	t.Helper()
	body := apiPost(t, handler, "/api/auth/register", "", map[string]any{
		"name": "HTTP API Tester", "email": email, "password": "password123",
	})
	token, _ := body["token"].(string)
	if token == "" {
		t.Fatal("register missing token")
	}
	return token
}

func apiGet(t *testing.T, handler http.Handler, path string, token string) map[string]any {
	t.Helper()
	return apiRequest(t, handler, http.MethodGet, path, token, nil)
}

func apiPost(t *testing.T, handler http.Handler, path string, token string, payload map[string]any) map[string]any {
	t.Helper()
	return apiRequest(t, handler, http.MethodPost, path, token, payload)
}

func apiRequest(t *testing.T, handler http.Handler, method string, path string, token string, payload map[string]any) map[string]any {
	t.Helper()
	var bodyReader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		bodyReader = strings.NewReader(string(raw))
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if payload != nil {
		req.Header.Set("content-type", "application/json")
	}
	if token != "" {
		req.Header.Set("authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("%s %s status %d body %s", method, path, rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s %s: %v", method, path, err)
	}
	return body
}
