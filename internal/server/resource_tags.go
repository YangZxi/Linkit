package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"linkit/internal/db"
)

func ResourceTagUpdateHandler(store *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, Fail[any]("未登录", 401))
			return
		}
		resourceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || resourceID <= 0 {
			c.JSON(http.StatusBadRequest, Fail[any]("资源ID无效", 400))
			return
		}
		var req struct {
			Tag    string `json:"tag"`
			Remove bool   `json:"remove"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Fail[any]("标签参数无效", 400))
			return
		}
		tags, err := db.ParseTagsFromStrings([]string{req.Tag})
		if err != nil || len(tags) != 1 {
			c.JSON(http.StatusBadRequest, Fail[any]("请提供一个不超过 6 个字符的标签", 400))
			return
		}
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		tags, err = store.Resource.ChangeTag(ctx, resourceID, user.ID, tags[0], req.Remove)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, Fail[any]("资源不存在", 404))
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("更新标签失败", 500))
			return
		}
		c.JSON(http.StatusOK, Ok(gin.H{"tags": tags}, "ok"))
	}
}
