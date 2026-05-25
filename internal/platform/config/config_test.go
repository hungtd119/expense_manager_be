package config

import (
	"os"
	"testing"
)

func TestLoadValidateDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("STORE_DRIVER", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Port != "3000" || cfg.StoreDriver != "sqlite" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestValidateRejectsUnknownDriver(t *testing.T) {
	cfg := TestDefaults()
	cfg.StoreDriver = "postgres"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateRejectsShortPasswordMin(t *testing.T) {
	cfg := TestDefaults()
	cfg.PasswordMinLength = 4
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadRespectsEnv(t *testing.T) {
	t.Setenv("PORT", "4000")
	t.Setenv("STORE_DRIVER", "sqlite")
	t.Setenv("SQLITE_FILE", "/tmp/test.sqlite")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Port != "4000" || cfg.StoreDriver != "sqlite" || cfg.SQLiteFile != "/tmp/test.sqlite" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	_ = os.Remove("/tmp/test.sqlite")
}
