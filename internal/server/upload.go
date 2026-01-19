package server

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"linkit/internal/config"
	"linkit/internal/db"
	"linkit/internal/db/model"
	"linkit/internal/storage"
)

const (
	uploadField = "file"
)

type uploadResponse struct {
	Merged      bool   `json:"merged"`
	UploadID    string `json:"uploadId"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size,omitempty"`
	Skipped     bool   `json:"skipped,omitempty"`
	ChunkIndex  *int64 `json:"chunkIndex,omitempty"`
	TotalChunks *int64 `json:"totalChunks,omitempty"`
	ChunkSize   *int64 `json:"chunkSize,omitempty"`
	ShareCode   string `json:"shareCode,omitempty"`
	ResourceID  int64  `json:"resourceId,omitempty"`
}

func UploadQueryHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		uploadID := c.Query("uploadId")
		if uploadID == "" {
			c.JSON(http.StatusBadRequest, Fail[any]("缺少 uploadId", 400))
			return
		}
		if err := ensureDir(cfg.ChunkDir); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("准备目录失败", 500))
			return
		}
		cleanupChunks(cfg)
		chunkFolder := filepath.Join(cfg.ChunkDir, uploadID)
		entries, err := os.ReadDir(chunkFolder)
		if err != nil {
			c.JSON(http.StatusOK, Ok(gin.H{"uploaded": []int64{}}, "ok"))
			return
		}
		var uploaded []int64
		for _, entry := range entries {
			if !entry.Type().IsRegular() {
				continue
			}
			if idx, err := strconv.ParseInt(entry.Name(), 10, 64); err == nil && idx >= 0 {
				uploaded = append(uploaded, idx)
			}
		}
		c.JSON(http.StatusOK, Ok(gin.H{"uploaded": uploaded}, "ok"))
	}
}

