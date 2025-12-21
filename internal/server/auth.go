package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"linkit/internal/config"
	"linkit/internal/db"
	"linkit/internal/db/model"
)

func generateSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func setSessionCookie(c *gin.Context, cfg config.Config, token string) {
	c.SetCookie(cfg.SessionCookie, token, int(cfg.CookieMaxAge.Seconds()), "/", "", cfg.CookieSecure, true)
}

func clearSessionCookie(c *gin.Context, cfg config.Config) {
	c.SetCookie(cfg.SessionCookie, "", -1, "/", "", cfg.CookieSecure, true)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userPayload struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
}

func LoginHandler(store *db.DB, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Fail[any]("缺少用户名或密码", 400))
			return
		}
		if req.Username == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, Fail[any]("缺少用户名或密码", 400))
			return
		}
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		user, err := store.User.FindByCredential(ctx, req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("登录失败", 500))
			return
		}
		if user == nil {
			c.JSON(http.StatusUnauthorized, Fail[any]("用户不存在或凭证错误", 401))
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, Fail[any]("用户不存在或凭证错误", 401))
			return
		}
		token, err := generateSessionToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("登录失败", 500))
			return
		}
		if err := store.User.UpdateToken(ctx, user.ID, &token); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("登录失败", 500))
			return
		}
		setSessionCookie(c, cfg, token)
		store.Logger.Info("用户登录成功", "user", user.Username)
		c.JSON(http.StatusOK, Ok(userPayload{ID: user.ID, Username: user.Username, Nickname: user.Nickname, Email: user.Email}, "登录成功"))
	}
}

func LogoutHandler(store *db.DB, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user != nil {
			ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
			defer cancel()
			_ = store.User.UpdateToken(ctx, user.ID, nil)
			store.Logger.Info("用户退出登录", "user", user.Username)
		}
		clearSessionCookie(c, cfg)
		c.JSON(http.StatusOK, Ok(gin.H{"success": true}, "退出成功"))
	}
}

func MeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, Fail[any]("未登录", 401))
			return
		}
		c.JSON(http.StatusOK, Ok(userPayload{ID: user.ID, Username: user.Username, Nickname: user.Nickname, Email: user.Email}, "ok"))
	}
}

type refreshResponse struct {
	User      userPayload `json:"user"`
	Refreshed bool        `json:"refreshed"`
}

func RefreshHandler(store *db.DB, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user == nil {
			clearSessionCookie(c, cfg)
			c.JSON(http.StatusUnauthorized, Fail[any]("未登录", 401))
			return
		}
		token, err := generateSessionToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("刷新失败", 500))
			return
		}
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if err := store.User.UpdateToken(ctx, user.ID, &token); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("刷新失败", 500))
			return
		}
		setSessionCookie(c, cfg, token)
		store.Logger.Debug("刷新会话", "user", user.Username)
		c.JSON(http.StatusOK, Ok(refreshResponse{User: userPayload{ID: user.ID, Username: user.Username, Nickname: user.Nickname, Email: user.Email}, Refreshed: true}, "刷新成功"))
	}
}

func middlewareGetUser(c *gin.Context) *model.User {
	if v, ok := c.Get("user"); ok {
		if u, ok := v.(*model.User); ok {
			return u
		}
	}
	return nil
}
