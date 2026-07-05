package repository

import (
	"context"

	"ops-kb-rag/backend/internal/model"

	"gorm.io/gorm"
)

type LogSourceRepository struct {
	db *gorm.DB
}

func NewLogSourceRepository(db *gorm.DB) *LogSourceRepository {
	return &LogSourceRepository{db: db}
}

func (r *LogSourceRepository) List(ctx context.Context) ([]model.LogSource, error) {
	var items []model.LogSource
	err := r.db.WithContext(ctx).Order("updated_at DESC").Find(&items).Error
	return items, err
}

func (r *LogSourceRepository) Create(ctx context.Context, source *model.LogSource) error {
	return r.db.WithContext(ctx).Create(source).Error
}

func (r *LogSourceRepository) GetByID(ctx context.Context, id uint64) (*model.LogSource, error) {
	var source model.LogSource
	if err := r.db.WithContext(ctx).First(&source, id).Error; err != nil {
		return nil, err
	}
	return &source, nil
}

func (r *LogSourceRepository) Update(ctx context.Context, source *model.LogSource) error {
	return r.db.WithContext(ctx).Save(source).Error
}

func (r *LogSourceRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.LogSource{}, id).Error
}
