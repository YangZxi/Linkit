package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 支持多来源：逗号分隔，* 表示全部
func CORS(allowOrigin string) gin.HandlerFunc {
	origins := parseOrigins(allowOrigin)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && originAllowed(origin, origins) {
			slog.Info(origin)
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func parseOrigins(val string) []string {
	if val == "" {
		return []string{"*"}
	}
	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.TrimRight(p, "/"))
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = []string{"*"}
	}
	return out
}

func originAllowed(origin string, allowed []string) bool {
	normalized := strings.TrimRight(origin, "/")
	for _, a := range allowed {
		if a == "*" || strings.EqualFold(normalized, a) {
			return true
		}
	}
	return false
}
