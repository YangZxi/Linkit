package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

func (s *ShareDao) CreateShareCode(ctx context.Context, resourceID int64, userID int64, password *string, expireTime *time.Time) (*model.ShareCode, error) {
	for i := 0; i < 5; i++ {
		code, err := randomCode(6)
		if err != nil {
			return nil, err
		}
		var passwordValue sql.NullString
		if password != nil && *password != "" {
			passwordValue = sql.NullString{String: *password, Valid: true}
		}
		var expireValue sql.NullTime
		if expireTime != nil {
			expireValue = sql.NullTime{Time: *expireTime, Valid: true}
		}
		row := s.store.Client.QueryRowContext(ctx, `INSERT INTO share(resource_id, user_id, code, password, expire_time, created_at) VALUES(?,?,?,?,?, CURRENT_TIMESTAMP) RETURNING id, resource_id, user_id, code, view_count, created_at`, resourceID, userID, code, passwordValue, expireValue)
		var sc model.ShareCode
		if err := row.Scan(&sc.ID, &sc.ResourceID, &sc.UserID, &sc.Code, &sc.ViewCount, &sc.CreatedAt); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				continue
			}
			return nil, err
		}
		return &sc, nil
	}
	return nil, fmt.Errorf("生成短链失败")
}

func (s *ShareDao) GetShareByCode(ctx context.Context, code string) (*model.ShareResource, error) {
	row := s.store.Client.QueryRowContext(ctx, `
SELECT sc.id as share_id, sc.code, sc.resource_id, sc.user_id, r.filename, r.path, r.type, sc.view_count, r.created_at, sc.password, sc.expire_time
FROM share sc
JOIN resource r ON r.id = sc.resource_id
WHERE sc.code = ?
LIMIT 1;
`, code)
	var res model.ShareResource
	var password sql.NullString
	var expireTime sql.NullTime
	if err := row.Scan(&res.ShareID, &res.Code, &res.ResourceID, &res.UserID, &res.Filename, &res.Path, &res.Type, &res.ViewCount, &res.CreatedAt, &password, &expireTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if password.Valid {
		value := password.String
		res.Password = &value
	}
	if expireTime.Valid {
		parsed := expireTime.Time
		res.ExpireTime = &parsed
	}
	return &res, nil
}

func (s *ShareDao) IncrementShareViewCount(ctx context.Context, shareID int64) error {
	_, err := s.store.Client.ExecContext(ctx, `UPDATE share SET view_count = view_count + 1 WHERE id = ?`, shareID)
	return err
}

func (s *ShareDao) GetTotalViewCount(ctx context.Context) (int64, error) {
	var totalViews int64
	if err := s.store.Client.QueryRowContext(ctx, `SELECT COALESCE(SUM(view_count), 0) FROM share`).Scan(&totalViews); err != nil {
		return 0, err
	}
	return totalViews, nil
}

