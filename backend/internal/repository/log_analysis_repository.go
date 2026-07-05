package repository

import (
	"context"

	"ops-kb-rag/backend/internal/model"

	"gorm.io/gorm"
)

type LogAnalysisRepository struct {
	db *gorm.DB
}

func NewLogAnalysisRepository(db *gorm.DB) *LogAnalysisRepository {
	return &LogAnalysisRepository{db: db}
}

func (r *LogAnalysisRepository) Create(ctx context.Context, task *model.LogAnalysisTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *LogAnalysisRepository) Update(ctx context.Context, task *model.LogAnalysisTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}
