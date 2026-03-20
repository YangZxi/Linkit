package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"gorm.io/gorm"
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
	if err := r.store.Client.WithContext(ctx).Create(&resource).Error; err != nil {
		return 0, err
	}
	return resource.ID, nil
}

func (r *ResourceDao) ListByUser(ctx context.Context, userID int64, page, size int, tagFilters []string) ([]model.UserResourceWithShare, int64, error) {
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

	baseQuery := r.store.Client.WithContext(ctx).Model(&model.Resource{}).Where("user_id = ?", userID)
	if len(tagFilters) > 0 {
		resourceIDs, err := r.findResourceIDsByTags(ctx, userID, tagFilters)
		if err != nil {
			return nil, 0, err
		}
		if len(resourceIDs) == 0 {
			return []model.UserResourceWithShare{}, 0, nil
		}
		baseQuery = baseQuery.Where("id IN ?", resourceIDs)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var resources []model.Resource
	if err := baseQuery.Order("created_at DESC").Order("id DESC").Offset(offset).Limit(size).Find(&resources).Error; err != nil {
		return nil, 0, err
	}
	if len(resources) == 0 {
		return []model.UserResourceWithShare{}, total, nil
	}

	resourceIDs := collectResourceIDs(resources)
	tagMap, err := r.listTagsMapByResourceIDs(ctx, resourceIDs)
	if err != nil {
		return nil, 0, err
	}
	shareMap, err := r.listLatestShareCodeMap(ctx, resourceIDs)
	if err != nil {
		return nil, 0, err
	}

	items := make([]model.UserResourceWithShare, 0, len(resources))
	for _, resource := range resources {
		item := model.UserResourceWithShare{
			ID:        resource.ID,
			Filename:  resource.Filename,
			Type:      resource.Type,
			Storage:   storageFromPath(resource.Path),
			CreatedAt: resource.CreatedAt,
			ShareCode: shareMap[resource.ID],
			Tags:      tagMap[resource.ID],
		}
		if item.Tags == nil {
			item.Tags = []string{}
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (r *ResourceDao) ListTagsByUser(ctx context.Context, userID int64) ([]string, error) {
	var resources []model.Resource
	if err := r.store.Client.WithContext(ctx).
		Model(&model.Resource{}).
		Select("id").
		Where("user_id = ?", userID).
		Find(&resources).Error; err != nil {
		return nil, err
	}
	resourceIDs := collectResourceIDs(resources)
	if len(resourceIDs) == 0 {
		return []string{}, nil
	}
	tagMap, err := r.listTagsMapByResourceIDs(ctx, resourceIDs)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	for _, values := range tagMap {
		for _, tag := range values {
			set[tag] = struct{}{}
		}
	}
	tags := make([]string, 0, len(set))
	for tag := range set {
		tags = append(tags, tag)
	}
	sort.Slice(tags, func(i, j int) bool {
		left := strings.ToLower(tags[i])
		right := strings.ToLower(tags[j])
		if left == right {
			return tags[i] < tags[j]
		}
		return left < right
	})
	return tags, nil
}

func (r *ResourceDao) ReplaceTags(ctx context.Context, resourceID int64, tags []string) error {
	return r.store.Client.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("resource_id = ?", resourceID).Delete(&model.ResourceTag{}).Error; err != nil {
			return err
		}
		if len(tags) == 0 {
			return nil
		}
		items := make([]model.ResourceTag, 0, len(tags))
		for _, tag := range tags {
			items = append(items, model.ResourceTag{
				ResourceID: resourceID,
				Tag:        tag,
			})
		}
		return tx.Create(&items).Error
	})
}

func (r *ResourceDao) GetDashboardStats(ctx context.Context) (int64, int64, int64, error) {
	var totalFiles int64
	if err := r.store.Client.WithContext(ctx).Model(&model.Resource{}).Count(&totalFiles).Error; err != nil {
		return 0, 0, 0, err
	}
	var fileSizeResult struct {
		Total int64 `gorm:"column:total"`
	}
	if err := r.store.Client.WithContext(ctx).
		Model(&model.Resource{}).
		Select("COALESCE(SUM(file_size), 0) AS total").
		Scan(&fileSizeResult).Error; err != nil {
		return 0, 0, 0, err
	}
	var viewResult struct {
		Total int64 `gorm:"column:total"`
	}
	if err := r.store.Client.WithContext(ctx).
		Model(&model.Share{}).
		Select("COALESCE(SUM(view_count), 0) AS total").
		Scan(&viewResult).Error; err != nil {
		return 0, 0, 0, err
	}
	return totalFiles, fileSizeResult.Total, viewResult.Total, nil
}

func (r *ResourceDao) FindByIDAndUser(ctx context.Context, resourceID, userID int64) (*model.Resource, error) {
	var res model.Resource
	err := r.store.Client.WithContext(ctx).Where("id = ? AND user_id = ?", resourceID, userID).First(&res).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *ResourceDao) FindLatestByUser(ctx context.Context, userID int64) (*model.Resource, error) {
	var res model.Resource
	err := r.store.Client.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Order("id DESC").
		First(&res).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *ResourceDao) DeleteWithShare(ctx context.Context, resourceID, userID int64) (bool, error) {
	var deleted bool
	err := r.store.Client.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("resource_id = ?", resourceID).Delete(&model.Share{}).Error; err != nil {
			return err
		}
		if err := tx.Where("resource_id = ?", resourceID).Delete(&model.ResourceTag{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ? AND user_id = ?", resourceID, userID).Delete(&model.Resource{})
		if result.Error != nil {
			return result.Error
		}
		deleted = result.RowsAffected > 0
		return nil
	})
	if err != nil {
		return false, err
	}
	return deleted, nil
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

func (r *ResourceDao) findResourceIDsByTags(ctx context.Context, userID int64, tagFilters []string) ([]int64, error) {
	var resourceIDs []int64
	err := r.store.Client.WithContext(ctx).
		Model(&model.ResourceTag{}).
		Distinct("resource_tag.resource_id").
		Joins("JOIN resource ON resource.id = resource_tag.resource_id").
		Where("resource.user_id = ?", userID).
		Where("resource_tag.tag IN ?", tagFilters).
		Pluck("resource_tag.resource_id", &resourceIDs).Error
	if err != nil {
		return nil, err
	}
	return resourceIDs, nil
}

func (r *ResourceDao) listTagsMapByResourceIDs(ctx context.Context, resourceIDs []int64) (map[int64][]string, error) {
	result := make(map[int64][]string, len(resourceIDs))
	if len(resourceIDs) == 0 {
		return result, nil
	}
	var tagRows []model.ResourceTag
	if err := r.store.Client.WithContext(ctx).
		Where("resource_id IN ?", resourceIDs).
		Order("tag ASC").
		Find(&tagRows).Error; err != nil {
		return nil, err
	}
	for _, row := range tagRows {
		result[row.ResourceID] = append(result[row.ResourceID], row.Tag)
	}
	return result, nil
}

func (r *ResourceDao) listLatestShareCodeMap(ctx context.Context, resourceIDs []int64) (map[int64]*string, error) {
	result := make(map[int64]*string, len(resourceIDs))
	if len(resourceIDs) == 0 {
		return result, nil
	}
	var shares []model.Share
	if err := r.store.Client.WithContext(ctx).
		Where("resource_id IN ?", resourceIDs).
		Where("password IS NULL OR password = ''").
		Order("created_at DESC").
		Order("id DESC").
		Find(&shares).Error; err != nil {
		return nil, err
	}
	for _, share := range shares {
		if _, exists := result[share.ResourceID]; exists {
			continue
		}
		code := share.Code
		result[share.ResourceID] = &code
	}
	return result, nil
}

func collectResourceIDs(resources []model.Resource) []int64 {
	ids := make([]int64, 0, len(resources))
	for _, resource := range resources {
		ids = append(ids, resource.ID)
	}
	return ids
}

func storageFromPath(path string) string {
	if strings.HasPrefix(path, "local@/") {
		return "local"
	}
	return "s3"
}
