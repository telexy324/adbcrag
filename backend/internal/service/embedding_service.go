package service

import (
	"context"

	"ops-kb-rag/backend/internal/client"
)

type EmbeddingService struct {
	client client.EmbeddingClient
}

func NewEmbeddingService(client client.EmbeddingClient) *EmbeddingService {
	return &EmbeddingService{client: client}
}

func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	return s.client.Embed(ctx, text)
}
