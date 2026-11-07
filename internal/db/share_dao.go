package db

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"linkit/internal/db/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

var ErrShareCodeTaken = errors.New("分享短码已被占用，请更换")

func (s *ShareDao) CreateShareCode(ctx context.Context, resourceID int64, userID int64, password *string, expireTime *time.Time, relay bool) (*model.ShareCode, error) {
	return s.CreateShareWithCode(ctx, resourceID, userID, "", password, expireTime, relay)
}

func (s *ShareDao) CreateShareWithCode(ctx context.Context, resourceID, userID int64, customCode string, password *string, expireTime *time.Time, relay bool) (*model.ShareCode, error) {
	for i := 0; i < 5; i++ {
		code := customCode
		if code == "" {
			var err error
			code, err = randomCode(6)
			if err != nil {
				return nil, err
			}
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
				if customCode != "" {
					return nil, ErrShareCodeTaken
				}
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

func (s *ShareDao) ListByResource(ctx context.Context, resourceID, userID int64) ([]model.Share, error) {
	shares := make([]model.Share, 0)
	err := s.store.Client.WithContext(ctx).
		Where("resource_id = ? AND user_id = ?", resourceID, userID).
		Find(&shares).Error
	// Compare parsed times so existing records with different UTC offsets sort correctly.
	now := time.Now()
	sort.Slice(shares, func(i, j int) bool {
		iExpired := shares[i].ExpireTime != nil && !shares[i].ExpireTime.After(now)
		jExpired := shares[j].ExpireTime != nil && !shares[j].ExpireTime.After(now)
		if iExpired != jExpired {
			return !iExpired
		}
		if !shares[i].CreatedAt.Equal(shares[j].CreatedAt) {
			return shares[i].CreatedAt.After(shares[j].CreatedAt)
		}
		return shares[i].ID > shares[j].ID
	})
	return shares, err
}

func (s *ShareDao) UpdateByResource(ctx context.Context, shareID, resourceID, userID int64, values map[string]any) (bool, error) {
	result := s.store.Client.WithContext(ctx).Model(&model.Share{}).
		Where("id = ? AND resource_id = ? AND user_id = ?", shareID, resourceID, userID).
		Updates(values)
	return result.RowsAffected > 0, result.Error
}

func (s *ShareDao) DeleteByResource(ctx context.Context, shareID, resourceID, userID int64) (bool, error) {
	result := s.store.Client.WithContext(ctx).
		Where("id = ? AND resource_id = ? AND user_id = ?", shareID, resourceID, userID).
		Delete(&model.Share{})
	return result.RowsAffected > 0, result.Error
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
