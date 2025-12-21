package task

import (
	"testing"
	"time"
)

func TestResolveDBPathPlain(t *testing.T) {
	got, ok := resolveDBPath("./data/app.db")
	if !ok {
		t.Fatalf("期望解析成功")
	}
	if got == "" {
		t.Fatalf("期望得到有效路径")
	}
}

func TestResolveDBPathFileURI(t *testing.T) {
	got, ok := resolveDBPath("file:./data/app.db?cache=shared")
	if !ok {
		t.Fatalf("期望解析成功")
	}
	if got == "" {
		t.Fatalf("期望得到有效路径")
	}
}

func TestResolveDBPathMemory(t *testing.T) {
	if _, ok := resolveDBPath(":memory:"); ok {
		t.Fatalf("内存数据库不应参与备份")
	}
}

func TestBuildBackupObjectKeyFormat(t *testing.T) {
	now := time.Date(2025, 12, 21, 19, 56, 22, 0, time.UTC)
	key := buildBackupObjectKey(now, "app.db")
	expect := "backup/2025_12_21_195622_app.db"
	if key != expect {
		t.Fatalf("对象路径格式不正确: %q", key)
	}
}