func UploadHandler(store *db.DB, cfg *config.Config, reg *storage.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middlewareGetUser(c)
		if user == nil {
			user = &model.User{ID: db.GuestUserID, Username: db.GuestUsername}
		}
		if err := ensureDir(cfg.ChunkDir); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("准备目录失败", 500))
			return
		}
		if err := ensureDir(cfg.MergeDir); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("准备目录失败", 500))
			return
		}
		if err := ensureDir(cfg.LocalRoot); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("准备目录失败", 500))
			return
		}
		cleanupChunks(cfg)

		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, Fail[any]("上传数据格式错误", 400))
			return
		}
		files := form.File[uploadField]
		if len(files) != 1 {
			c.JSON(http.StatusBadRequest, Fail[any]("只支持单文件上传", 400))
			return
		}
		fh := files[0]

		uploadID := firstValue(form.Value["uploadId"], fmt.Sprintf("%d-%s", time.Now().UnixMilli(), fh.Filename))
		fileName := filepath.Base(firstValue(form.Value["fileName"], fh.Filename))
		fileSize := parseInt64(firstValue(form.Value["fileSize"], fmt.Sprintf("%d", fh.Size)), fh.Size)
		chunkIndexPtr := parseOptionalInt64(firstValue(form.Value["chunkIndex"], ""))
		totalChunksPtr := parseOptionalInt64(firstValue(form.Value["totalChunks"], ""))
		chunkSizePtr := parseOptionalInt64(firstValue(form.Value["chunkSize"], ""))

		// 访客上传按白名单限制
		if user.ID == db.GuestUserID {
			guestPolicy := newGuestUploadPolicy(cfg)
			if guestPolicy == nil {
				c.JSON(http.StatusForbidden, Fail[any]("不允许访客上传", 403))
				return
			}
			if ok, msg := guestPolicy.allow(fileName, fileSize); !ok {
				c.JSON(http.StatusBadRequest, Fail[any](msg, 400))
				return
			}
		}

		requireChunk := fileSize > cfg.ChunkThreshold
		if fileSize > cfg.MaxFileSize {
			c.JSON(http.StatusBadRequest, Fail[any]("文件大小超过限制", 400))
			return
		}
		fileType := storage.GuessMime(fileName)

		stg := reg.Active()
		reg.Logger.Info("接收上传请求", "user", user.Username, "file", fileName, "size", fileSize, "chunk", requireChunk)

		// 小文件直接写
		if !requireChunk && (totalChunksPtr == nil || *totalChunksPtr <= 1) {
			hash, data, err := readAndHash(fh)
			if err != nil {
				slog.Error("获取文件Hash失败", "err", err)
				c.JSON(http.StatusInternalServerError, Fail[any]("存储失败", 500))
				return
			}
			fileSize := int64(len(data))
			objectKey := storage.BuildObjectKey(hash, fileName, time.Now())
			storedPath, err := stg.Write(objectKey, bytes.NewReader(data), int64(len(data)), fileType)
			if err != nil {
				slog.Error("写入文件失败", "err", err)
				c.JSON(http.StatusInternalServerError, Fail[any]("存储失败", 500))
				return
			}
			resID, share, err := persistResource(c, store, model.Resource{Filename: fileName, Hash: hash, Type: fileType, Path: storedPath, FileSize: fileSize, UserID: user.ID})
			if err != nil {
				slog.Error("写入数据库失败", "err", err)
				c.JSON(http.StatusInternalServerError, Fail[any]("存储失败", 500))
				return
			}
			reg.Logger.Info("文件上传完成", "user", user.Username, "file", fileName, "resource_id", resID, "share", share)
			c.JSON(http.StatusOK, Ok(uploadResponse{Merged: true, UploadID: uploadID, Filename: fileName, Size: fileSize, ShareCode: share, ResourceID: resID}, "ok"))
			return
		}

		// 分片参数校验
		if chunkIndexPtr == nil || totalChunksPtr == nil || *chunkIndexPtr < 0 || *totalChunksPtr <= 0 || *chunkIndexPtr >= *totalChunksPtr {
			c.JSON(http.StatusBadRequest, Fail[any]("分片参数错误", 400))
			return
		}
		chunkFolder := filepath.Join(cfg.ChunkDir, uploadID)
		if err := ensureDir(chunkFolder); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("准备分片目录失败", 500))
			return
		}
		chunkPath := filepath.Join(chunkFolder, fmt.Sprintf("%d", *chunkIndexPtr))

		if _, err := os.Stat(chunkPath); err == nil {
			chunkEntries, _ := os.ReadDir(chunkFolder)
			isComplete := int64(len(chunkEntries)) >= *totalChunksPtr
			c.JSON(http.StatusOK, Ok(uploadResponse{Skipped: true, Merged: isComplete, UploadID: uploadID, Filename: fileName, ChunkIndex: chunkIndexPtr, TotalChunks: totalChunksPtr, ChunkSize: chunkSizePtr}, "ok"))
			return
		}

		if err := c.SaveUploadedFile(fh, chunkPath); err != nil {
			c.JSON(http.StatusInternalServerError, Fail[any]("保存分片失败", 500))
			return
		}

		chunkFiles, _ := os.ReadDir(chunkFolder)
		hasAll := int64(len(chunkFiles)) >= *totalChunksPtr
		if hasAll {
			// 检查是否缺失分片
			missing := false
			for i := int64(0); i < *totalChunksPtr; i++ {
				if _, err := os.Stat(filepath.Join(chunkFolder, fmt.Sprintf("%d", i))); err != nil {
					missing = true
					break
				}
			}
			if !missing {
				mergedPath := filepath.Join(cfg.MergeDir, fmt.Sprintf("%s-%s", uploadID, fileName))
				if err := mergeChunks(chunkFolder, *totalChunksPtr, mergedPath); err != nil {
					c.JSON(http.StatusInternalServerError, Fail[any]("合并失败", 500))
					return
				}
				reg.Logger.Info("分片合并完成", "upload_id", uploadID, "file", fileName, "total", *totalChunksPtr)
				stat, err := os.Stat(mergedPath)
				if err != nil {
					c.JSON(http.StatusInternalServerError, Fail[any]("读取文件失败", 500))
					return
				}
				fileSize = stat.Size()
				hash, err := hashFile(mergedPath)
				if err != nil {
					c.JSON(http.StatusInternalServerError, Fail[any]("计算摘要失败", 500))
					return
				}
				objectKey := storage.BuildObjectKey(hash, fileName, time.Now())
				f, err := os.Open(mergedPath)
				if err != nil {
					c.JSON(http.StatusInternalServerError, Fail[any]("读取文件失败", 500))
					return
				}
				defer f.Close()
				storedPath, err := stg.Write(objectKey, f, -1, fileType)
				if err != nil {
					c.JSON(http.StatusInternalServerError, Fail[any]("存储失败", 500))
					return
				}
				resID, share, err := persistResource(c, store, model.Resource{Filename: fileName, Hash: hash, Type: fileType, Path: storedPath, FileSize: fileSize, UserID: user.ID})
				if err != nil {
					c.JSON(http.StatusInternalServerError, Fail[any]("记录失败", 500))
					return
				}
				reg.Logger.Info("分片上传完成", "user", user.Username, "file", fileName, "resource_id", resID, "share", share)
				_ = os.Remove(mergedPath)
				_ = os.RemoveAll(chunkFolder)
				c.JSON(http.StatusOK, Ok(uploadResponse{Merged: true, UploadID: uploadID, Filename: fileName, Size: fileSize, ShareCode: share, ResourceID: resID}, "ok"))
				return
			}
		}
		c.JSON(http.StatusOK, Ok(uploadResponse{Merged: false, UploadID: uploadID, Filename: fileName, ChunkIndex: chunkIndexPtr, TotalChunks: totalChunksPtr, ChunkSize: chunkSizePtr}, "ok"))
	}
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func cleanupChunks(cfg *config.Config) {
	total, err := dirSize(cfg.ChunkDir)
	if err != nil || total <= cfg.CleanLimit {
		return
	}
	now := time.Now()
	entries, err := os.ReadDir(cfg.ChunkDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		folder := filepath.Join(cfg.ChunkDir, entry.Name())
		files, err := os.ReadDir(folder)
		if err != nil {
			continue
		}
		for _, f := range files {
			info, err := f.Info()
			if err != nil {
				continue
			}
			if now.Sub(info.ModTime()) > cfg.CleanExpire {
				_ = os.Remove(filepath.Join(folder, f.Name()))
			}
		}
		remain, _ := os.ReadDir(folder)
		if len(remain) == 0 {
			_ = os.RemoveAll(folder)
		}
	}
}

