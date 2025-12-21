package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"strings"

	"linkit/internal/config"
)

type AppConfigDao struct {
	store *DB
}

func (dao *AppConfigDao) getConfigs(ctx context.Context) (map[string]string, error) {
	rows, err := dao.store.Client.QueryContext(ctx, `SELECT "key", "value" FROM app_config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var key string
		var value sql.NullString
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		key = strings.ToUpper(strings.TrimSpace(key))
		if key == "" {
			continue
		}
		if value.Valid {
			out[key] = value.String
		} else {
			out[key] = ""
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetConfigs 从数据库读取所有配置项，返回 kv。
func (dao *AppConfigDao) GetConfigs(ctx context.Context) (map[string]string, error) {
	return dao.getConfigs(ctx)
}

// Sync 将数据库中的白名单配置(仅 AppConfig)同步到 cfg。
func (dao *AppConfigDao) Sync(ctx context.Context, cfg *config.Config) error {
	items, err := dao.getConfigs(ctx)
	if err != nil {
		return err
	}
	for k, v := range items {
		cfg.SetAppConfigValue(k, v)
	}
	return nil
}

func (dao *AppConfigDao) setConfig(ctx context.Context, key, value string) error {
	key = strings.ToUpper(strings.TrimSpace(key))
	_, err := dao.store.Client.ExecContext(ctx, `
INSERT INTO app_config("key", "value")
VALUES(?, ?)
ON CONFLICT("key") DO UPDATE SET "value" = excluded."value";
`, key, value)
	return err
}

// SetConfig 写入数据库并更新本地 cfg（仅允许白名单 AppConfig）。
func (dao *AppConfigDao) SetConfig(ctx context.Context, cfg *config.Config, key, value string) error {
	key = strings.ToUpper(strings.TrimSpace(key))
	if !config.IsAppConfigKey(key) {
		return fmt.Errorf("不支持的配置项: %s", key)
	}
	if err := dao.setConfig(ctx, key, value); err != nil {
		return err
	}
	if ok := cfg.SetAppConfigValue(key, value); !ok {
		return errors.New("配置项写入成功，但应用失败")
	}
	return nil
}
