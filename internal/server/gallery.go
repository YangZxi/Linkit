package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"linkit/internal/db"
	"linkit/internal/db/model"
	"linkit/internal/storage"
)

func parsePositiveInt(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func GalleryHandler(store *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, Fail[any]("未登录", 401))
			return
		}
		page := parsePositiveInt(c.Query("page"), 1)
		size := parsePositiveInt(c.Query("size"), 10)
		tags, err := db.ParseTagsFromStrings([]string{c.Query("tags")})
		if err != nil {
			c.JSON(http.StatusBadRequest, Fail[any](err.Error(), 400))
			return
		}
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		items, total, err := store.Resource.ListByUser(ctx, user.ID, page, size, tags)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("获取资源失败", 500))
			return
		}
		store.Logger.Debug("获取资源列表", "user", user.Username, "page", page, "size", size, "tags", tags, "total", total)
		c.JSON(http.StatusOK, Ok(gin.H{"data": items, "total": total, "page": page}, "ok"))
	}
}

func GalleryTagsHandler(store *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, Fail[any]("未登录", 401))
			return
		}

		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		tags, err := store.Resource.ListTagsByUser(ctx, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("获取标签失败", 500))
			return
		}
		c.JSON(http.StatusOK, Ok(gin.H{"tags": tags}, "ok"))
	}
}

type galleryDeleteRequest struct {
	ID int64 `json:"id"`
}

type galleryPickUpdateRequest struct {
	ResourceID int64 `json:"resourceId"`
}

func GalleryDeleteHandler(store *db.DB, reg *storage.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, Fail[any]("未登录", 401))
			return
		}
		var req galleryDeleteRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.ID <= 0 {
			c.JSON(http.StatusBadRequest, Fail[any]("缺少资源ID", 400))
			return
		}
		ctx, cancel := store.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		resource, err := store.Resource.FindByIDAndUser(ctx, req.ID, user.ID)
		if err != nil || resource == nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("资源不存在", 500))
			return
		}
		stg, err := reg.ByStoredPath(resource.Path)
		if err != nil {
			reg.Logger.Error("存储路径无效", "err", err, "path", resource.Path)
		}
		if stg != nil {
			if err := stg.Delete(resource.Path); err != nil {
				reg.Logger.Error("删除存储文件失败", "err", err, "path", resource.Path)
				c.JSON(http.StatusInternalServerError, Fail[any]("删除资源失败", 500))
				return
			}
		}
		_, err = store.Resource.DeleteWithShare(ctx, resource.ID, user.ID)
		if err != nil {
			reg.Logger.Error("删除数据失败", "err", err, "resource", resource)
			c.JSON(http.StatusInternalServerError, Fail[any]("删除资源失败", 500))
			return
		}
		if err := store.Resource.ClearUserPickIfMatch(ctx, user.ID, resource.ID); err != nil {
			store.Logger.Warn("清理 pick 记录失败", "user", user.Username, "resource_id", resource.ID, "error", err)
		}
		store.Logger.Info("删除资源完成", "user", user.Username, "resource_id", resource.ID, "file", resource.Filename)
		c.JSON(http.StatusOK, Ok(*new(any), "ok"))
	}
}

func GalleryPickHandler(store *db.DB, reg *storage.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, Fail[any]("未登录", 401))
			return
		}

		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var (
			resource *model.Resource
			mode     = "latest"
		)
		if pickID, ok, err := store.Resource.GetUserPickResourceID(ctx, user.ID); err != nil {
			reg.Logger.Error("读取 pick 资源失败", "err", err, "user_id", user.ID)
			c.JSON(http.StatusInternalServerError, Fail[any]("获取资源失败", 500))
			return
		} else if ok {
			mode = "pick"
			resource, err = store.Resource.FindByIDAndUser(ctx, pickID, user.ID)
			if err != nil {
				reg.Logger.Error("读取 pick 资源详情失败", "err", err, "user_id", user.ID, "resource_id", pickID)
				c.JSON(http.StatusInternalServerError, Fail[any]("获取资源失败", 500))
				return
			}
			if resource == nil {
				mode = "latest"
				if err := store.Resource.ClearUserPickIfMatch(ctx, user.ID, pickID); err != nil {
					store.Logger.Warn("清理失效 pick 失败", "user_id", user.ID, "resource_id", pickID, "error", err)
				}
			}
		}
		if resource == nil {
			var err error
			resource, err = store.Resource.FindLatestByUser(ctx, user.ID)
			if err != nil {
				reg.Logger.Error("查询最新资源失败", "err", err, "user_id", user.ID)
				c.JSON(http.StatusInternalServerError, Fail[any]("获取资源失败", 500))
				return
			}
		}
		if resource == nil {
			c.JSON(http.StatusNotFound, Fail[any]("暂无可下载资源", 404))
			return
		}

		record := &model.ShareResource{
			ResourceID: resource.ID,
			UserID:     resource.UserID,
			Filename:   resource.Filename,
			Path:       resource.Path,
			Type:       resource.Type,
			Relay:      true,
		}
		storageDriver, err := reg.ByStoredPath(resource.Path)
		if err != nil {
			reg.Logger.Error("存储路径无效", "err", err, "path", resource.Path)
			c.JSON(http.StatusInternalServerError, Fail[any]("资源路径无效", 500))
			return
		}
		reg.Logger.Debug("下载资源", "user_id", user.ID, "resource_id", resource.ID, "mode", mode, "file", resource.Filename, "storage", storageDriver.Platform())

		if storageDriver.Platform() != storage.PlatformLocal {
			// 最新资源下载要求直接返回文件流，云存储场景统一走 relay 代理并强制 attachment。
			downloadForS3(c, reg, record, storageDriver)
			return
		}
		downloadForLocal(c, reg, record, storageDriver)
	}
}

func GalleryPickUpdateHandler(store *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, Fail[any]("未登录", 401))
			return
		}
		var req galleryPickUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.ResourceID <= 0 {
			c.JSON(http.StatusBadRequest, Fail[any]("缺少资源ID", 400))
			return
		}
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		resource, err := store.Resource.FindByIDAndUser(ctx, req.ResourceID, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("检索资源失败", 500))
			return
		}
		if resource == nil {
			c.JSON(http.StatusNotFound, Fail[any]("资源不存在", 404))
			return
		}
		if err := store.Resource.SetUserPickResourceID(user.ID, resource.ID); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("保存失败", 500))
			return
		}
		c.JSON(http.StatusOK, Ok(*new(any), "ok"))
	}
}
