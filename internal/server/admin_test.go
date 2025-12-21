package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"linkit/internal/config"
	"linkit/internal/db"
)

func TestAdminUpsertConfigHandler_白名单校验与写入(t *testing.T) {
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

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/admin/config", AdminUpsertConfigHandler(store, &cfg))

	body, _ := json.Marshal(gin.H{"items": []gin.H{{"key": "STORAGE_DRIVER", "value": "local"}}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	items, err := store.AppConfig.GetConfigs(context.Background())
	if err != nil {
		t.Fatalf("GetConfigs 失败: %v", err)
	}
	if items["STORAGE_DRIVER"] != "local" {
		t.Fatalf("期望 DB STORAGE_DRIVER=local，实际=%q", items["STORAGE_DRIVER"])
	}
	if cfg.AppConfig.StorageDriver != "local" {
		t.Fatalf("期望 cfg.StorageDriver=local，实际=%q", cfg.AppConfig.StorageDriver)
	}

	body2, _ := json.Marshal(gin.H{"key": "NOT_ALLOWED", "value": "x"})
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/admin/config", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，实际=%d body=%s", w2.Code, w2.Body.String())
	}
}

func TestAdminChangePasswordHandler_校验原密码并更新(t *testing.T) {
	t.Setenv("DATABASE_PATH", ":memory:")
	t.Setenv("ADMIN_PASSWORD", "oldpass")
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

	admin, err := store.User.FindByCredential(context.Background(), cfg.AdminEmail)
	if err != nil || admin == nil {
		t.Fatalf("读取管理员失败 err=%v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user", admin)
		c.Next()
	})
	r.POST("/admin/password", AdminChangePasswordHandler(store, cfg))

	body, _ := json.Marshal(gin.H{
		"oldPassword":  "oldpass",
		"newPassword":  "newpass",
		"newPassword2": "newpass",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	updated, err := store.User.FindByCredential(context.Background(), cfg.AdminEmail)
	if err != nil || updated == nil {
		t.Fatalf("读取更新后管理员失败 err=%v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("newpass")); err != nil {
		t.Fatalf("期望新密码生效")
	}
}
