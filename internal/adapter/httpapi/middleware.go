package httpapi

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"expense-manager-mvp/internal/platform/config"
)

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && originAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Expose-Headers", "X-Request-Id")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func originAllowed(origin string, allowed []string) bool {
	for _, item := range allowed {
		if item == "*" || strings.EqualFold(item, origin) {
			return true
		}
	}
	return false
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		id := c.GetString(ctxRequestID)
		log.Printf(
			"http method=%s path=%s status=%d duration_ms=%d request_id=%s client_ip=%s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start).Milliseconds(),
			id,
			c.ClientIP(),
		)
	}
}

func authRateLimitMiddleware(cfg config.Config) gin.HandlerFunc {
	limiter := newAuthRateLimiter(cfg.AuthRatePerMinute)
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		switch c.Request.URL.Path {
		case "/api/auth/register", "/api/auth/login":
		default:
			c.Next()
			return
		}
		if !limiter.Allow(c.ClientIP()) {
			requestID := c.GetString(ctxRequestID)
			if requestID == "" {
				requestID = "rate-limit"
			}
			writeError(c.Writer, http.StatusTooManyRequests, "RATE_LIMITED", "Qua nhieu yeu cau. Thu lai sau.", nil, requestID)
			c.Abort()
			return
		}
		c.Next()
	}
}
