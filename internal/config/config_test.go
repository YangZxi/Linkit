package config

import (
	"context"
	"testing"
)

type fakeAppConfigSource struct {
	items map[string]string
}

func (f fakeAppConfigSource) GetConfigs(ctx context.Context) (map[string]string, error) {
	return f.items, nil
}

func TestSync_AppConfigPriority_DBOverEnv(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "s3")
	cfg := Load()

	if err := cfg.Sync(context.Background(), fakeAppConfigSource{items: map[string]string{"STORAGE_DRIVER": "local"}}); err != nil {
		t.Fatalf("Sync 失败: %v", err)
	}
	if cfg.AppConfig.StorageDriver != "local" {
		t.Fatalf("期望 STORAGE_DRIVER=local，实际=%q", cfg.AppConfig.StorageDriver)
	}
}

func TestSync_AppConfigPriority_EnvOverHardcoded(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "s3")
	cfg := Load()

	if err := cfg.Sync(context.Background(), nil); err != nil {
		t.Fatalf("Sync 失败: %v", err)
	}
	if cfg.AppConfig.StorageDriver != "s3" {
		t.Fatalf("期望 STORAGE_DRIVER=s3，实际=%q", cfg.AppConfig.StorageDriver)
	}
}

func TestSetAppConfigValue_WhitelistOnly(t *testing.T) {
	var cfg Config
	if ok := cfg.SetAppConfigValue("PORT", "123"); ok {
		t.Fatalf("非白名单 key 不应生效")
	}
	if ok := cfg.SetAppConfigValue("storage_driver", "s3"); !ok {
		t.Fatalf("白名单 key 应该生效")
	}
	if cfg.AppConfig.StorageDriver != "s3" {
		t.Fatalf("期望 STORAGE_DRIVER=s3，实际=%q", cfg.AppConfig.StorageDriver)
	}
}

func TestSetAppConfigValue_GuestUploadConfig(t *testing.T) {
	var cfg Config
	if ok := cfg.SetAppConfigValue("GUEST_UPLOAD_EXT_WHITELIST", "jpg,png"); !ok {
		t.Fatalf("后缀白名单应该生效")
	}
	if cfg.AppConfig.GuestUploadExtWhitelist != "jpg,png" {
		t.Fatalf("期望 GUEST_UPLOAD_EXT_WHITELIST=jpg,png，实际=%q", cfg.AppConfig.GuestUploadExtWhitelist)
	}
	if ok := cfg.SetAppConfigValue("GUEST_UPLOAD_MAX_MB_SIZE", "8"); !ok {
		t.Fatalf("大小白名单应该生效")
	}
	if cfg.AppConfig.GuestUploadMaxMbSize != 8 {
		t.Fatalf("期望 GUEST_UPLOAD_MAX_MB_SIZE=8，实际=%d", cfg.AppConfig.GuestUploadMaxMbSize)
	}
	if ok := cfg.SetAppConfigValue("GUEST_UPLOAD_ENABLE", "true"); !ok {
		t.Fatalf("启用开关应该生效")
	}
	if !cfg.AppConfig.GuestUploadEnable {
		t.Fatalf("期望 GUEST_UPLOAD_ENABLE=true")
	}
}
