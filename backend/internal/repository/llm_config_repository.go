package repository

import (
	"context"

	"ops-kb-rag/backend/internal/model"

	"gorm.io/gorm"
)

type LLMConfigRepository struct {
	db *gorm.DB
}

func NewLLMConfigRepository(db *gorm.DB) *LLMConfigRepository {
	return &LLMConfigRepository{db: db}
}

func (r *LLMConfigRepository) List(ctx context.Context) ([]model.LLMConfig, error) {
	var items []model.LLMConfig
	err := r.db.WithContext(ctx).Order("is_default DESC, updated_at DESC").Find(&items).Error
	return items, err
}

func (r *LLMConfigRepository) GetByID(ctx context.Context, id uint64) (*model.LLMConfig, error) {
	var item model.LLMConfig
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *LLMConfigRepository) GetDefault(ctx context.Context) (*model.LLMConfig, error) {
	var item model.LLMConfig
	if err := r.db.WithContext(ctx).Where("enabled = ? AND is_default = ?", true, true).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *LLMConfigRepository) Create(ctx context.Context, item *model.LLMConfig) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if item.IsDefault {
			if err := tx.Model(&model.LLMConfig{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(item).Error
	})
}

func (r *LLMConfigRepository) Update(ctx context.Context, item *model.LLMConfig) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if item.IsDefault {
			if err := tx.Model(&model.LLMConfig{}).Where("id <> ? AND is_default = ?", item.ID, true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(item).Error
	})
}

func (r *LLMConfigRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.LLMConfig{}, id).Error
}

func (r *LLMConfigRepository) SetDefault(ctx context.Context, id uint64) (*model.LLMConfig, error) {
	var item model.LLMConfig
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&item, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.LLMConfig{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		item.IsDefault = true
		item.Enabled = true
		return tx.Save(&item).Error
	})
	if err != nil {
		return nil, err
	}
	return &item, nil
}
