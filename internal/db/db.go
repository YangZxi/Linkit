package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"linkit/internal/config"
)

const (
	GuestUserID   int64 = 2
	GuestUsername       = "guest"
	guestEmail          = "guest@example.com"
	guestNickname       = "访客"
)

const (
	createUserTable = `
CREATE TABLE IF NOT EXISTS "user" (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  nickname TEXT NOT NULL,
  token TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
	createAppConfigTable = `
CREATE TABLE IF NOT EXISTS "app_config" (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  "key" TEXT NOT NULL UNIQUE,
  "value" TEXT
);
`
	createResourceTable = `
CREATE TABLE IF NOT EXISTS "resource" (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  filename TEXT NOT NULL,
  hash TEXT NOT NULL,
  type TEXT NOT NULL,
  path TEXT NOT NULL,
  file_size INTEGER NOT NULL DEFAULT 0,
  user_id INTEGER NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
	createShareTable = `
CREATE TABLE IF NOT EXISTS "share" (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  resource_id INTEGER NOT NULL,
  code TEXT NOT NULL UNIQUE,
  user_id INTEGER NOT NULL,
  password TEXT,
  expire_time DATETIME,
  view_count INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
)

type DB struct {
	Client    *sql.DB
	Logger    *slog.Logger
	Cfg       config.Config
	AppConfig *AppConfigDao
	User      *UserDao
	Resource  *ResourceDao
}

func NewStore(cfg config.Config, logger *slog.Logger, init bool) (*DB, error) {
	if err := ensureDir(cfg.DatabasePath); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		return nil, err
	}
	store := &DB{Client: db, Logger: logger, Cfg: cfg}
	store.Resource = &ResourceDao{store: store}
	store.User = &UserDao{store: store}
	store.AppConfig = &AppConfigDao{store: store}
	if init {
		if err := store.upgradeSchema(context.Background()); err != nil {
			return nil, err
		}
		if err := store.ensureAdmin(context.Background()); err != nil {
			return nil, err
		}
		if err := store.ensureGuest(context.Background()); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func ensureDir(dbPath string) error {
	if strings.HasPrefix(dbPath, ":memory:") {
		return nil
	}
	path := dbPath
	if strings.HasPrefix(dbPath, "file:") {
		path = strings.TrimPrefix(dbPath, "file:")
	}
	clean := filepath.Clean(path)
	dir := filepath.Dir(clean)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func (s *DB) Close() error {
	return s.Client.Close()
}

func (s *DB) upgradeSchema(ctx context.Context) error {
	stmts := []string{createUserTable, createAppConfigTable, createResourceTable, createShareTable}
	for _, stmt := range stmts {
		if _, err := s.Client.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if err := s.ensureColumn(ctx, "share", "password", "password TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn(ctx, "share", "expire_time", "expire_time DATETIME"); err != nil {
		return err
	}
	return nil
}

func (s *DB) ensureColumn(ctx context.Context, table, column, columnDef string) error {
	exists, err := s.columnExists(ctx, table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = s.Client.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN %s`, table, columnDef))
	return err
}

func (s *DB) columnExists(ctx context.Context, table, column string) (bool, error) {
	rows, err := s.Client.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info("%s");`, table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func (s *DB) ensureAdmin(ctx context.Context) error {
	var count int
	err := s.Client.QueryRowContext(ctx, "SELECT COUNT(1) FROM user WHERE email = ?", s.Cfg.AdminEmail).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		s.Logger.Info("管理员账户已存在", "email", s.Cfg.AdminEmail)
		return nil
	}
	pwHash, err := bcrypt.GenerateFromPassword([]byte(s.Cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.Client.ExecContext(ctx, `INSERT INTO user(username, password, email, nickname) VALUES(?,?,?,?)`, s.Cfg.AdminUsername, string(pwHash), s.Cfg.AdminEmail, s.Cfg.AdminUsername)
	if err == nil {
		s.Logger.Info("创建默认管理员账户", "email", s.Cfg.AdminEmail)
	}
	return err
}

func (s *DB) ensureGuest(ctx context.Context) error {
	var username string
	err := s.Client.QueryRowContext(ctx, "SELECT username FROM user WHERE id = ?", GuestUserID).Scan(&username)
	// guest user exists
	if err == nil {
		if username != GuestUsername {
			s.Logger.Warn("访客ID已被占用，访客上传将复用该用户", "guest_id", GuestUserID, "username", username)
			return nil
		}
		s.Logger.Info("访客账户已存在", "id", GuestUserID, "username", GuestUsername)
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}

	var count int
	if err := s.Client.QueryRowContext(ctx, "SELECT COUNT(1) FROM user WHERE username = ? OR email = ?", GuestUsername, guestEmail).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		s.Logger.Warn("访客账户已存在但ID不匹配", "guest_id", GuestUserID)
		return nil
	}

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		s.Logger.Error("生成访客密码失败", "error", err)
		return err
	}
	secret := hex.EncodeToString(buf)
	pwHash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.Client.ExecContext(ctx, `INSERT INTO user(id, username, password, email, nickname) VALUES(?,?,?,?,?)`, GuestUserID, GuestUsername, string(pwHash), guestEmail, guestNickname)
	if err == nil {
		s.Logger.Info("创建访客账户", "id", GuestUserID, "username", GuestUsername)
	}
	return err
}

func (s *DB) WithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, d)
}

// 为了与前端分页兼容，提供安全页数
func SafePage(page int) int {
	return int(math.Max(1, float64(page)))
}
