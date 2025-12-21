package db

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"linkit/internal/config"
)

func TestAppConfigDAO_SetConfig_UpsertAndSync(t *testing.T) {
	t.Setenv("DATABASE_PATH", ":memory:")
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	store, err := NewStore(cfg, logger)
	if err != nil {
		t.Fatalf("NewStore 失败: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	liveCfg := cfg

	if err := store.AppConfig.SetConfig(ctx, &liveCfg, "STORAGE_DRIVER", "s3"); err != nil {
		t.Fatalf("SetConfig 失败: %v", err)
	}
	if liveCfg.AppConfig.StorageDriver != "s3" {
		t.Fatalf("期望 StorageDriver=s3，实际=%q", liveCfg.AppConfig.StorageDriver)
	}

	if err := store.AppConfig.SetConfig(ctx, &liveCfg, "STORAGE_DRIVER", "local"); err != nil {
		t.Fatalf("SetConfig(更新) 失败: %v", err)
	}
	items, err := store.AppConfig.GetConfigs(ctx)
	if err != nil {
		t.Fatalf("GetConfigs 失败: %v", err)
	}
	if items["STORAGE_DRIVER"] != "local" {
		t.Fatalf("期望 DB STORAGE_DRIVER=local，实际=%q", items["STORAGE_DRIVER"])
	}

	var count int
	if err := store.Client.QueryRowContext(ctx, `SELECT COUNT(1) FROM app_config WHERE "key" = ?`, "STORAGE_DRIVER").Scan(&count); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("期望仅 1 条记录，实际=%d", count)
	}

	synced := cfg
	if err := synced.Sync(ctx, store.AppConfig); err != nil {
		t.Fatalf("cfg.Sync 失败: %v", err)
	}
	if synced.AppConfig.StorageDriver != "local" {
		t.Fatalf("期望 Sync 后 StorageDriver=local，实际=%q", synced.AppConfig.StorageDriver)
	}
}
