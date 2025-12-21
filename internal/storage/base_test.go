package storage

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestBuildObjectKey_UTF8Safe(t *testing.T) {
	now := time.Date(2025, 12, 21, 0, 0, 0, 0, time.UTC)
	key := BuildObjectKey("hash", "1105-隐身人机.mp4", now)
	if !utf8.ValidString(key) {
		t.Fatalf("生成的对象路径不是有效 UTF-8: %q", key)
	}
	if !strings.Contains(key, "隐身人机.mp4") {
		t.Fatalf("对象路径未包含完整文件名片段: %q", key)
	}
}

func TestBuildObjectKey_TruncateRunes(t *testing.T) {
	now := time.Date(2025, 12, 21, 0, 0, 0, 0, time.UTC)
	key := BuildObjectKey("hash", "这是一个非常非常长的文件名示例.mp4", now)
	if !utf8.ValidString(key) {
		t.Fatalf("生成的对象路径不是有效 UTF-8: %q", key)
	}
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		t.Fatalf("对象路径格式错误: %q", key)
	}
	name := strings.TrimPrefix(parts[1], "hash-")
	if !strings.HasSuffix(name, ".mp4") {
		t.Fatalf("对象路径未保留后缀: %q", key)
	}
	base := strings.TrimSuffix(name, ".mp4")
	if utf8.RuneCountInString(base) != 10 {
		t.Fatalf("文件名截断长度不正确: %q", base)
	}
}
