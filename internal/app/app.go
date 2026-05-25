package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"expense-manager-mvp/internal/adapter/store/jsonstore"
	"expense-manager-mvp/internal/adapter/store/mysqlstore"
	"expense-manager-mvp/internal/adapter/store/sqlitestore"
	"expense-manager-mvp/internal/platform/config"
	"expense-manager-mvp/internal/store"
)

// HandlerFactory tao http.Handler tu store va config (Gin adapter o httpapi).
type HandlerFactory func(store.Store, config.Config) http.Handler

func Run(newHandler HandlerFactory) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	st, err := newStore(cfg)
	if err != nil {
		return err
	}
	if err := st.Ensure(); err != nil {
		return err
	}

	addr := ":" + cfg.Port
	handler := newHandler(st, cfg)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("Expense Manager Go backend dang chay tai http://localhost%s (driver=%s)", cfg.Port, cfg.StoreDriver)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		log.Printf("Nhan tin hieu %s, dang shutdown...", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Printf("Server da shutdown.")
	return nil
}

func newStore(cfg config.Config) (store.Store, error) {
	switch cfg.StoreDriver {
	case "json":
		return jsonstore.New(cfg.DataFile), nil
	case "sqlite":
		return sqlitestore.NewSQLiteStore(cfg.SQLiteFile, cfg.SQLiteImportJSON), nil
	case "mysql":
		return mysqlstore.NewMySQLStore(cfg.MySQLDSN, cfg.MySQLImportJSON), nil
	default:
		return nil, fmt.Errorf("storage driver khong ho tro: %s", cfg.StoreDriver)
	}
}
