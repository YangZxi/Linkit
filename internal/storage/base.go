package storage

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"path/filepath"
	"strings"
	"time"

	"linkit/internal/config"
)

type BucketPlatform string

const (
	PlatformLocal BucketPlatform = "local"
	PlatformS3    BucketPlatform = "s3"
)

type Storage interface {
	Platform() BucketPlatform
	Write(objectKey string, r io.Reader, size int64, contentType string) (string, error)
	GetURL(storedPath string, expires time.Duration) (string, error)
	Delete(storedPath string) error
}

type Registry struct {
	DefaultDriver BucketPlatform
	Storages      map[BucketPlatform]Storage
	Logger        *slog.Logger
}

func NormalizeDriver(driver string) (BucketPlatform, error) {
	switch strings.ToLower(driver) {
	case "local", "":
		return PlatformLocal, nil
	case "s3", "cloudflare":
		return PlatformS3, nil
	default:
		return "", fmt.Errorf("无效的存储驱动: %s", driver)
	}
}

func BuildObjectKey(hash, filename string, now time.Time) string {
	year := now.Year()
	month := int(now.Month())
	ext := path.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	if strings.TrimSpace(base) == "" {
		base = "file"
	}
	shortBase := base
	if len([]rune(shortBase)) > 10 {
		// 按 rune 截断，避免破坏 UTF-8
		shortBase = string([]rune(shortBase)[:10])
	}
	shortName := shortBase + ext
	return fmt.Sprintf("%d-%02d/%s-%s", year, month, hash, shortName)
}

func NormalizeObjectKey(key string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(key, "\\", "/"))
	clean = strings.TrimLeft(clean, "/")
	if strings.Contains(clean, "..") { // 保守防护
		return "", errors.New("存储路径非法")
	}
	return clean, nil
}

func BuildStoredPath(platform BucketPlatform, bucket, objectKey string) (string, error) {
	key, err := NormalizeObjectKey(objectKey)
	if err != nil {
		return "", err
	}
	if platform == PlatformLocal {
		return fmt.Sprintf("local@/%s", key), nil
	}
	return fmt.Sprintf("%s:%s@/%s", platform, bucket, key), nil
}

func ParseStoredPath(storedPath string) (platform BucketPlatform, bucket string, key string, err error) {
	if storedPath == "" {
		return "", "", "", errors.New("空存储路径")
	}
	if strings.HasPrefix(storedPath, "local@/") {
		key = strings.TrimPrefix(storedPath, "local@/")
		key, err = NormalizeObjectKey(key)
		return PlatformLocal, "", key, err
	}
	parts := strings.SplitN(storedPath, "@/", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("存储路径格式错误")
	}
	prefix := parts[0]
	key, err = NormalizeObjectKey(parts[1])
	if err != nil {
		return "", "", "", err
	}
	prefixParts := strings.SplitN(prefix, ":", 2)
	if len(prefixParts) != 2 {
		return "", "", "", fmt.Errorf("存储路径格式错误")
	}
	p, err := NormalizeDriver(prefixParts[0])
	if err != nil {
		return "", "", "", err
	}
	return p, prefixParts[1], key, nil
}

func SetupRegistry(cfg config.Config, logger *slog.Logger) (*Registry, error) {
	platform, err := NormalizeDriver(cfg.AppConfig.StorageDriver)
	if err != nil {
		return nil, err
	}
	reg := &Registry{DefaultDriver: platform, Storages: make(map[BucketPlatform]Storage), Logger: logger}

	local, err := NewLocal(cfg.LocalRoot)
	if err != nil {
		return nil, err
	}
	reg.Storages[PlatformLocal] = local

	if platform == PlatformS3 || (cfg.AppConfig.S3Bucket != "" && cfg.AppConfig.S3AccessKey != "" && cfg.AppConfig.S3SecretKey != "" && cfg.AppConfig.S3Endpoint != "") {
		s3, err := NewS3(cfg, logger)
		if err != nil {
			return nil, err
		}
		reg.Storages[PlatformS3] = s3
	}
	if platform == PlatformS3 {
		if _, ok := reg.Storages[PlatformS3]; !ok {
			return nil, fmt.Errorf("已选择 S3 存储驱动，但缺少必要配置")
		}
	}
	logger.Info(fmt.Sprintf("初始化 Storage 成功，%s", platform))
	return reg, nil
}

func (r *Registry) Active() Storage {
	return r.Storages[r.DefaultDriver]
}

func (r *Registry) ByStoredPath(path string) (Storage, error) {
	plat, _, _, err := ParseStoredPath(path)
	if err != nil {
		return nil, err
	}
	s, ok := r.Storages[plat]
	if !ok {
		return nil, fmt.Errorf("未找到存储驱动: %s", plat)
	}
	return s, nil
}

// 基于扩展名推断 MIME
func GuessMime(filename string) string {
	lower := strings.ToLower(filepath.Ext(filename))
	switch lower {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".pdf":
		return "application/pdf"
	case ".txt", ".md", ".json", ".log", ".xml", ".csv", ".html", ".css", ".js", ".ts":
		return "text/plain"
	}
	return "application/octet-stream"
}
