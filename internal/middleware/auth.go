package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"linkit/internal/config"
	"linkit/internal/db"
	"linkit/internal/db/model"
	"linkit/internal/server"
)

const userContextKey = "user"

func AuthOptional(store *db.DB, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cfg.SessionCookie)
		if err != nil || token == "" {
			c.Next()
			return
		}
		user, err := store.User.GetByToken(c.Request.Context(), token)
		if err != nil || user == nil {
			c.Next()
			return
		}
		c.Set(userContextKey, user)
		c.Next()
	}
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
		if user.Username != cfg.AdminUsername {
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
