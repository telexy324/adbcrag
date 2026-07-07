package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/repository"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ConversationService struct {
	repo *repository.ConversationRepository
}

func NewConversationService(repo *repository.ConversationRepository) *ConversationService {
	return &ConversationService{repo: repo}
}

func (s *ConversationService) List(ctx context.Context, userID uint64, role string, all bool) ([]model.Conversation, error) {
	return s.repo.List(ctx, userID, all && role == model.UserRoleAdmin)
}

func (s *ConversationService) Create(ctx context.Context, userID uint64, req dto.CreateConversationRequest) (*model.Conversation, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "新的问答会话"
	}
	conversationType := req.ConversationType
	if conversationType == "" {
		conversationType = model.ConversationTypeQA
	}
	now := time.Now()
	item := &model.Conversation{UserID: userID, Title: title, ConversationType: conversationType, Status: model.ConversationStatusActive, LastMessageAt: now}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ConversationService) GetMessages(ctx context.Context, userID uint64, role string, conversationID uint64) (*dto.ConversationWithMessages, error) {
	conversation, err := s.requireAccess(ctx, userID, role, conversationID)
	if err != nil {
		return nil, err
	}
	messages, err := s.repo.Messages(ctx, conversationID, 0)
	if err != nil {
		return nil, err
	}
	summary, _ := s.repo.GetSummary(ctx, conversationID)
	text := ""
	if summary != nil {
		text = summary.Summary
	}
	return &dto.ConversationWithMessages{Conversation: *conversation, Messages: messages, Summary: text}, nil
}

func (s *ConversationService) Archive(ctx context.Context, userID uint64, role string, conversationID uint64) error {
	conversation, err := s.requireAccess(ctx, userID, role, conversationID)
	if err != nil {
		return err
	}
	conversation.Status = model.ConversationStatusArchived
	return s.repo.Update(ctx, conversation)
}

func (s *ConversationService) Ensure(ctx context.Context, userID uint64, role string, conversationID uint64, title string) (*model.Conversation, error) {
	if conversationID > 0 {
		return s.requireAccess(ctx, userID, role, conversationID)
	}
	return s.Create(ctx, userID, dto.CreateConversationRequest{Title: title, ConversationType: model.ConversationTypeQA})
}

func (s *ConversationService) AddMessage(ctx context.Context, conversation *model.Conversation, userID uint64, role, content string, metadata any) (*model.ConversationMessage, error) {
	data, _ := json.Marshal(metadata)
	message := &model.ConversationMessage{
		ConversationID: conversation.ID, UserID: userID, Role: role, Content: content, MessageType: "text", Metadata: datatypes.JSON(data),
	}
	if err := s.repo.CreateMessage(ctx, message); err != nil {
		return nil, err
	}
	conversation.LastMessageAt = time.Now()
	if conversation.Title == "" || conversation.Title == "新的问答会话" {
		conversation.Title = truncate(content, 36)
	}
	_ = s.repo.Update(ctx, conversation)
	return message, nil
}

func (s *ConversationService) RecentMessages(ctx context.Context, conversationID uint64, limit int) ([]model.ConversationMessage, error) {
	return s.repo.Messages(ctx, conversationID, limit)
}

func (s *ConversationService) Summary(ctx context.Context, conversationID uint64) string {
	summary, err := s.repo.GetSummary(ctx, conversationID)
	if err != nil {
		return ""
	}
	return summary.Summary
}

func (s *ConversationService) UpdateSummary(ctx context.Context, conversationID uint64) error {
	messages, err := s.repo.Messages(ctx, conversationID, 12)
	if err != nil {
		return err
	}
	parts := []string{}
	for _, message := range messages {
		if message.Role == "user" || message.Role == "assistant" {
			parts = append(parts, message.Role+": "+truncate(message.Content, 160))
		}
	}
	summaryText := strings.Join(parts, "\n")
	existing, err := s.repo.GetSummary(ctx, conversationID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if existing == nil {
		existing = &model.ConversationSummary{ConversationID: conversationID}
	}
	existing.Summary = summaryText
	existing.MessageCount = len(messages)
	existing.UpdatedAt = time.Now()
	return s.repo.UpsertSummary(ctx, existing)
}

func (s *ConversationService) SaveSnapshot(ctx context.Context, userID, conversationID, taskID uint64, snapshotType string, content any) error {
	data, _ := json.Marshal(content)
	return s.repo.CreateSnapshot(ctx, &model.ContextSnapshot{UserID: userID, ConversationID: conversationID, TaskID: taskID, SnapshotType: snapshotType, Content: datatypes.JSON(data)})
}

func (s *ConversationService) requireAccess(ctx context.Context, userID uint64, role string, conversationID uint64) (*model.Conversation, error) {
	conversation, err := s.repo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if role != model.UserRoleAdmin && conversation.UserID != userID {
		return nil, fmt.Errorf("conversation access denied")
	}
	if conversation.Status == model.ConversationStatusArchived {
		return nil, fmt.Errorf("conversation is archived")
	}
	return conversation, nil
}
