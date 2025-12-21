package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type LocalStorage struct {
	root string
}

func NewLocal(root string) (*LocalStorage, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &LocalStorage{root: root}, nil
}

func (l *LocalStorage) Platform() BucketPlatform {
	return PlatformLocal
}

func (l *LocalStorage) Write(objectKey string, r io.Reader, size int64, contentType string) (string, error) {
	normalized, err := NormalizeObjectKey(objectKey)
	if err != nil {
		return "", err
	}
	target := filepath.Join(l.root, filepath.FromSlash(normalized))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("local@/%s", normalized), nil
}

func (l *LocalStorage) GetURL(storedPath string, expires time.Duration) (string, error) {
	_, _, key, err := ParseStoredPath(storedPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(l.root, filepath.FromSlash(key)), nil
}

func (l *LocalStorage) Delete(storedPath string) error {
	platform, _, key, err := ParseStoredPath(storedPath)
	if err != nil {
		return err
	}
	if platform != PlatformLocal {
		return fmt.Errorf("存储路径与本地存储不匹配")
	}
	target := filepath.Join(l.root, filepath.FromSlash(key))
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
