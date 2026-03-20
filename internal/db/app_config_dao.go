package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm/clause"
	"linkit/internal/config"
	"linkit/internal/db/model"
)

type AppConfigDao struct {
	store *DB
}

func (dao *AppConfigDao) getConfigs(ctx context.Context) (map[string]string, error) {
	var items []model.AppConfig
	if err := dao.store.Client.WithContext(ctx).Find(&items).Error; err != nil {
		return nil, err
	}

	out := make(map[string]string)
	for _, item := range items {
		key := item.Key
		key = strings.ToUpper(strings.TrimSpace(key))
		if key == "" {
			continue
		}
		if item.Value != nil {
			out[key] = *item.Value
		} else {
			out[key] = ""
		}
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
	val := value
	return dao.store.Client.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]any{"value": val}),
	}).Create(&model.AppConfig{
		Key:   key,
		Value: &val,
	}).Error
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
