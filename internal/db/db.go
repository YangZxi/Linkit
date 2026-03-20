package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"linkit/internal/config"
	"linkit/internal/db/model"
)

const (
	GuestUserID   int64 = 2
	GuestUsername       = "guest"
	guestEmail          = "guest@example.com"
)

type DB struct {
	Client    *gorm.DB
	Logger    *slog.Logger
	Cfg       config.Config
	AppConfig *AppConfigDao
	User      *UserDao
	Resource  *ResourceDao
	Share     *ShareDao
}

func NewStore(cfg config.Config, logger *slog.Logger, init bool) (*DB, error) {
	if err := ensureDir(cfg.DatabasePath); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, err
	}
	store := &DB{Client: db, Logger: logger, Cfg: cfg}
	store.Resource = NewResourceDao(store)
	store.User = &UserDao{store: store}
	store.AppConfig = &AppConfigDao{store: store}
	store.Share = &ShareDao{store: store}
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
	sqlDB, err := s.Client.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *DB) upgradeSchema(ctx context.Context) error {
	return s.Client.WithContext(ctx).AutoMigrate(
		&model.User{},
		&model.AppConfig{},
		&model.Resource{},
		&model.ResourceTag{},
		&model.Share{},
	)
}

func (s *DB) ensureAdmin(ctx context.Context) error {
	var user model.User
	err := s.Client.WithContext(ctx).Where("id = ?", s.Cfg.AdminUserId).First(&user).Error
	if err == nil {
		s.Logger.Info("管理员账户已存在", "id", user.ID, "username", user.Username, "email", s.Cfg.AdminEmail)
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	pwHash, err := bcrypt.GenerateFromPassword([]byte(s.Cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user = model.User{
		ID:       s.Cfg.AdminUserId,
		Username: s.Cfg.AdminUsername,
		Password: string(pwHash),
		Email:    s.Cfg.AdminEmail,
		Nickname: s.Cfg.AdminUsername,
	}
	err = s.Client.WithContext(ctx).Create(&user).Error
	if err == nil {
		s.Logger.Info("创建默认管理员账户", "username", s.Cfg.AdminUsername, "password", s.Cfg.AdminPassword, "email", s.Cfg.AdminEmail)
	}
	return err
}

func (s *DB) ensureGuest(ctx context.Context) error {
	var guestUser model.User
	err := s.Client.WithContext(ctx).Where("id = ?", GuestUserID).First(&guestUser).Error
	if err == nil {
		if guestUser.Username != GuestUsername {
			s.Logger.Warn("访客ID已被占用，访客上传将复用该用户", "guest_id", GuestUserID, "username", guestUser.Username)
			return nil
		}
		s.Logger.Info("访__客账户已存在", "id", GuestUserID, "username", GuestUsername)
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	var count int64
	if err := s.Client.WithContext(ctx).Model(&model.User{}).Where("username = ? OR email = ?", GuestUsername, guestEmail).Count(&count).Error; err != nil {
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
	guestUser = model.User{
		ID:       GuestUserID,
		Username: GuestUsername,
		Password: string(pwHash),
		Email:    guestEmail,
		Nickname: GuestUsername,
	}
	err = s.Client.WithContext(ctx).Create(&guestUser).Error
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
