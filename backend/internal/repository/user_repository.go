package repository

import (
	"context"

	"ops-kb-rag/backend/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CountAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AppUser{}).Where("role = ?", model.UserRoleAdmin).Count(&count).Error
	return count, err
}

func (r *UserRepository) Create(ctx context.Context, user *model.AppUser) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) GetByID(ctx context.Context, id uint64) (*model.AppUser, error) {
	var user model.AppUser
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.AppUser, error) {
	var user model.AppUser
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) List(ctx context.Context) ([]model.AppUser, error) {
	var users []model.AppUser
	err := r.db.WithContext(ctx).Order("id desc").Find(&users).Error
	return users, err
}

func (r *UserRepository) Update(ctx context.Context, user *model.AppUser) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepository) AuditLogin(ctx context.Context, audit *model.LoginAudit) error {
	return r.db.WithContext(ctx).Create(audit).Error
}
