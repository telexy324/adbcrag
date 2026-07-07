package repository

import (
	"context"

	"ops-kb-rag/backend/internal/model"

	"gorm.io/gorm"
)

type ConversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) Create(ctx context.Context, conversation *model.Conversation) error {
	return r.db.WithContext(ctx).Create(conversation).Error
}

func (r *ConversationRepository) GetByID(ctx context.Context, id uint64) (*model.Conversation, error) {
	var conversation model.Conversation
	if err := r.db.WithContext(ctx).First(&conversation, id).Error; err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (r *ConversationRepository) List(ctx context.Context, userID uint64, all bool) ([]model.Conversation, error) {
	var items []model.Conversation
	query := r.db.WithContext(ctx).Where("status <> ?", model.ConversationStatusArchived)
	if !all {
		query = query.Where("user_id = ?", userID)
	}
	err := query.Order("last_message_at DESC, id DESC").Find(&items).Error
	return items, err
}

func (r *ConversationRepository) Update(ctx context.Context, conversation *model.Conversation) error {
	return r.db.WithContext(ctx).Save(conversation).Error
}

func (r *ConversationRepository) CreateMessage(ctx context.Context, message *model.ConversationMessage) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *ConversationRepository) Messages(ctx context.Context, conversationID uint64, limit int) ([]model.ConversationMessage, error) {
	var items []model.ConversationMessage
	query := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("id ASC")
	if limit > 0 {
		var recent []model.ConversationMessage
		err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("id DESC").Limit(limit).Find(&recent).Error
		if err != nil {
			return nil, err
		}
		for i := len(recent) - 1; i >= 0; i-- {
			items = append(items, recent[i])
		}
		return items, nil
	}
	err := query.Find(&items).Error
	return items, err
}

func (r *ConversationRepository) GetSummary(ctx context.Context, conversationID uint64) (*model.ConversationSummary, error) {
	var summary model.ConversationSummary
	if err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).First(&summary).Error; err != nil {
		return nil, err
	}
	return &summary, nil
}

func (r *ConversationRepository) UpsertSummary(ctx context.Context, summary *model.ConversationSummary) error {
	return r.db.WithContext(ctx).Save(summary).Error
}

func (r *ConversationRepository) CreateSnapshot(ctx context.Context, snapshot *model.ContextSnapshot) error {
	return r.db.WithContext(ctx).Create(snapshot).Error
}
