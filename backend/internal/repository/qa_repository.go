package repository

import (
	"context"

	"ops-kb-rag/backend/internal/model"

	"gorm.io/gorm"
)

type QARepository struct {
	db *gorm.DB
}

func NewQARepository(db *gorm.DB) *QARepository {
	return &QARepository{db: db}
}

func (r *QARepository) Create(ctx context.Context, record *model.QARecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *QARepository) Recent(ctx context.Context, limit int) ([]model.QARecord, error) {
	var records []model.QARecord
	if limit <= 0 {
		limit = 5
	}
	err := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&records).Error
	return records, err
}
