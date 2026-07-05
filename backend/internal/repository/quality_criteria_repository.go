package repository

import (
	"context"

	"ops-kb-rag/backend/internal/model"

	"gorm.io/gorm"
)

type QualityCriteriaRepository struct {
	db *gorm.DB
}

func NewQualityCriteriaRepository(db *gorm.DB) *QualityCriteriaRepository {
	return &QualityCriteriaRepository{db: db}
}

func (r *QualityCriteriaRepository) List(ctx context.Context) ([]model.QualityCriteria, error) {
	var items []model.QualityCriteria
	err := r.db.WithContext(ctx).Order("is_default DESC, updated_at DESC").Find(&items).Error
	return items, err
}

func (r *QualityCriteriaRepository) Create(ctx context.Context, item *model.QualityCriteria) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if item.IsDefault {
			if err := tx.Model(&model.QualityCriteria{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(item).Error
	})
}

func (r *QualityCriteriaRepository) GetByID(ctx context.Context, id uint64) (*model.QualityCriteria, error) {
	var item model.QualityCriteria
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *QualityCriteriaRepository) Update(ctx context.Context, item *model.QualityCriteria) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if item.IsDefault {
			if err := tx.Model(&model.QualityCriteria{}).Where("id <> ? AND is_default = ?", item.ID, true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(item).Error
	})
}

func (r *QualityCriteriaRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.QualityCriteria{}, id).Error
}

func (r *QualityCriteriaRepository) SetDefault(ctx context.Context, id uint64) (*model.QualityCriteria, error) {
	var item model.QualityCriteria
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&item, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.QualityCriteria{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		item.IsDefault = true
		return tx.Save(&item).Error
	})
	if err != nil {
		return nil, err
	}
	return &item, nil
}
