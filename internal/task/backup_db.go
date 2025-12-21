package task

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"linkit/internal/config"
	"linkit/internal/storage"
)

const defaultDBName = "app.db"

// StartS3DBBackup 当使用 S3 存储驱动时，启动每日数据库备份任务。
func StartS3DBBackup(cfg config.Config, reg *storage.Registry) {
	if reg == nil || reg.DefaultDriver != storage.PlatformS3 {
		return
	}
	storage, ok := reg.Storages[storage.PlatformS3]
	if !ok {
		reg.Logger.Warn("S3 存储未初始化，跳过数据库备份")
		return
	}
	dbPath, ok := resolveDBPath(cfg.DatabasePath)
	if !ok {
		reg.Logger.Warn("数据库路径无效，跳过数据库备份", "path", cfg.DatabasePath)
		return
	}
	ctx := context.Background()

	logger := reg.Logger
	go func() {
		backupOnce(dbPath, storage, logger)
	}()
	go func() {
		logger.Info("启动 S3 数据库备份任务", "path", dbPath)
		for {
			wait := time.Until(nextMidnight(time.Now()))
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				logger.Info("数据库备份任务已停止")
				return
			case <-timer.C:
			}
			if err := backupOnce(dbPath, storage, logger); err != nil {
				logger.Error("数据库备份失败", "err", err)
			}
		}
	}()
}

func backupOnce(dbPath string, stg storage.Storage, logger *slog.Logger) error {
	file, err := os.Open(dbPath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("数据库路径是目录: %s", dbPath)
	}

	objectKey := buildBackupObjectKey(time.Now(), filepath.Base(dbPath))
	if _, err := stg.Write(objectKey, file, info.Size(), "application/octet-stream"); err != nil {
		return err
	}
	logger.Info("数据库已备份到 S3",
		"objectKey", objectKey,
		"size", fmt.Sprintf("%.2f KB", float64(info.Size())/1024.0))
	return nil
}

func buildBackupObjectKey(now time.Time, base string) string {
	name := strings.TrimSpace(base)
	if name == "" {
		name = defaultDBName
	}
	prefix := now.Format("2006_01_02_150405")
	filename := fmt.Sprintf("%s_%s", prefix, name)
	return path.Join("backup", filename)
}

func nextMidnight(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location()).Add(24 * time.Hour)
}

func resolveDBPath(dbPath string) (string, bool) {
	pathStr := strings.TrimSpace(dbPath)
	if pathStr == "" {
		return "", false
	}
	if strings.HasPrefix(pathStr, ":memory:") {
		return "", false
	}
	if after, ok := strings.CutPrefix(pathStr, "file:"); ok {
		pathStr = after
	}
	if idx := strings.Index(pathStr, "?"); idx >= 0 {
		pathStr = pathStr[:idx]
	}
	if pathStr == "" {
		return "", false
	}
	return filepath.Clean(pathStr), true
}
