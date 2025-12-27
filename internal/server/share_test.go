package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"linkit/internal/config"
	"linkit/internal/db"
	"linkit/internal/db/model"
	"linkit/internal/storage"
)

func TestDownloadHandler_本地文件启用协商缓存_命中ETag返回304(t *testing.T) {
	t.Setenv("DATABASE_PATH", ":memory:")
	t.Setenv("STORAGE_DRIVER", "local")
	localRoot := t.TempDir()
	t.Setenv("LOCAL_STORAGE_ROOT", localRoot)

	cfg := config.Load()
	if err := cfg.Sync(context.Background(), nil); err != nil {
		t.Fatalf("cfg.Sync 失败: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	store, err := db.NewStore(cfg, logger)
	if err != nil {
		t.Fatalf("NewStore 失败: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	reg, err := storage.SetupRegistry(cfg, logger)
	if err != nil {
		t.Fatalf("SetupRegistry 失败: %v", err)
	}

	objectKey := "tests/hello.txt"
	absPath := filepath.Join(localRoot, filepath.FromSlash(objectKey))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("MkdirAll 失败: %v", err)
	}
	content := []byte("hello")
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile 失败: %v", err)
	}
	storedPath, err := storage.BuildStoredPath(storage.PlatformLocal, "", objectKey)
	if err != nil {
		t.Fatalf("BuildStoredPath 失败: %v", err)
	}

	ctx := context.Background()
	resourceID, err := store.Resource.Insert(ctx, model.Resource{
		Filename: "hello.txt",
		Hash:     "hash",
		Type:     "text/plain",
		Path:     storedPath,
		UserID:   1,
	})
	if err != nil {
		t.Fatalf("插入资源失败: %v", err)
	}
	code := "ABC123"
	if _, err := store.Client.ExecContext(ctx, `INSERT INTO share(resource_id, code, user_id, created_at) VALUES(?,?,?, CURRENT_TIMESTAMP)`, resourceID, code, 1); err != nil {
		t.Fatalf("插入短链失败: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/r/:code", DownloadHandler(store, reg))

	// 第一次请求：应返回 200 + 缓存头
	req := httptest.NewRequest(http.MethodGet, "/r/"+code, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	resp := w.Result()
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != string(content) {
		t.Fatalf("响应体不匹配，期望=%q 实际=%q", string(content), string(body))
	}
	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Fatalf("期望返回 ETag")
	}
	if got := resp.Header.Get("Cache-Control"); !strings.Contains(got, "must-revalidate") {
		t.Fatalf("期望 Cache-Control 包含 must-revalidate，实际=%q", got)
	}
	if resp.Header.Get("Last-Modified") == "" {
		t.Fatalf("期望返回 Last-Modified")
	}

	// 第二次请求：携带 If-None-Match，应返回 304
	req2 := httptest.NewRequest(http.MethodGet, "/r/"+code, nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	resp2 := w2.Result()
	t.Cleanup(func() { _ = resp2.Body.Close() })

	if resp2.StatusCode != http.StatusNotModified {
		t.Fatalf("期望 304，实际=%d", resp2.StatusCode)
	}
	body2, _ := io.ReadAll(resp2.Body)
	if len(body2) != 0 {
		t.Fatalf("304 不应返回响应体，实际=%q", string(body2))
	}
	if resp2.Header.Get("ETag") == "" {
		t.Fatalf("304 仍应返回 ETag 供缓存使用")
	}

	var viewCount int64
	if err := store.Client.QueryRowContext(ctx, `SELECT view_count FROM share WHERE code = ?`, code).Scan(&viewCount); err != nil {
		t.Fatalf("读取 view_count 失败: %v", err)
	}
	if viewCount != 2 {
		t.Fatalf("view_count 期望=2 实际=%d", viewCount)
	}
}

func TestDownloadHandler_本地文件_命中IfModifiedSince返回304(t *testing.T) {
	t.Setenv("DATABASE_PATH", ":memory:")
	t.Setenv("STORAGE_DRIVER", "local")
	localRoot := t.TempDir()
	t.Setenv("LOCAL_STORAGE_ROOT", localRoot)

	cfg := config.Load()
	if err := cfg.Sync(context.Background(), nil); err != nil {
		t.Fatalf("cfg.Sync 失败: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	store, err := db.NewStore(cfg, logger)
	if err != nil {
		t.Fatalf("NewStore 失败: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	reg, err := storage.SetupRegistry(cfg, logger)
	if err != nil {
		t.Fatalf("SetupRegistry 失败: %v", err)
	}

	objectKey := "tests/ims.txt"
	absPath := filepath.Join(localRoot, filepath.FromSlash(objectKey))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("MkdirAll 失败: %v", err)
	}
	if err := os.WriteFile(absPath, []byte("ims"), 0o644); err != nil {
		t.Fatalf("WriteFile 失败: %v", err)
	}
	storedPath, err := storage.BuildStoredPath(storage.PlatformLocal, "", objectKey)
	if err != nil {
		t.Fatalf("BuildStoredPath 失败: %v", err)
	}

	ctx := context.Background()
	resourceID, err := store.Resource.Insert(ctx, model.Resource{
		Filename: "ims.txt",
		Hash:     "hash2",
		Type:     "text/plain",
		Path:     storedPath,
		UserID:   1,
	})
	if err != nil {
		t.Fatalf("插入资源失败: %v", err)
	}
	code := "DEF456"
	if _, err := store.Client.ExecContext(ctx, `INSERT INTO share(resource_id, code, user_id, created_at) VALUES(?,?,?, CURRENT_TIMESTAMP)`, resourceID, code, 1); err != nil {
		t.Fatalf("插入短链失败: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/r/:code", DownloadHandler(store, reg))

	// 先请求一次，拿到 Last-Modified
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/r/"+code, nil))
	resp := w.Result()
	t.Cleanup(func() { _ = resp.Body.Close() })
	lastMod := resp.Header.Get("Last-Modified")
	if lastMod == "" {
		t.Fatalf("期望返回 Last-Modified")
	}

	// 触发 If-Modified-Since：应返回 304
	req2 := httptest.NewRequest(http.MethodGet, "/r/"+code, nil)
	req2.Header.Set("If-Modified-Since", lastMod)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotModified {
		t.Fatalf("期望 304，实际=%d", w2.Code)
	}
}

func TestShareInfoHandler_分享密码校验(t *testing.T) {
	t.Setenv("DATABASE_PATH", ":memory:")

	cfg := config.Load()
	if err := cfg.Sync(context.Background(), nil); err != nil {
		t.Fatalf("cfg.Sync 失败: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	store, err := db.NewStore(cfg, logger)
	if err != nil {
		t.Fatalf("NewStore 失败: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	resourceID, err := store.Resource.Insert(ctx, model.Resource{
		Filename: "pwd.txt",
		Hash:     "hash",
		Type:     "text/plain",
		Path:     "local://tests/pwd.txt",
		UserID:   1,
	})
	if err != nil {
		t.Fatalf("插入资源失败: %v", err)
	}
	password := "pass1234"
	code := "PWD123"
	if _, err := store.Client.ExecContext(ctx, `INSERT INTO share(resource_id, code, user_id, password, created_at) VALUES(?,?,?,?, CURRENT_TIMESTAMP)`, resourceID, code, 1, password); err != nil {
		t.Fatalf("插入短链失败: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/share/:code", ShareInfoHandler(store))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/share/"+code, nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，实际=%d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/api/share/"+code+"?pwd=wrong", nil))
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，实际=%d body=%s", w2.Code, w2.Body.String())
	}

	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/api/share/"+code+"?pwd="+password, nil))
	if w3.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w3.Code, w3.Body.String())
	}
}

func TestDownloadHandler_分享过期返回410(t *testing.T) {
	t.Setenv("DATABASE_PATH", ":memory:")
	t.Setenv("STORAGE_DRIVER", "local")
	localRoot := t.TempDir()
	t.Setenv("LOCAL_STORAGE_ROOT", localRoot)

	cfg := config.Load()
	if err := cfg.Sync(context.Background(), nil); err != nil {
		t.Fatalf("cfg.Sync 失败: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	store, err := db.NewStore(cfg, logger)
	if err != nil {
		t.Fatalf("NewStore 失败: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	reg, err := storage.SetupRegistry(cfg, logger)
	if err != nil {
		t.Fatalf("SetupRegistry 失败: %v", err)
	}

	ctx := context.Background()
	resourceID, err := store.Resource.Insert(ctx, model.Resource{
		Filename: "expired.txt",
		Hash:     "hash-exp",
		Type:     "text/plain",
		Path:     "local://tests/expired.txt",
		UserID:   1,
	})
	if err != nil {
		t.Fatalf("插入资源失败: %v", err)
	}
	code := "EXP123"
	expireAt := time.Now().Add(-time.Hour)
	if _, err := store.Client.ExecContext(ctx, `INSERT INTO share(resource_id, code, user_id, expire_time, created_at) VALUES(?,?,?,?, CURRENT_TIMESTAMP)`, resourceID, code, 1, expireAt); err != nil {
		t.Fatalf("插入短链失败: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/r/:code", DownloadHandler(store, reg))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/r/"+code, nil))
	if w.Code != http.StatusGone {
		t.Fatalf("期望 410，实际=%d body=%s", w.Code, w.Body.String())
	}
}
