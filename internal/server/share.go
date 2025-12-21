package server

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"linkit/internal/db"
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
		record, err := store.Resource.GetShareByCode(ctx, code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("查询失败", 500))
			return
		}
		if record == nil {
			c.JSON(http.StatusNotFound, Fail[any]("资源不存在", 404))
			return
		}
		store.Logger.Debug("查询分享信息", "code", code, "file", record.Filename)
		c.JSON(http.StatusOK, Ok(record, "ok"))
	}
}

func DownloadHandler(store *db.DB, reg *storage.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("slug")
		if !codeRegex.MatchString(code) {
			c.JSON(http.StatusNotFound, Fail[any]("短链无效", 404))
			return
		}
		ctx, cancel := store.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		record, err := store.Resource.GetShareByCode(ctx, code)
		if err != nil {
			reg.Logger.Error("获取短链失败", "err", err)
			c.JSON(http.StatusInternalServerError, Fail[any]("资源不存在", 404))
			return
		}
		if record == nil {
			c.JSON(http.StatusNotFound, Fail[any]("资源不存在", 404))
			return
		}

		if err := store.Resource.IncrementShareViewCount(ctx, record.ShareID); err != nil {
			reg.Logger.Error("更新短链访问次数失败", "err", err, "code", code)
		}

		storageDriver, err := reg.ByStoredPath(record.Path)
		if err != nil {
			reg.Logger.Error("存储路径无效", "err", err)
			c.JSON(http.StatusInternalServerError, Fail[any]("资源路径无效", 500))
			return
		}
		reg.Logger.Debug("处理分享请求", "code", code, "file", record.Filename, "storage", storageDriver.Platform())

		// 云端文件：重定向到带签名的下载链接。
		if storageDriver.Platform() != storage.PlatformLocal {
			signed, err := storageDriver.GetURL(record.Path, 30*time.Minute)
			if err != nil {
				reg.Logger.Error("生成签名链接失败", "err", err)
				c.JSON(http.StatusInternalServerError, Fail[any]("资源失效", 410))
				return
			}
			c.Redirect(http.StatusFound, signed)
			return
		}

		// 本地文件：直接传输文件内容。
		filePath, err := storageDriver.GetURL(record.Path, 0)
		if err != nil {
			reg.Logger.Error("获取本地文件路径失败", "err", err)
			c.JSON(http.StatusInternalServerError, Fail[any]("资源失效", 410))
			return
		}
		f, err := os.Open(filePath)
		if err != nil {
			reg.Logger.Error("打开文件失败", "err", err)
			c.JSON(http.StatusGone, Fail[any]("文件已失效", 410))
			return
		}
		defer f.Close()
		stat, err := f.Stat()
		if err != nil {
			reg.Logger.Error("读取文件信息失败", "err", err)
			c.JSON(http.StatusGone, Fail[any]("文件已失效", 410))
			return
		}
		// 本地文件：允许浏览器缓存，并通过 ETag/Last-Modified 协商避免重复下载。
		etag := buildWeakETag(stat)
		setCacheHeaders(c, stat, etag)
		if isNotModified(c, stat, etag) {
			c.Status(http.StatusNotModified)
			return
		}
		contentType := record.Type
		if contentType == "" {
			contentType = storage.GuessMime(record.Filename)
		}
		rangeHeader := c.GetHeader("Range")
		if rangeHeader != "" {
			start, end, ok := parseRange(rangeHeader, stat.Size())
			if !ok {
				c.JSON(http.StatusRequestedRangeNotSatisfiable, Fail[any]("Range 无效", 416))
				return
			}
			if _, err := f.Seek(start, io.SeekStart); err != nil {
				c.JSON(http.StatusInternalServerError, Fail[any]("读取失败", 500))
				return
			}
			length := end - start + 1
			c.Status(http.StatusPartialContent)
			c.Header("Content-Type", contentType)
			c.Header("Content-Length", strconv.FormatInt(length, 10))
			c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, stat.Size()))
			c.Header("Accept-Ranges", "bytes")
			c.Header("Content-Disposition", buildContentDisposition(filepath.Base(record.Filename), true))
			if c.Request.Method == http.MethodHead {
				return
			}
			io.CopyN(c.Writer, f, length)
			return
		}

		c.Header("Content-Type", contentType)
		c.Header("Content-Length", strconv.FormatInt(stat.Size(), 10))
		c.Header("Accept-Ranges", "bytes")
		c.Header("Content-Disposition", buildContentDisposition(filepath.Base(record.Filename), true))
		c.Status(http.StatusOK)
		if c.Request.Method == http.MethodHead {
			return
		}
		io.Copy(c.Writer, f)
		reg.Logger.Info("完成文件传输", "code", code, "file", record.Filename, "size", stat.Size())
	}
}

func buildWeakETag(stat os.FileInfo) string {
	// 采用弱 ETag：避免误判“强一致”；同时足以用于协商缓存，减少重复下载。
	return fmt.Sprintf(`W/"%x-%x"`, stat.Size(), stat.ModTime().UnixNano())
}

func setCacheHeaders(c *gin.Context, stat os.FileInfo, etag string) {
	c.Header("ETag", etag)
	c.Header("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
	// 安全优先：允许缓存，但每次使用前必须与服务端协商；避免长期缓存导致短链复用时内容错配。
	c.Header("Cache-Control", "public, max-age=0, must-revalidate")
	// Range 会影响响应体，提示中间缓存按 Range 区分（浏览器也会更谨慎处理）。
	c.Header("Vary", "Range")
}

func isNotModified(c *gin.Context, stat os.FileInfo, etag string) bool {
	// 优先 ETag；命中则直接 304。
	if matchIfNoneMatch(c.GetHeader("If-None-Match"), etag) {
		return true
	}
	// 其次 Last-Modified。
	if ims := strings.TrimSpace(c.GetHeader("If-Modified-Since")); ims != "" {
		t, err := http.ParseTime(ims)
		if err == nil {
			mod := stat.ModTime().UTC().Truncate(time.Second)
			if !mod.After(t.UTC()) {
				return true
			}
		}
	}
	return false
}

func matchIfNoneMatch(header string, etag string) bool {
	h := strings.TrimSpace(header)
	if h == "" {
		return false
	}
	if h == "*" {
		return true
	}
	parts := strings.Split(h, ",")
	for _, p := range parts {
		if strings.TrimSpace(p) == etag {
			return true
		}
	}
	return false
}

func parseRange(header string, size int64) (int64, int64, bool) {
	const prefix = "bytes="
	if !strings.HasPrefix(header, prefix) {
		return 0, 0, false
	}
	parts := strings.Split(strings.TrimPrefix(header, prefix), "-")
	if len(parts) != 2 {
		return 0, 0, false
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 {
		return 0, 0, false
	}
	end := size - 1
	if parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < start {
			return 0, 0, false
		}
		if end >= size {
			end = size - 1
		}
	}
	return start, end, true
}

func buildContentDisposition(filename string, inline bool) string {
	safe := regexp.MustCompile(`[^\x20-\x7E]`).ReplaceAllString(filename, "_")
	encoded := url.QueryEscape(filename)
	typeStr := "attachment"
	if inline {
		typeStr = "inline"
	}
	return fmt.Sprintf("%s; filename=\"%s\"; filename*=UTF-8''%s", typeStr, safe, encoded)
}
