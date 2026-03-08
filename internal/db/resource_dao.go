package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"linkit/internal/db/model"
)

const maxTagRuneLength = 6

func splitRawTags(value string) []string {
	if value == "" {
		return nil
	}
	return strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case ',', ';', '|':
			return true
		default:
			return r <= 32
		}
	})
}

// NormalizeTag 负责去除首尾空白、统一大小写并校验长度。
// 返回空字符串表示输入为空，可在上层忽略。
func NormalizeTag(raw string) (string, error) {
	tag := strings.TrimSpace(raw)
	if tag == "" {
		return "", nil
	}
	if utf8.RuneCountInString(tag) > maxTagRuneLength {
		return "", fmt.Errorf("标签长度不能超过 %d 个字符", maxTagRuneLength)
	}
	tag = strings.ToLower(tag)
	return tag, nil
}

// ParseTagsFromStrings 支持多值、多分隔符（逗号/空白/|/;）解析，并自动去重。
func ParseTagsFromStrings(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{})
	tags := make([]string, 0, len(values))
	for _, raw := range values {
		for _, piece := range splitRawTags(raw) {
			normalized, err := NormalizeTag(piece)
			if err != nil {
				return nil, err
			}
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			tags = append(tags, normalized)
		}
	}
	if len(tags) == 0 {
		return nil, nil
	}
	return tags, nil
}

type ResourceDao struct {
	store  *DB
	pickMu sync.RWMutex
	picks  map[int64]int64
}

func NewResourceDao(store *DB) *ResourceDao {
	return &ResourceDao{
		store: store,
		picks: make(map[int64]int64),
	}
}

func (r *ResourceDao) Insert(ctx context.Context, resource model.Resource) (int64, error) {
	res, err := r.store.Client.ExecContext(ctx, `INSERT INTO resource(filename, hash, type, path, file_size, user_id) VALUES(?,?,?,?,?,?)`, resource.Filename, resource.Hash, resource.Type, resource.Path, resource.FileSize, resource.UserID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ResourceDao) ListByUser(ctx context.Context, userID int64, page, size int, tagFilter string) ([]model.UserResourceWithShare, int64, error) {
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

	tagFilter = strings.TrimSpace(strings.ToLower(tagFilter))

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
SELECT r.id,
       r.filename,
       r.type,
       r.created_at,
       CASE
         WHEN r.path LIKE 'local@/%' THEN 'local'
         ELSE 's3'
       END as storage,
       (SELECT sc.code FROM share sc WHERE sc.resource_id = r.id AND (sc.password IS NULL OR sc.password = '') ORDER BY sc.created_at DESC LIMIT 1) as share_code,
       COALESCE(tag_agg.tags, '') as tags
FROM resource r
LEFT JOIN (
    SELECT resource_id, GROUP_CONCAT(tag, ',') AS tags
    FROM resource_tag
    GROUP BY resource_id
) tag_agg ON tag_agg.resource_id = r.id
WHERE r.user_id = ?`)
	args := []any{userID}
	if tagFilter != "" {
		queryBuilder.WriteString(`
  AND EXISTS (
    SELECT 1 FROM resource_tag rt WHERE rt.resource_id = r.id AND rt.tag = ?
  )`)
		args = append(args, tagFilter)
	}
	queryBuilder.WriteString(`
ORDER BY r.created_at DESC
LIMIT ? OFFSET ?;`)
	args = append(args, size, offset)

	rows, err := r.store.Client.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]model.UserResourceWithShare, 0)
	for rows.Next() {
		var item model.UserResourceWithShare
		var tags sql.NullString
		if err := rows.Scan(&item.ID, &item.Filename, &item.Type, &item.CreatedAt, &item.Storage, &item.ShareCode, &tags); err != nil {
			return nil, 0, err
		}
		if tags.Valid && tags.String != "" {
			item.Tags = splitRawTags(tags.String)
		} else {
			item.Tags = []string{}
		}
		items = append(items, item)
	}

	var total int64
	countBuilder := strings.Builder{}
	countBuilder.WriteString(`SELECT COUNT(1) FROM resource r WHERE r.user_id = ?`)
	countArgs := []any{userID}
	if tagFilter != "" {
		countBuilder.WriteString(`
  AND EXISTS (
    SELECT 1 FROM resource_tag rt WHERE rt.resource_id = r.id AND rt.tag = ?
  )`)
		countArgs = append(countArgs, tagFilter)
	}
	if err := r.store.Client.QueryRowContext(ctx, countBuilder.String(), countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *ResourceDao) ReplaceTags(ctx context.Context, resourceID int64, tags []string) error {
	tx, err := r.store.Client.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	needRollback := true
	defer func() {
		if needRollback {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `DELETE FROM resource_tag WHERE resource_id = ?`, resourceID); err != nil {
		return err
	}
	if len(tags) > 0 {
		stmt, err := tx.PrepareContext(ctx, `INSERT INTO resource_tag(resource_id, tag) VALUES(?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, tag := range tags {
			if _, err := stmt.ExecContext(ctx, resourceID, tag); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	needRollback = false
	return nil
}

func (r *ResourceDao) GetDashboardStats(ctx context.Context) (int64, int64, int64, error) {
	var totalFiles int64
	var totalSize int64
	if err := r.store.Client.QueryRowContext(ctx, `SELECT COUNT(1), COALESCE(SUM(file_size), 0) FROM resource`).Scan(&totalFiles, &totalSize); err != nil {
		return 0, 0, 0, err
	}
	var totalViews int64
	if err := r.store.Client.QueryRowContext(ctx, `SELECT COALESCE(SUM(view_count), 0) FROM share`).Scan(&totalViews); err != nil {
		return 0, 0, 0, err
	}
	return totalFiles, totalSize, totalViews, nil
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

func (r *ResourceDao) FindLatestByUser(ctx context.Context, userID int64) (*model.Resource, error) {
	row := r.store.Client.QueryRowContext(ctx, `
SELECT id, filename, hash, type, path, file_size, user_id, created_at
FROM resource
WHERE user_id = ?
ORDER BY created_at DESC, id DESC
LIMIT 1;
`, userID)
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
	if _, err := tx.ExecContext(ctx, `DELETE FROM share WHERE resource_id = ?`, resourceID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM resource_tag WHERE resource_id = ?`, resourceID); err != nil {
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

func (r *ResourceDao) GetUserPickResourceID(ctx context.Context, userID int64) (int64, bool, error) {
	r.pickMu.RLock()
	defer r.pickMu.RUnlock()
	resourceID, ok := r.picks[userID]
	return resourceID, ok, nil
}

func (r *ResourceDao) SetUserPickResourceID(userID, resourceID int64) error {
	r.pickMu.Lock()
	defer r.pickMu.Unlock()
	if r.picks == nil {
		r.picks = make(map[int64]int64)
	}
	r.picks[userID] = resourceID
	return nil
}

func (r *ResourceDao) ClearUserPickIfMatch(ctx context.Context, userID, resourceID int64) error {
	r.pickMu.Lock()
	defer r.pickMu.Unlock()
	if existing, ok := r.picks[userID]; ok && existing == resourceID {
		delete(r.picks, userID)
	}
	return nil
}
