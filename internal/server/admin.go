package server

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"linkit/internal/config"
	"linkit/internal/db"
	"linkit/internal/db/model"
	"linkit/internal/session"
	"linkit/internal/storage"
)

type adminConfigItem struct {
	Key    string  `json:"key"`
	Value  string  `json:"value"`
	Source string  `json:"source"` // db/env/default
	DB     *string `json:"dbValue,omitempty"`
}

type adminUpsertConfigPayload struct {
	AppConfig map[string]*string `json:"appConfig"`
}

type adminDashboardStats struct {
	TotalFiles      int64 `json:"totalFiles"`
	TotalFileSize   int64 `json:"totalFileSize"`
	TotalShareViews int64 `json:"totalShareViews"`
}

func AdminGetConfigHandler(store *db.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		dbItems, err := store.AppConfig.GetConfigs(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("读取配置失败", 500))
			return
		}

		keys := config.AppConfigKeys()
		items := make([]adminConfigItem, 0, len(keys))
		for _, key := range keys {
			val, _ := cfg.GetAppConfigValue(key)

			item := adminConfigItem{Key: key, Value: val, Source: "default"}
			if dbv, ok := dbItems[key]; ok {
				item.Source = "db"
				item.DB = &dbv
			} else if os.Getenv(key) != "" {
				item.Source = "env"
			}
			items = append(items, item)
		}

		c.JSON(http.StatusOK, Ok(gin.H{"items": items}, "ok"))
	}
}

func AdminUpsertConfigHandler(store *db.DB, cfg *config.Config, reg *storage.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminUpsertConfigPayload
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Fail[any]("参数错误", 400))
			return
		}
		if len(req.AppConfig) == 0 {
			c.JSON(http.StatusBadRequest, Fail[any]("缺少配置项", 400))
			return
		}

		updates := make(map[string]string, len(req.AppConfig))
		for key, value := range req.AppConfig {
			if value == nil {
				continue
			}
			key = strings.ToUpper(strings.TrimSpace(key))
			if key == "" {
				c.JSON(http.StatusBadRequest, Fail[any]("配置 key 不能为空", 400))
				return
			}
			if !config.IsAppConfigKey(key) {
				c.JSON(http.StatusBadRequest, Fail[any]("配置项不在白名单中", 400))
				return
			}
			updates[key] = *value
		}
		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, Fail[any]("缺少可更新的配置项", 400))
			return
		}

		nextCfg := *cfg
		for key, value := range updates {
			if ok := nextCfg.SetAppConfigValue(key, value); !ok {
				c.JSON(http.StatusBadRequest, Fail[any]("配置值格式错误", 400))
				return
			}
		}
		if err := reg.Validate(nextCfg); err != nil {
			c.JSON(http.StatusBadRequest, Fail[any]("存储配置无效: "+err.Error(), 400))
			return
		}

		ctx, cancel := store.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()

		for key, value := range updates {
			if err := store.AppConfig.SetConfig(ctx, cfg, key, value); err != nil {
				c.JSON(http.StatusInternalServerError, Fail[any]("保存配置失败", 500))
				return
			}
		}
		if err := reg.Reload(*cfg); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("保存成功但热更新失败，请检查配置后重试", 500))
			return
		}

		c.JSON(http.StatusOK, Ok(gin.H{"success": true}, "保存成功"))
	}
}

func AdminDashboardStatsHandler(store *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		totalFiles, totalSize, totalViews, err := store.Resource.GetDashboardStats(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("读取统计失败", 500))
			return
		}

		c.JSON(http.StatusOK, Ok(adminDashboardStats{
			TotalFiles:      totalFiles,
			TotalFileSize:   totalSize,
			TotalShareViews: totalViews,
		}, "ok"))
	}
}

type adminChangePasswordRequest struct {
	OldPassword  string `json:"oldPassword"`
	NewPassword1 string `json:"newPassword"`
	NewPassword2 string `json:"newPassword2"`
}

func AdminChangePasswordHandler(store *db.DB, cfg config.Config, sessions *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminChangePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Fail[any]("参数错误", 400))
			return
		}
		req.OldPassword = strings.TrimSpace(req.OldPassword)
		req.NewPassword1 = strings.TrimSpace(req.NewPassword1)
		req.NewPassword2 = strings.TrimSpace(req.NewPassword2)
		if req.OldPassword == "" || req.NewPassword1 == "" || req.NewPassword2 == "" {
			c.JSON(http.StatusBadRequest, Fail[any]("缺少密码参数", 400))
			return
		}
		if req.NewPassword1 != req.NewPassword2 {
			c.JSON(http.StatusBadRequest, Fail[any]("两次新密码不一致", 400))
			return
		}

		u := getUserFromContext(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, Fail[any]("未登录", 401))
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.OldPassword)); err != nil {
			c.JSON(http.StatusBadRequest, Fail[any]("原密码错误", 400))
			return
		}
		pwHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword1), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("修改失败", 500))
			return
		}

		ctx, cancel := store.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()

		if err := store.User.UpdatePassword(ctx, u.ID, string(pwHash)); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("修改失败", 500))
			return
		}

		// 刷新登录态，延长会话有效期，并轮换会话 ID。
		if err := issueSession(c, cfg, sessions, u.ID); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("修改失败", 500))
			return
		}

		c.JSON(http.StatusOK, Ok(gin.H{"success": true}, "修改成功"))
	}
}

func getUserFromContext(c *gin.Context) *model.User {
	if v, ok := c.Get("user"); ok {
		if u, ok := v.(*model.User); ok {
			return u
		}
	}
	return nil
}
