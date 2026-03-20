package db

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"linkit/internal/db/model"
)

type ShareDao struct {
	store *DB
}

func randomCode(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := range buf {
		buf[i] = letters[int(buf[i])%len(letters)]
	}
	return string(buf), nil
}

func (s *ShareDao) CreateShareCode(ctx context.Context, resourceID int64, userID int64, password *string, expireTime *time.Time, relay bool) (*model.ShareCode, error) {
	for i := 0; i < 5; i++ {
		code, err := randomCode(6)
		if err != nil {
			return nil, err
		}
		share := model.Share{
			ResourceID: resourceID,
			UserID:     userID,
			Code:       code,
			Relay:      relay,
		}
		if password != nil && strings.TrimSpace(*password) != "" {
			share.Password = password
		}
		if expireTime != nil {
			share.ExpireTime = expireTime
		}
		if err := s.store.Client.WithContext(ctx).Clauses(clause.Returning{}).Create(&share).Error; err != nil {
			if isUniqueConstraintError(err) {
				continue
			}
			return nil, err
		}
		return &model.ShareCode{
			ID:         share.ID,
			ResourceID: share.ResourceID,
			UserID:     share.UserID,
			Code:       share.Code,
			ViewCount:  share.ViewCount,
			Relay:      share.Relay,
			CreatedAt:  share.CreatedAt,
		}, nil
	}
	return nil, fmt.Errorf("生成短链失败")
}

func (s *ShareDao) GetShareByCode(ctx context.Context, code string) (*model.ShareResource, error) {
	var share model.Share
	err := s.store.Client.WithContext(ctx).Preload("Resource").Where("code = ?", code).First(&share).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &model.ShareResource{
		ShareID:    share.ID,
		Code:       share.Code,
		ResourceID: share.ResourceID,
		UserID:     share.UserID,
		Filename:   share.Resource.Filename,
		Path:       share.Resource.Path,
		Type:       share.Resource.Type,
		Relay:      share.Relay,
		ViewCount:  share.ViewCount,
		CreatedAt:  share.Resource.CreatedAt,
		Password:   share.Password,
		ExpireTime: share.ExpireTime,
	}, nil
}

func (s *ShareDao) IncrementShareViewCount(ctx context.Context, shareID int64) error {
	return s.store.Client.WithContext(ctx).
		Model(&model.Share{}).
		Where("id = ?", shareID).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

func (s *ShareDao) GetTotalViewCount(ctx context.Context) (int64, error) {
	var result struct {
		Total int64 `gorm:"column:total"`
	}
	if err := s.store.Client.WithContext(ctx).
		Model(&model.Share{}).
		Select("COALESCE(SUM(view_count), 0) AS total").
		Scan(&result).Error; err != nil {
		return 0, err
	}
	return result.Total, nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}
