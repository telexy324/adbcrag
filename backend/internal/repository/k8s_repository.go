package repository

import (
	"context"

	"ops-kb-rag/backend/internal/model"

	"gorm.io/gorm"
)

type K8sClusterRepository struct {
	db *gorm.DB
}

func NewK8sClusterRepository(db *gorm.DB) *K8sClusterRepository {
	return &K8sClusterRepository{db: db}
}

func (r *K8sClusterRepository) List(ctx context.Context) ([]model.K8sCluster, error) {
	var items []model.K8sCluster
	err := r.db.WithContext(ctx).Order("id desc").Find(&items).Error
	return items, err
}

func (r *K8sClusterRepository) GetByID(ctx context.Context, id uint64) (*model.K8sCluster, error) {
	var item model.K8sCluster
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *K8sClusterRepository) Create(ctx context.Context, item *model.K8sCluster) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *K8sClusterRepository) Update(ctx context.Context, item *model.K8sCluster) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *K8sClusterRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&model.K8sCluster{}, id).Error
}

type K8sDiagnosisRepository struct {
	db *gorm.DB
}

func NewK8sDiagnosisRepository(db *gorm.DB) *K8sDiagnosisRepository {
	return &K8sDiagnosisRepository{db: db}
}

func (r *K8sDiagnosisRepository) Create(ctx context.Context, task *model.K8sDiagnosisTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *K8sDiagnosisRepository) Update(ctx context.Context, task *model.K8sDiagnosisTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}
