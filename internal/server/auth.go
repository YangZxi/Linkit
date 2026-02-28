package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"linkit/internal/config"
	"linkit/internal/db"
	"linkit/internal/db/model"
	"linkit/internal/session"
)

func setSessionCookie(c *gin.Context, cfg config.Config, sessionID string) {
	c.SetCookie(cfg.SessionCookie, sessionID, int(cfg.CookieMaxAge.Seconds()), "/", "", cfg.CookieSecure, true)
}

func clearSessionCookie(c *gin.Context, cfg config.Config) {
	c.SetCookie(cfg.SessionCookie, "", -1, "/", "", cfg.CookieSecure, true)
}

func getSessionIDFromCookie(c *gin.Context, cfg config.Config) string {
	raw, err := c.Cookie(cfg.SessionCookie)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(raw)
}

func issueSession(c *gin.Context, cfg config.Config, sessions *session.Manager, userID int64) error {
	oldSessionID := getSessionIDFromCookie(c, cfg)
	sessionID, err := sessions.Rotate(oldSessionID, userID, cfg.CookieMaxAge)
	if err != nil {
		return err
	}
	setSessionCookie(c, cfg, sessionID)
	return nil
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

func LoginHandler(store *db.DB, cfg config.Config, sessions *session.Manager) gin.HandlerFunc {
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
		if err := issueSession(c, cfg, sessions, user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("登录失败", 500))
			return
		}
		store.Logger.Info("用户登录成功", "user", user.Username)
		c.JSON(http.StatusOK, Ok(userPayload{ID: user.ID, Username: user.Username, Nickname: user.Nickname, Email: user.Email}, "登录成功"))
	}
}

func LogoutHandler(store *db.DB, cfg config.Config, sessions *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessions.Delete(getSessionIDFromCookie(c, cfg))
		user := middlewareGetUser(c)
		if user != nil {
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

func RefreshHandler(store *db.DB, cfg config.Config, sessions *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user == nil {
			sessions.Delete(getSessionIDFromCookie(c, cfg))
			clearSessionCookie(c, cfg)
			c.JSON(http.StatusUnauthorized, Fail[any]("未登录", 401))
			return
		}
		if err := issueSession(c, cfg, sessions, user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("刷新失败", 500))
			return
		}
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