func dirSize(dir string) (int64, error) {
	var total int64
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

func readAndHash(fh *multipart.FileHeader) (string, []byte, error) {
	file, err := fh.Open()
	if err != nil {
		return "", nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", nil, err
	}
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:]), data, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func mergeChunks(folder string, total int64, target string) error {
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return err
	}
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	for i := int64(0); i < total; i++ {
		chunkPath := filepath.Join(folder, fmt.Sprintf("%d", i))
		r, err := os.Open(chunkPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, r); err != nil {
			r.Close()
			return err
		}
		r.Close()
	}
	return nil
}

func firstValue(values []string, fallback string) string {
	if len(values) > 0 && values[0] != "" {
		return values[0]
	}
	return fallback
}

func parseInt64(value string, fallback int64) int64 {
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return n
	}
	return fallback
}

func parseOptionalInt64(value string) *int64 {
	if value == "" {
		return nil
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return &n
	}
	return nil
}

type guestUploadPolicy struct {
	maxBytes int64
	extSet   map[string]struct{}
}

func newGuestUploadPolicy(cfg *config.Config) *guestUploadPolicy {
	if !cfg.AppConfig.GuestUploadEnable {
		return nil
	}
	maxMb := cfg.AppConfig.GuestUploadMaxMbSize
	extSet := parseExtWhitelist(cfg.AppConfig.GuestUploadExtWhitelist)
	if maxMb <= 0 || len(extSet) == 0 {
		return nil
	}
	maxBytes := int64(maxMb) * 1024 * 1024
	return &guestUploadPolicy{maxBytes: maxBytes, extSet: extSet}
}

func parseExtWhitelist(raw string) map[string]struct{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	items := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '|', ' ', '\n', '\t', '\r':
			return true
		default:
			return false
		}
	})
	extSet := make(map[string]struct{})
	for _, item := range items {
		ext := normalizeExt(item)
		if ext == "" {
			continue
		}
		extSet[ext] = struct{}{}
	}
	if len(extSet) == 0 {
		return nil
	}
	return extSet
}

func normalizeExt(ext string) string {
	ext = strings.TrimSpace(strings.ToLower(ext))
	ext = strings.TrimPrefix(ext, ".")
	return ext
}

func (p *guestUploadPolicy) allow(fileName string, fileSize int64) (bool, string) {
	if p == nil {
		return false, "不允许访客上传"
	}
	ext := normalizeExt(filepath.Ext(fileName))
	if ext == "" {
		return false, "请登录后再进行上传"
	}
	if _, ok := p.extSet["*"]; !ok {
		if _, ok := p.extSet[ext]; !ok {
			return false, "请登录后再进行上传"
		}
	}
	if fileSize > p.maxBytes {
		return false, "文件大小超过限制"
	}
	return true, ""
}

func persistResource(c *gin.Context, store *db.DB, res model.Resource) (int64, string, error) {
	ctx, cancel := store.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	resID, err := store.Resource.Insert(ctx, res)
	if err != nil {
		return 0, "", err
	}
	share, err := store.Resource.CreateShareCode(ctx, resID, res.UserID, nil, nil)
	if err != nil {
		return 0, "", err
	}
	return resID, share.Code, nil
}
