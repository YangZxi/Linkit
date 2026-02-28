package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"linkit/internal/config"
	"linkit/internal/db"
	"linkit/internal/db/model"
	"linkit/internal/server"
	"linkit/internal/session"
)

const userContextKey = "user"

func AuthOptional(store *db.DB, cfg config.Config, sessions *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := resolveUser(c, store, cfg, sessions)
		if err != nil || user == nil {
			c.Next()
			return
		}
		c.Set(userContextKey, user)
		c.Next()
	}
}

func resolveUser(c *gin.Context, store *db.DB, cfg config.Config, sessions *session.Manager) (*model.User, error) {
	if token, ok := tokenFromAuthorization(c.GetHeader("Authorization")); ok {
		if token == "" {
			return nil, nil
		}
		return store.User.GetByToken(c.Request.Context(), token)
	}
	sessionID, ok := sessionIDFromCookie(c, cfg)
	if !ok {
		return nil, nil
	}
	userID, ok := sessions.Resolve(sessionID)
	if !ok {
		return nil, nil
	}
	return store.User.GetByID(c.Request.Context(), userID)
}

func tokenFromAuthorization(raw string) (string, bool) {
	val := strings.TrimSpace(raw)
	if val == "" {
		return "", false
	}
	parts := strings.Fields(val)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1]), true
	}
	return val, true
}

func sessionIDFromCookie(c *gin.Context, cfg config.Config) (string, bool) {
	raw, err := c.Cookie(cfg.SessionCookie)
	if err != nil {
		return "", false
	}
	sessionID := strings.TrimSpace(raw)
	if sessionID == "" {
		return "", false
	}
	return sessionID, true
}

func AuthRequired(store *db.DB, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUserFromContext(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, server.Fail[any]("未登录", 401))
			return
		}
		c.Next()
	}
}

func AdminRequired(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUserFromContext(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, server.Fail[any]("未登录", 401))
			return
		}
		if user.Username != cfg.AdminUsername && user.ID != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, server.Fail[any]("无权限", 403))
			return
		}
		c.Next()
	}
}

func GetUserFromContext(c *gin.Context) *model.User {
	if v, ok := c.Get(userContextKey); ok {
		if user, ok := v.(*model.User); ok {
			return user
		}
	}
	return nil
}
