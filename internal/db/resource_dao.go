package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"linkit/internal/db/model"
)

type ResourceDao struct {
	store *DB
}

func (r *ResourceDao) Insert(ctx context.Context, resource model.Resource) (int64, error) {
	res, err := r.store.Client.ExecContext(ctx, `INSERT INTO resource(filename, hash, type, path, file_size, user_id) VALUES(?,?,?,?,?,?)`, resource.Filename, resource.Hash, resource.Type, resource.Path, resource.FileSize, resource.UserID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
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

func (r *ResourceDao) CreateShareCode(ctx context.Context, resourceID int64, userID int64) (*model.ShareCode, error) {
	for i := 0; i < 5; i++ {
		code, err := randomCode(6)
		if err != nil {
			return nil, err
		}
		row := r.store.Client.QueryRowContext(ctx, `INSERT INTO share_code(resource_id, user_id, code, created_at) VALUES(?,?,?, CURRENT_TIMESTAMP) RETURNING id, resource_id, user_id, code, view_count, created_at`, resourceID, userID, code)
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

func (r *ResourceDao) ListByUser(ctx context.Context, userID int64, page, size int) ([]model.UserResourceWithShare, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 15
	}
	if size > 100 {
		size = 100
	}
	offset := (page - 1) * size

	query := `
SELECT r.id, r.filename, r.type, r.created_at,
       (SELECT sc.code FROM share_code sc WHERE sc.resource_id = r.id ORDER BY sc.created_at DESC LIMIT 1) as share_code
FROM resource r
WHERE r.user_id = ?
ORDER BY r.created_at DESC
LIMIT ? OFFSET ?;
`
	rows, err := r.store.Client.QueryContext(ctx, query, userID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]model.UserResourceWithShare, 0)
	for rows.Next() {
		var item model.UserResourceWithShare
		if err := rows.Scan(&item.ID, &item.Filename, &item.Type, &item.CreatedAt, &item.ShareCode); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	var total int64
	if err := r.store.Client.QueryRowContext(ctx, `SELECT COUNT(1) FROM resource WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *ResourceDao) GetDashboardStats(ctx context.Context) (int64, int64, int64, error) {
	var totalFiles int64
	var totalSize int64
	if err := r.store.Client.QueryRowContext(ctx, `SELECT COUNT(1), COALESCE(SUM(file_size), 0) FROM resource`).Scan(&totalFiles, &totalSize); err != nil {
		return 0, 0, 0, err
	}
	var totalViews int64
	if err := r.store.Client.QueryRowContext(ctx, `SELECT COALESCE(SUM(view_count), 0) FROM share_code`).Scan(&totalViews); err != nil {
		return 0, 0, 0, err
	}
	return totalFiles, totalSize, totalViews, nil
}

func (r *ResourceDao) GetShareByCode(ctx context.Context, code string) (*model.ShareResource, error) {
	row := r.store.Client.QueryRowContext(ctx, `
SELECT sc.id as share_id, sc.code, sc.resource_id, r.filename, r.path, r.type, sc.view_count, r.created_at
FROM share_code sc
JOIN resource r ON r.id = sc.resource_id
WHERE sc.code = ?
LIMIT 1;
`, code)
	var res model.ShareResource
	if err := row.Scan(&res.ShareID, &res.Code, &res.ResourceID, &res.Filename, &res.Path, &res.Type, &res.ViewCount, &res.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *ResourceDao) IncrementShareViewCount(ctx context.Context, shareID int64) error {
	_, err := r.store.Client.ExecContext(ctx, `UPDATE share_code SET view_count = view_count + 1 WHERE id = ?`, shareID)
	return err
}

func (r *ResourceDao) FindByIDAndUser(ctx context.Context, resourceID, userID int64) (*model.Resource, error) {
	row := r.store.Client.QueryRowContext(ctx, `
SELECT id, filename, hash, type, path, file_size, user_id, created_at
FROM resource
WHERE id = ? AND user_id = ?
LIMIT 1;
`, resourceID, userID)
	var res model.Resource
	if err := row.Scan(&res.ID, &res.Filename, &res.Hash, &res.Type, &res.Path, &res.FileSize, &res.UserID, &res.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *ResourceDao) DeleteWithShare(ctx context.Context, resourceID, userID int64) (bool, error) {
	tx, err := r.store.Client.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	needRollback := true
	defer func() {
		if needRollback {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `DELETE FROM share_code WHERE resource_id = ?`, resourceID); err != nil {
		return false, err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM resource WHERE id = ? AND user_id = ?`, resourceID, userID)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	needRollback = false
	return true, nil
}
