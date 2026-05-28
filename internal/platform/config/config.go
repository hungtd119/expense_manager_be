package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config gom cau hinh runtime cho Go backend.
type Config struct {
	Port                  string
	StoreDriver           string
	DataFile              string
	SQLiteFile            string
	SQLiteImportJSON      string
	MySQLDSN              string
	MySQLImportJSON       string
	PublicDir             string
	ShutdownTimeout       time.Duration
	ReadHeaderTimeout     time.Duration
	CORSAllowedOrigins    []string
	AuthRatePerMinute     int
	AuthRateBurst         int
	PasswordMinLength     int
	PasswordRequireLetter bool
	PasswordRequireDigit  bool
}

// Load doc cau hinh tu bien moi truong.
func Load() (Config, error) {
	cfg := Config{
		Port:                  env("PORT", "3000"),
		StoreDriver:           strings.ToLower(env("STORE_DRIVER", env("STORAGE_DRIVER", "mysql"))),
		DataFile:              env("DATA_FILE", filepath.Join("data", "go-app.db.json")),
		SQLiteFile:            env("SQLITE_FILE", filepath.Join("data", "go-app.sqlite")),
		SQLiteImportJSON:      env("SQLITE_IMPORT_JSON", env("DATA_FILE", filepath.Join("data", "go-app.db.json"))),
		MySQLDSN:              env("MYSQL_DSN", "expense:expense@tcp(127.0.0.1:3306)/expense_manager?charset=utf8mb4&parseTime=false&loc=Local"),
		MySQLImportJSON:       env("MYSQL_IMPORT_JSON", env("DATA_FILE", filepath.Join("data", "go-app.db.json"))),
		PublicDir:             env("PUBLIC_DIR", filepath.Join("..", "frontend", "public")),
		ShutdownTimeout:       durationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadHeaderTimeout:     durationEnv("READ_HEADER_TIMEOUT", 5*time.Second),
		CORSAllowedOrigins:    splitCSV(env("CORS_ORIGINS", "*")),
		AuthRatePerMinute:     intEnv("AUTH_RATE_PER_MINUTE", 30),
		AuthRateBurst:         intEnv("AUTH_RATE_BURST", 10),
		PasswordMinLength:     intEnv("PASSWORD_MIN_LENGTH", 8),
		PasswordRequireLetter: boolEnv("PASSWORD_REQUIRE_LETTER", true),
		PasswordRequireDigit:  boolEnv("PASSWORD_REQUIRE_DIGIT", true),
	}
	return cfg, cfg.Validate()
}

// TestDefaults cau hinh thoai mai cho unit/integration test.
func TestDefaults() Config {
	return Config{
		Port:                  "3000",
		StoreDriver:           "mysql",
		DataFile:              filepath.Join("data", "go-app.db.json"),
		SQLiteFile:            filepath.Join("data", "go-app.sqlite"),
		SQLiteImportJSON:      filepath.Join("data", "go-app.db.json"),
		MySQLDSN:              "expense:expense@tcp(127.0.0.1:3306)/expense_manager?charset=utf8mb4&parseTime=false&loc=Local",
		MySQLImportJSON:       filepath.Join("data", "go-app.db.json"),
		PublicDir:             filepath.Join("..", "frontend", "public"),
		ShutdownTimeout:       2 * time.Second,
		ReadHeaderTimeout:     5 * time.Second,
		CORSAllowedOrigins:    []string{"http://localhost:3000"},
		AuthRatePerMinute:     1000,
		AuthRateBurst:         100,
		PasswordMinLength:     8,
		PasswordRequireLetter: true,
		PasswordRequireDigit:  true,
	}
}

func (c Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("PORT khong duoc rong")
	}
	if _, err := strconv.Atoi(c.Port); err != nil {
		return fmt.Errorf("PORT phai la so: %s", c.Port)
	}
	switch c.StoreDriver {
	case "json", "sqlite", "mysql":
	default:
		return fmt.Errorf("STORE_DRIVER khong ho tro: %s", c.StoreDriver)
	}
	if c.StoreDriver == "json" && c.DataFile == "" {
		return fmt.Errorf("DATA_FILE khong duoc rong khi dung json store")
	}
	if c.StoreDriver == "sqlite" && c.SQLiteFile == "" {
		return fmt.Errorf("SQLITE_FILE khong duoc rong khi dung sqlite store")
	}
	if c.StoreDriver == "mysql" && c.MySQLDSN == "" {
		return fmt.Errorf("MYSQL_DSN khong duoc rong khi dung mysql store")
	}
	if c.PublicDir == "" {
		return fmt.Errorf("PUBLIC_DIR khong duoc rong")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT phai > 0")
	}
	if c.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("READ_HEADER_TIMEOUT phai > 0")
	}
	if c.AuthRatePerMinute < 1 {
		return fmt.Errorf("AUTH_RATE_PER_MINUTE phai >= 1")
	}
	if c.AuthRateBurst < 1 {
		return fmt.Errorf("AUTH_RATE_BURST phai >= 1")
	}
	if c.PasswordMinLength < 8 {
		return fmt.Errorf("PASSWORD_MIN_LENGTH phai >= 8")
	}
	return nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func boolEnv(key string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	return raw == "1" || raw == "true" || raw == "yes"
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
