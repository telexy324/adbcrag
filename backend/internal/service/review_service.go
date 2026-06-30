package service

import (
	"context"
	"fmt"
	"time"

	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/repository"
)

type ReviewService struct {
	docs    *repository.DocumentRepository
	reviews *repository.ReviewRepository
}

func NewReviewService(docs *repository.DocumentRepository, reviews *repository.ReviewRepository) *ReviewService {
	return &ReviewService{docs: docs, reviews: reviews}
}

func (s *ReviewService) Review(ctx context.Context, id uint64, action, reviewer, comment string) (*model.KBDocument, error) {
	doc, err := s.docs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	from := doc.Status
	to, err := reviewTargetStatus(action)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	doc.Status = to
	doc.ReviewedBy = reviewer
	doc.ReviewedAt = &now
	if err := s.docs.Update(ctx, doc); err != nil {
		return nil, err
	}
	_ = s.reviews.Create(ctx, &model.KBReviewRecord{
		DocumentID: id, FromStatus: from, ToStatus: to, Reviewer: reviewer, Comment: comment,
	})
	return doc, nil
}

func reviewTargetStatus(action string) (string, error) {
	switch action {
	case "approve":
		return model.DocumentStatusPublished, nil
	case "reject":
		return model.DocumentStatusRejected, nil
	case "archive":
		return model.DocumentStatusArchived, nil
	case "deprecate":
		return model.DocumentStatusDeprecated, nil
	default:
		return "", fmt.Errorf("unsupported review action: %s", action)
	}
}
