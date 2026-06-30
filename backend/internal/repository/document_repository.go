package repository

import (
	"context"

	"ops-kb-rag/backend/internal/model"

	"gorm.io/gorm"
)

type DocumentFilter struct {
	Status        string
	SystemName    string
	ComponentName string
	DocType       string
	Page          int
	PageSize      int
}

type DocumentRepository struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

func (r *DocumentRepository) Create(ctx context.Context, doc *model.KBDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *DocumentRepository) Update(ctx context.Context, doc *model.KBDocument) error {
	return r.db.WithContext(ctx).Save(doc).Error
}

func (r *DocumentRepository) GetByID(ctx context.Context, id uint64) (*model.KBDocument, error) {
	var doc model.KBDocument
	if err := r.db.WithContext(ctx).First(&doc, id).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *DocumentRepository) List(ctx context.Context, filter DocumentFilter) ([]model.KBDocument, int64, error) {
	var docs []model.KBDocument
	var total int64
	query := r.db.WithContext(ctx).Model(&model.KBDocument{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.SystemName != "" {
		query = query.Where("system_name = ?", filter.SystemName)
	}
	if filter.ComponentName != "" {
		query = query.Where("component_name = ?", filter.ComponentName)
	}
	if filter.DocType != "" {
		query = query.Where("doc_type = ?", filter.DocType)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	err := query.Order("updated_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&docs).Error
	return docs, total, err
}

func (r *DocumentRepository) Stats(ctx context.Context) (map[string]int64, float64, error) {
	stats := map[string]int64{}
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.KBDocument{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	stats["total"] = total
	for _, status := range []string{model.DocumentStatusPublished, model.DocumentStatusReviewing} {
		var count int64
		if err := r.db.WithContext(ctx).Model(&model.KBDocument{}).Where("status = ?", status).Count(&count).Error; err != nil {
			return nil, 0, err
		}
		stats[status] = count
	}
	var avg *float64
	if err := r.db.WithContext(ctx).Model(&model.KBDocument{}).Select("AVG(quality_score)").Scan(&avg).Error; err != nil {
		return nil, 0, err
	}
	if avg == nil {
		return stats, 0, nil
	}
	return stats, *avg, nil
}
