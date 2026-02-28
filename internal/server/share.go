package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"linkit/internal/db"
	"linkit/internal/db/model"
	"linkit/internal/storage"
)

var codeRegex = regexp.MustCompile(`^[a-zA-Z0-9]{6}$`)

func ShareInfoHandler(store *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		if !codeRegex.MatchString(code) {
			c.JSON(http.StatusNotFound, Fail[any]("短链无效", 404))
			return
		}
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		record, err := store.Share.GetShareByCode(ctx, code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("查询失败", 500))
			return
		}
		if record == nil {
			c.JSON(http.StatusNotFound, Fail[any]("资源不存在", 404))
			return
		}
		slog.Info("record", "data", record, "password", record.Password)
		if !validateShareAccess(c, record) {
			return
		}
		store.Logger.Debug("查询分享信息", "code", code, "file", record.Filename)
		c.JSON(http.StatusOK, Ok(record, "ok"))
	}
}

func DownloadHandler(store *db.DB, reg *storage.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		if !codeRegex.MatchString(code) {
			c.JSON(http.StatusNotFound, Fail[any]("短链无效", 404))
			return
		}
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		record, err := store.Share.GetShareByCode(ctx, code)
		if err != nil {
			reg.Logger.Error("获取短链失败", "err", err)
			c.JSON(http.StatusInternalServerError, Fail[any]("资源不存在", 404))
			return
		}
		if record == nil {
			c.JSON(http.StatusNotFound, Fail[any]("资源不存在", 404))
			return
		}
		if !validateShareAccess(c, record) {
			return
		}

		if err := store.Share.IncrementShareViewCount(ctx, record.ShareID); err != nil {
			reg.Logger.Error("更新短链访问次数失败", "err", err, "code", code)
		}

		storageDriver, err := reg.ByStoredPath(record.Path)
		if err != nil {
			reg.Logger.Error("存储路径无效", "err", err)
			c.JSON(http.StatusInternalServerError, Fail[any]("资源路径无效", 500))
			return
		}
		reg.Logger.Debug("处理分享请求", "code", code, "file", record.Filename, "storage", storageDriver.Platform())

		if storageDriver.Platform() != storage.PlatformLocal {
			downloadForS3(c, reg, record, storageDriver)
			return
		}

		downloadForLocal(c, reg, record, storageDriver)
	}
}

func validateShareAccess(c *gin.Context, record *model.ShareResource) bool {
	if record == nil {
		c.JSON(http.StatusNotFound, Fail[any]("分享不存在或已失效", 404))
		return false
	}
	user := middlewareGetUser(c)
	if user != nil && user.ID == record.UserID {
		return true
	}
	if record.ExpireTime != nil && time.Now().After(*record.ExpireTime) {
		c.JSON(http.StatusGone, Fail[any]("分享已过期", 410))
		return false
	}
	if record.Password == nil || *record.Password == "" {
		return true
	}
	password := strings.TrimSpace(c.Query("pwd"))
	if password == "" {
		c.JSON(http.StatusUnauthorized, Fail[any]("密码错误", 401))
		return false
	}
	if password != *record.Password {
		c.JSON(http.StatusUnauthorized, Fail[any]("密码错误", 401))
		return false
	}
	return true
}

type createShareRequest struct {
	ResourceID int64   `json:"resourceId"`
	Password   string  `json:"password"`
	ExpireTime *string `json:"expireTime"`
	Relay      bool    `json:"relay"`
}

type createShareResponse struct {
	Code string `json:"code"`
}

func CreateShareHandler(store *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, Fail[any]("未登录", 401))
			return
		}
		var req createShareRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.ResourceID <= 0 {
			c.JSON(http.StatusBadRequest, Fail[any]("参数错误", 400))
			return
		}
		password := strings.TrimSpace(req.Password)
		passwordLen := len([]rune(password))
		if passwordLen < 4 || passwordLen > 32 {
			c.JSON(http.StatusBadRequest, Fail[any]("分享密码长度需为 4-32 位", 400))
			return
		}
		expireTime, err := parseExpireTime(req.ExpireTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, Fail[any](err.Error(), 400))
			return
		}
		if expireTime != nil && time.Now().After(*expireTime) {
			c.JSON(http.StatusBadRequest, Fail[any]("过期时间需晚于当前时间", 400))
			return
		}
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resource, err := store.Resource.FindByIDAndUser(ctx, req.ResourceID, user.ID)
		if err != nil || resource == nil {
			c.JSON(http.StatusNotFound, Fail[any]("资源不存在", 404))
			return
		}
		shareRecord, err := store.Share.CreateShareCode(ctx, req.ResourceID, user.ID, &password, expireTime, req.Relay)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("创建分享失败", 500))
			return
		}
		c.JSON(http.StatusOK, Ok(createShareResponse{Code: shareRecord.Code}, "ok"))
	}
}

func parseExpireTime(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	if isDigits(value) {
		ts, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("过期时间格式错误")
		}
		if ts > 1_000_000_000_000 {
			ts = ts / 1000
		}
		t := time.Unix(ts, 0)
		return &t, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("过期时间格式错误")
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
