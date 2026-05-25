package httpapi

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"expense-manager-mvp/internal/platform"
	"expense-manager-mvp/internal/platform/config"
	"expense-manager-mvp/internal/store"
	"expense-manager-mvp/internal/usecase"
)

const (
	ctxRequestID = "requestId"
	ctxDB        = "db"
)

type server struct {
	store store.Store
	cfg   config.Config
	ids   platform.IDGenerator
}

// NewHandler tao HTTP handler Gin voi static frontend va API contract hien tai.
func NewHandler(store store.Store, cfg config.Config) http.Handler {
	gin.SetMode(gin.ReleaseMode)

	s := &server{
		store: store,
		cfg:   cfg,
		ids:   platform.CryptoIDGenerator{},
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware(cfg.CORSAllowedOrigins))
	r.Use(requestLogger())

	api := r.Group("/api")
	api.Use(s.prepareContext)
	api.Use(authRateLimitMiddleware(cfg))
	{
		api.GET("/health", s.health)
		api.POST("/auth/register", s.register)
		api.POST("/auth/login", s.login)
		api.POST("/auth/logout", s.logout)
		api.GET("/me", s.me)
		api.GET("/categories", s.categories)
		api.GET("/wallets", s.wallets)
		api.GET("/transactions", s.listTransactions)
		api.POST("/transactions", s.createTransaction)
		api.PUT("/transactions/:id", s.transactionByID)
		api.DELETE("/transactions/:id", s.transactionByID)
		api.GET("/dashboard", s.dashboard)
		api.GET("/budgets", s.listBudgets)
		api.POST("/budgets", s.createBudget)
		api.PUT("/budgets/:id", s.budgetByID)
		api.DELETE("/budgets/:id", s.budgetByID)
		api.GET("/recurring-transactions", s.listRecurring)
		api.POST("/recurring-transactions", s.createRecurring)
		api.PUT("/recurring-transactions/:id", s.recurringByID)
		api.DELETE("/recurring-transactions/:id", s.recurringByID)
	}

	public := cfg.PublicDir
	r.GET("/", func(c *gin.Context) { c.File(filepath.Join(public, "index.html")) })
	r.GET("/app.js", func(c *gin.Context) { c.File(filepath.Join(public, "app.js")) })
	r.GET("/styles.css", func(c *gin.Context) { c.File(filepath.Join(public, "styles.css")) })
	r.NoRoute(s.notFound)
	return r
}

func (s *server) authService() usecase.AuthService {
	return usecase.NewAuthServiceWithPolicy(
		s.store,
		usecase.PasswordPolicyFromConfig(
			s.cfg.PasswordMinLength,
			s.cfg.PasswordRequireLetter,
			s.cfg.PasswordRequireDigit,
		),
	)
}

func (s *server) prepareContext(c *gin.Context) {
	requestID := s.ids.UUID()
	c.Set(ctxRequestID, requestID)

	db, err := s.store.Read()
	if err != nil {
		writeError(c.Writer, http.StatusInternalServerError, "STORE_ERROR", "Khong doc duoc du lieu.", nil, requestID)
		c.Abort()
		return
	}
	_ = s.authService().PruneExpiredSessions()
	c.Set(ctxDB, db)
	c.Next()
}

func (s *server) health(c *gin.Context) {
	requestID := requestID(c)
	writeJSON(c.Writer, http.StatusOK, map[string]any{
		"ok":            true,
		"now":           time.Now().UTC().Format(time.RFC3339Nano),
		"storageDriver": s.store.Driver(),
		"storage":       s.store.Location(),
		"requestId":     requestID,
	})
}

func (s *server) register(c *gin.Context) {
	s.Register(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) login(c *gin.Context) {
	s.Login(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) logout(c *gin.Context) {
	s.Logout(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) me(c *gin.Context) {
	s.Me(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) categories(c *gin.Context) {
	s.Categories(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) wallets(c *gin.Context) {
	s.Wallets(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) listTransactions(c *gin.Context) {
	s.ListTransactions(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) createTransaction(c *gin.Context) {
	s.CreateTransaction(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) transactionByID(c *gin.Context) {
	s.TransactionByID(c.Writer, c.Request, db(c), c.Param("id"), requestID(c))
}

func (s *server) dashboard(c *gin.Context) {
	s.Dashboard(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) listBudgets(c *gin.Context) {
	s.ListBudgets(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) createBudget(c *gin.Context) {
	s.CreateBudget(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) budgetByID(c *gin.Context) {
	s.BudgetByID(c.Writer, c.Request, db(c), c.Param("id"), requestID(c))
}

func (s *server) listRecurring(c *gin.Context) {
	s.ListRecurring(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) createRecurring(c *gin.Context) {
	s.CreateRecurring(c.Writer, c.Request, db(c), requestID(c))
}

func (s *server) recurringByID(c *gin.Context) {
	s.RecurringByID(c.Writer, c.Request, db(c), c.Param("id"), requestID(c))
}

func (s *server) notFound(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api") {
		requestID := s.ids.UUID()
		writeError(c.Writer, http.StatusNotFound, "NOT_FOUND", "Endpoint khong ton tai.", nil, requestID)
		return
	}
	c.Status(http.StatusNotFound)
}

func requestID(c *gin.Context) string {
	return c.GetString(ctxRequestID)
}

func db(c *gin.Context) DB {
	return c.MustGet(ctxDB).(DB)
}
