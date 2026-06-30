package repository

import (
	"context"

	"ops-kb-rag/backend/internal/model"

	"gorm.io/gorm"
)

type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Create(ctx context.Context, record *model.KBReviewRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}
