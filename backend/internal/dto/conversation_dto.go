package dto

import "ops-kb-rag/backend/internal/model"

type CreateConversationRequest struct {
	Title            string `json:"title"`
	ConversationType string `json:"conversationType"`
}

type ConversationWithMessages struct {
	Conversation model.Conversation          `json:"conversation"`
	Messages     []model.ConversationMessage `json:"messages"`
	Summary      string                      `json:"summary"`
}
