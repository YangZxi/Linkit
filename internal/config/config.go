package config

import (
	"context"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AppConfig struct {
	StorageDriver string `config:"STORAGE_DRIVER"`
	// s3 config
	S3Bucket    string `config:"S3_BUCKET"`
	S3AccessKey string `config:"S3_ACCESS_KEY"`
	S3SecretKey string `config:"S3_SECRET_KEY"`
	S3Endpoint  string `config:"S3_ENDPOINT"`
	S3Region    string `config:"S3_REGION"`
}

type Config struct {
	Port           int
	FrontendOrigin string
	DatabasePath   string
	LocalRoot      string
	SessionCookie  string
	CookieMaxAge   time.Duration
	CookieSecure   bool
	ChunkDir       string
	MergeDir       string
	MaxFileSize    int64
	ChunkThreshold int64
	CleanLimit     int64
	CleanExpire    time.Duration
	AdminUsername  string
	AdminPassword  string
	AdminEmail     string
	LogLevel       string
	AppConfig      AppConfig
}

type AppConfigDao interface {
	GetConfigs(ctx context.Context) (map[string]string, error)
}

var (
	appConfigKeyOnce  sync.Once
	appConfigKeyIndex map[string][]int
)

func getAppConfigKeyIndex() map[string][]int {
	appConfigKeyOnce.Do(func() {
		appConfigKeyIndex = make(map[string][]int)
		t := reflect.TypeOf(AppConfig{})
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			key := strings.ToUpper(strings.TrimSpace(f.Tag.Get("config")))
			if key == "" {
				continue
			}
			appConfigKeyIndex[key] = f.Index
		}
	})
	return appConfigKeyIndex
}

// AppConfigKeys 返回所有 AppConfig 白名单 key（来自结构体 tag），并按字母序排序。
func AppConfigKeys() []string {
	index := getAppConfigKeyIndex()
	keys := make([]string, 0, len(index))
	for k := range index {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func Load() Config {
	cfg := Config{
		Port:           getInt("PORT", 3301),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "*"),
		DatabasePath:   getEnv("DATABASE_PATH", "./data/app.db"),
		LocalRoot:      getEnv("LOCAL_STORAGE_ROOT", "./data/storage"),
		ChunkDir:       getEnv("CHUNK_DIR", "./data/temp/chunk"),
		MergeDir:       getEnv("MERGE_DIR", "./data/temp/merged"),

		SessionCookie:  getEnv("SESSION_COOKIE", "session_token"),
		CookieMaxAge:   time.Hour * 24 * 30,
		CookieSecure:   getEnv("COOKIE_SECURE", "false") == "true",
		MaxFileSize:    1 << 30,                // 1GB
		ChunkThreshold: 100 * 1024 * 1024,      // 100MB
		CleanLimit:     2 * 1024 * 1024 * 1024, // 2GB
		CleanExpire:    30 * time.Minute,

		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "123123"),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@example.com"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
	}

	return cfg
}

func IsAppConfigKey(key string) bool {
	key = strings.ToUpper(strings.TrimSpace(key))
	_, ok := getAppConfigKeyIndex()[key]
	return ok
}

// SetAppConfigValue 仅对 AppConfig 白名单 key 生效；非白名单 key 会被忽略。
func (cfg *Config) SetAppConfigValue(key, value string) bool {
	key = strings.ToUpper(strings.TrimSpace(key))
	index, ok := getAppConfigKeyIndex()[key]
	if !ok {
		return false
	}
	v := reflect.ValueOf(&cfg.AppConfig).Elem()
	f := v.FieldByIndex(index)
	if !f.IsValid() || !f.CanSet() {
		return false
	}

	switch f.Kind() {
	case reflect.String:
		f.SetString(value)
		return true
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return false
		}
		f.SetBool(b)
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// 若未来 AppConfig 有时长字段，可在此扩展（例如识别 time.Duration）
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return false
		}
		f.SetInt(i)
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return false
		}
		f.SetUint(u)
		return true
	case reflect.Float32, reflect.Float64:
		fl, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return false
		}
		f.SetFloat(fl)
		return true
	default:
		return false
	}
}

// GetAppConfigValue 获取 AppConfig 白名单配置的当前值（用于展示/回显）。
func (cfg *Config) GetAppConfigValue(key string) (string, bool) {
	key = strings.ToUpper(strings.TrimSpace(key))
	index, ok := getAppConfigKeyIndex()[key]
	if !ok {
		return "", false
	}
	v := reflect.ValueOf(cfg.AppConfig)
	f := v.FieldByIndex(index)
	if !f.IsValid() {
		return "", false
	}
	switch f.Kind() {
	case reflect.String:
		return f.String(), true
	case reflect.Bool:
		return strconv.FormatBool(f.Bool()), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(f.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(f.Uint(), 10), true
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(f.Float(), 'f', -1, 64), true
	default:
		return "", false
	}
}

// Sync 用于在 Load() 之后同步 AppConfig。
// 优先级：数据库 > env > 硬编码。
func (cfg *Config) Sync(ctx context.Context, dao AppConfigDao) error {
	cfg.AppConfig = AppConfig{
		StorageDriver: getEnv("STORAGE_DRIVER", "local"),
		S3Bucket:      os.Getenv("S3_BUCKET"),
		S3AccessKey:   os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:   os.Getenv("S3_SECRET_KEY"),
		S3Endpoint:    os.Getenv("S3_ENDPOINT"),
		S3Region:      getEnv("S3_REGION", "auto"),
	}
	if dao == nil {
		return nil
	}
	items, err := dao.GetConfigs(ctx)
	if err != nil {
		return err
	}
	for k, v := range items {
		cfg.SetAppConfigValue(k, v)
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
