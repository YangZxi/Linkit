package middleware

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"linkit/internal/config"
)

var builtinWildcardDomains = []string{"xiaosm.cn", "waizx.com"}

type CORSManager struct {
	mu      sync.RWMutex
	origins []string
}

func NewCORSManager(cfg *config.Config, fallback string) *CORSManager {
	m := &CORSManager{}
	m.refresh(cfg, fallback)
	return m
}

func (m *CORSManager) UpdateFromConfig(cfg *config.Config) {
	m.refresh(cfg, "")
}

func (m *CORSManager) allowed(origin string) bool {
	m.mu.RLock()
	origins := m.origins
	m.mu.RUnlock()
	normalized := strings.TrimRight(origin, "/")

	for _, a := range origins {
		if a == "*" || strings.EqualFold(normalized, a) || wildcardPatternAllows(normalized, a) {
			return true
		}
	}
	host := extractHost(normalized)
	if host == "" {
		return false
	}
	for _, domain := range builtinWildcardDomains {
		if hostMatchesDomain(host, domain) {
			return true
		}
	}
	return false
}

func (m *CORSManager) refresh(cfg *config.Config, fallback string) {
	allow := mergeOrigins(cfg, fallback)
	if allow == "" {
		allow = "*"
	}
	m.mu.Lock()
	m.origins = parseOrigins(allow)
	m.mu.Unlock()
}

func mergeOrigins(cfg *config.Config, fallback string) string {
	allow := strings.TrimSpace(fallback)
	if cfg == nil {
		return allow
	}
	base := strings.TrimSpace(cfg.FrontendOrigin)
	extra := strings.TrimSpace(cfg.AppConfig.CorsAllowedList)
	switch {
	case base == "" && extra == "":
		return allow
	case base == "":
		return extra
	case extra == "":
		return base
	default:
		return base + "," + extra
	}
}

func CORSMiddleware(manager *CORSManager) gin.HandlerFunc {
	if manager == nil {
		manager = NewCORSManager(nil, "*")
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && manager.allowed(origin) {
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

func wildcardPatternAllows(origin string, pattern string) bool {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return false
	}
	if !strings.Contains(trimmed, "*") && !strings.HasPrefix(trimmed, ".") {
		return false
	}
	host := extractHost(origin)
	if host == "" {
		return false
	}
	target := trimmed
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		u, err := url.Parse(target)
		if err != nil {
			return false
		}
		target = u.Hostname()
	}
	target = strings.TrimPrefix(target, "*.")
	target = strings.TrimPrefix(target, ".")
	if target == "" {
		return false
	}
	return hostMatchesDomain(host, target)
}

func extractHost(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func hostMatchesDomain(host, domain string) bool {
	host = strings.ToLower(host)
	domain = strings.ToLower(domain)
	if host == domain {
		return true
	}
	return strings.HasSuffix(host, "."+domain)
}
