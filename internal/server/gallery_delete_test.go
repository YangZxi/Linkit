package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"linkit/internal/config"
	"linkit/internal/db"
	"linkit/internal/db/model"
	"linkit/internal/storage"
)

func TestGalleryDeleteHandler_删除资源并清理存储(t *testing.T) {
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

	admin, err := store.User.FindByCredential(context.Background(), cfg.AdminEmail)
	if err != nil || admin == nil {
		t.Fatalf("读取管理员失败 err=%v", err)
	}

	objectKey := "tests/delete.txt"
	absPath := filepath.Join(localRoot, filepath.FromSlash(objectKey))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("MkdirAll 失败: %v", err)
	}
	if err := os.WriteFile(absPath, []byte("delete"), 0o644); err != nil {
		t.Fatalf("WriteFile 失败: %v", err)
	}
	storedPath, err := storage.BuildStoredPath(storage.PlatformLocal, "", objectKey)
	if err != nil {
		t.Fatalf("BuildStoredPath 失败: %v", err)
	}

	ctx := context.Background()
	resourceID, err := store.Resource.Insert(ctx, model.Resource{
		Filename: "delete.txt",
		Hash:     "hash",
		Type:     "text/plain",
		Path:     storedPath,
		UserID:   admin.ID,
	})
	if err != nil {
		t.Fatalf("插入资源失败: %v", err)
	}
	if _, err := store.Client.ExecContext(ctx, `INSERT INTO share_code(resource_id, code, user_id, created_at) VALUES(?,?,?, CURRENT_TIMESTAMP)`, resourceID, "DEL123", admin.ID); err != nil {
		t.Fatalf("插入短链失败: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user", admin)
		c.Next()
	})
	r.POST("/gallery/delete", GalleryDeleteHandler(store, reg))

	body, _ := json.Marshal(gin.H{"id": resourceID})
	req := httptest.NewRequest(http.MethodPost, "/gallery/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Success bool `json:"success"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Code != 200 || !resp.Data.Success {
		t.Fatalf("期望删除成功，code=%d success=%v", resp.Code, resp.Data.Success)
	}

	if _, err := os.Stat(absPath); err == nil || !os.IsNotExist(err) {
		t.Fatalf("期望文件已删除，err=%v", err)
	}

	var count int
	if err := store.Client.QueryRowContext(ctx, `SELECT COUNT(1) FROM resource WHERE id = ?`, resourceID).Scan(&count); err != nil {
		t.Fatalf("查询资源失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("期望资源已删除，实际 count=%d", count)
	}
	if err := store.Client.QueryRowContext(ctx, `SELECT COUNT(1) FROM share_code WHERE resource_id = ?`, resourceID).Scan(&count); err != nil {
		t.Fatalf("查询短链失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("期望短链已删除，实际 count=%d", count)
	}
}
