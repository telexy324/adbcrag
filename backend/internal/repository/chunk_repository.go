package repository

import (
	"context"

	"ops-kb-rag/backend/internal/model"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SearchFilter struct {
	SystemName    string
	ComponentName string
	DocType       string
	TopK          int
	MinScore      float64
}

type SearchResult struct {
	ID            uint64
	DocumentID    uint64
	Content       string
	SourceSection string
	DocumentTitle string
	SystemName    string
	ComponentName string
	DocType       string
	Score         float64
}

type ChunkRepository struct {
	db *gorm.DB
}

func NewChunkRepository(db *gorm.DB) *ChunkRepository {
	return &ChunkRepository{db: db}
}

func (r *ChunkRepository) CreateBatch(ctx context.Context, chunks []model.KBChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(chunks, 100).Error
}

func (r *ChunkRepository) Search(ctx context.Context, vector []float32, filter SearchFilter) ([]SearchResult, error) {
	results := []SearchResult{}
	topK := filter.TopK
	if topK <= 0 {
		topK = 5
	}
	query := r.db.WithContext(ctx).Table("kb_chunk c").
		Select("c.id, c.document_id, c.content, c.source_section, d.title AS document_title, d.system_name, d.component_name, d.doc_type, 1 - (c.embedding <=> ?) AS score", pgvector.NewVector(vector)).
		Joins("JOIN kb_document d ON c.document_id = d.id").
		Where("d.status = ?", model.DocumentStatusPublished)
	if filter.SystemName != "" {
		query = query.Where("d.system_name = ?", filter.SystemName)
	}
	if filter.ComponentName != "" {
		query = query.Where("d.component_name = ?", filter.ComponentName)
	}
	if filter.DocType != "" {
		query = query.Where("d.doc_type = ?", filter.DocType)
	}
	if filter.MinScore > 0 {
		query = query.Where("1 - (c.embedding <=> ?) >= ?", pgvector.NewVector(vector), filter.MinScore)
	}
	err := query.Order(clause.Expr{SQL: "c.embedding <=> ?", Vars: []interface{}{pgvector.NewVector(vector)}}).Limit(topK).Scan(&results).Error
	return results, err
}
