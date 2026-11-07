package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"linkit/internal/db"
)

// Relative durations are applied on save so client clock skew cannot shorten a share.
func shareExpireTime(duration *int64, absolute *string) (*time.Time, error) {
	if duration == nil {
		return parseExpireTime(absolute)
	}
	if *duration < 0 || *duration > 100*365*24*60*60 {
		return nil, fmt.Errorf("有效期需在 0 到 100 年之间")
	}
	if *duration == 0 {
		return nil, nil
	}
	expires := time.Now().Add(time.Duration(*duration) * time.Second)
	return &expires, nil
}

type managedShare struct {
	ID         int64      `json:"id"`
	Code       string     `json:"code"`
	Password   *string    `json:"password"`
	ExpireTime *time.Time `json:"expireTime"`
	Relay      bool       `json:"relay"`
	ViewCount  int64      `json:"viewCount"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func resourceShareIDs(c *gin.Context) (int64, int64, bool) {
	resourceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || resourceID <= 0 {
		c.JSON(http.StatusBadRequest, Fail[any]("资源ID无效", 400))
		return 0, 0, false
	}
	var shareID int64
	if raw := c.Param("shareId"); raw != "" {
		shareID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || shareID <= 0 {
			c.JSON(http.StatusBadRequest, Fail[any]("分享ID无效", 400))
			return 0, 0, false
		}
	}
	return resourceID, shareID, true
}

// This owner-only list is shared by the management panel and copy-link action.
func ResourceSharesHandler(store *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		resourceID, _, ok := resourceShareIDs(c)
		if !ok {
			return
		}
		user := middlewareGetUser(c)
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resource, err := store.Resource.FindByIDAndUser(ctx, resourceID, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("查询资源失败", 500))
			return
		}
		if resource == nil {
			c.JSON(http.StatusNotFound, Fail[any]("资源不存在", 404))
			return
		}
		shares, err := store.Share.ListByResource(ctx, resourceID, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("查询分享失败", 500))
			return
		}
		items := make([]managedShare, 0, len(shares))
		for _, share := range shares {
			items = append(items, managedShare{ID: share.ID, Code: share.Code, Password: share.Password,
				ExpireTime: share.ExpireTime, Relay: share.Relay, ViewCount: share.ViewCount, CreatedAt: share.CreatedAt})
		}
		c.Header("Cache-Control", "private, no-store")
		c.JSON(http.StatusOK, Ok(items, "ok"))
	}
}

type updateShareRequest struct {
	Password        *string `json:"password"`
	DurationSeconds *int64  `json:"durationSeconds"`
	Relay           *bool   `json:"relay"`
}

func UpdateResourceShareHandler(store *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		resourceID, shareID, ok := resourceShareIDs(c)
		if !ok {
			return
		}
		var req updateShareRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Fail[any]("参数错误", 400))
			return
		}
		values := make(map[string]any)
		if req.Password != nil {
			password := strings.TrimSpace(*req.Password)
			if password != "" && (len([]rune(password)) < 4 || len([]rune(password)) > 32) {
				c.JSON(http.StatusBadRequest, Fail[any]("分享密码长度需为 4-32 位", 400))
				return
			}
			values["password"] = nil
			if password != "" {
				values["password"] = password
			}
		}
		if req.DurationSeconds != nil {
			expires, err := shareExpireTime(req.DurationSeconds, nil)
			if err != nil {
				c.JSON(http.StatusBadRequest, Fail[any](err.Error(), 400))
				return
			}
			values["expire_time"] = expires
		}
		if req.Relay != nil {
			values["relay"] = *req.Relay
		}
		if len(values) == 0 {
			c.JSON(http.StatusBadRequest, Fail[any]("没有需要修改的设置", 400))
			return
		}
		user := middlewareGetUser(c)
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		updated, err := store.Share.UpdateByResource(ctx, shareID, resourceID, user.ID, values)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("更新分享失败", 500))
			return
		}
		if !updated {
			c.JSON(http.StatusNotFound, Fail[any]("分享不存在", 404))
			return
		}
		c.JSON(http.StatusOK, Ok(gin.H{}, "ok"))
	}
}

func DeleteResourceShareHandler(store *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		resourceID, shareID, ok := resourceShareIDs(c)
		if !ok {
			return
		}
		user := middlewareGetUser(c)
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		deleted, err := store.Share.DeleteByResource(ctx, shareID, resourceID, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("删除分享失败", 500))
			return
		}
		if !deleted {
			c.JSON(http.StatusNotFound, Fail[any]("分享不存在", 404))
			return
		}
		c.JSON(http.StatusOK, Ok(gin.H{}, "ok"))
	}
}
