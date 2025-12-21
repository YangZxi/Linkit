package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"log/slog"
)

func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		logger.Info("请求", "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "duration_ms", duration.Milliseconds())
	}
}
