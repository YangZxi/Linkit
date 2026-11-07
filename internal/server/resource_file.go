package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"linkit/internal/db"
	"linkit/internal/db/model"
	"linkit/internal/storage"
)

func ResourceFileHandler(store *db.DB, reg *storage.Registry) gin.HandlerFunc {
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
		driver, err := reg.ByStoredPath(resource.Path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("资源存储不可用", 500))
			return
		}
		c.Set("ownerFile", true)
		c.Header("Cache-Control", "private, no-store")
		record := &model.ShareResource{ResourceID: resource.ID, UserID: user.ID,
			Filename: resource.Filename, Path: resource.Path, Type: resource.Type, Relay: true}
		if driver.Platform() == storage.PlatformLocal {
			downloadForLocal(c, reg, record, driver)
		} else {
			downloadForS3(c, reg, record, driver)
		}
	}
}

func persistUploadedResource(c *gin.Context, store *db.DB, reg *storage.Registry, previous *model.Resource, next model.Resource, tags []string) (int64, string, error) {
	if previous == nil {
		return persistResource(c, store, next, tags)
	}
	ctx, cancel := store.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	updated, err := store.Resource.ReplaceFile(ctx, *previous, next)
	if err == nil && !updated {
		err = fmt.Errorf("资源已被修改或删除，请刷新后重试")
	}
	if err != nil {
		cleanupUnusedFile(c, store, reg, next.Path)
		return 0, "", err
	}
	if !cleanupUnusedFile(c, store, reg, previous.Path) {
		c.Set("uploadWarning", "文件已替换，但旧文件清理失败，请联系管理员处理")
	}
	return previous.ID, "", nil
}

// Legacy uploads can share a stored path; never remove a file still in use.
func cleanupUnusedFile(c *gin.Context, store *db.DB, reg *storage.Registry, path string) bool {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 10*time.Second)
	defer cancel()
	count, err := store.Resource.CountByPath(ctx, path)
	if err == nil && count > 0 {
		return true
	}
	if err == nil {
		var driver storage.Storage
		driver, err = reg.ByStoredPath(path)
		if err == nil {
			err = driver.Delete(path)
		}
	}
	if err != nil {
		reg.Logger.Error("清理未引用文件失败", "path", path, "error", err)
		return false
	}
	return true
}
