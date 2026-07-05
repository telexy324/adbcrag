package service

import (
	"context"
	"fmt"
	"strings"

	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/repository"
)

type QualityCriteriaService struct {
	repo *repository.QualityCriteriaRepository
}

func NewQualityCriteriaService(repo *repository.QualityCriteriaRepository) *QualityCriteriaService {
	return &QualityCriteriaService{repo: repo}
}

func (s *QualityCriteriaService) List(ctx context.Context) ([]model.QualityCriteria, error) {
	return s.repo.List(ctx)
}

func (s *QualityCriteriaService) Create(ctx context.Context, req dto.SaveQualityCriteriaRequest, createdBy string) (*model.QualityCriteria, error) {
	item, err := buildQualityCriteria(req)
	if err != nil {
		return nil, err
	}
	item.CreatedBy = createdBy
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *QualityCriteriaService) Update(ctx context.Context, id uint64, req dto.SaveQualityCriteriaRequest) (*model.QualityCriteria, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	next, err := buildQualityCriteria(req)
	if err != nil {
		return nil, err
	}
	item.Name = next.Name
	item.Content = next.Content
	item.IsDefault = next.IsDefault
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *QualityCriteriaService) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}

func (s *QualityCriteriaService) SetDefault(ctx context.Context, id uint64) (*model.QualityCriteria, error) {
	return s.repo.SetDefault(ctx, id)
}

func buildQualityCriteria(req dto.SaveQualityCriteriaRequest) (*model.QualityCriteria, error) {
	name := strings.TrimSpace(req.Name)
	content := strings.TrimSpace(req.Content)
	if name == "" {
		return nil, fmt.Errorf("criteria name is required")
	}
	if content == "" {
		return nil, fmt.Errorf("criteria content is required")
	}
	if len([]rune(content)) > 5000 {
		return nil, fmt.Errorf("criteria content exceeds 5000 characters")
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}
	return &model.QualityCriteria{Name: name, Content: content, IsDefault: isDefault}, nil
}
