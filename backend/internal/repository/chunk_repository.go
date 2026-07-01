package repository

import (
	"context"
	"fmt"
	"strings"

	"ops-kb-rag/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SearchFilter struct {
	SystemName    string
	ComponentName string
	DocType       string
	TopK          int
	Query         string
	Keywords      []string
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

func (r *ChunkRepository) KeywordSearch(ctx context.Context, filter SearchFilter) ([]SearchResult, error) {
	results := []SearchResult{}
	topK := filter.TopK
	if topK <= 0 {
		topK = 30
	}
	searchText := strings.TrimSpace(filter.Query + " " + strings.Join(filter.Keywords, " "))
	if searchText == "" {
		return results, nil
	}
	scoreSQL := `(
		GREATEST(
			similarity(c.content, ?),
			similarity(COALESCE(c.search_text, ''), ?),
			similarity(COALESCE(c.source_section, ''), ?),
			similarity(d.title, ?)
		)
		+ CASE WHEN c.content ILIKE ? THEN 0.35 ELSE 0 END
		+ CASE WHEN COALESCE(c.search_text, '') ILIKE ? THEN 0.45 ELSE 0 END
	) AS score`
	likeQuery := "%" + filter.Query + "%"
	query := r.db.WithContext(ctx).Table("kb_chunk c").
		Select("c.id, c.document_id, c.content, c.source_section, d.title AS document_title, d.system_name, d.component_name, d.doc_type, "+scoreSQL, searchText, searchText, searchText, searchText, likeQuery, likeQuery).
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
	conditions := []string{
		`(
			c.content ILIKE ?
			OR COALESCE(c.search_text, '') ILIKE ?
			OR COALESCE(c.source_section, '') ILIKE ?
			OR d.title ILIKE ?
			OR similarity(c.content, ?) > 0.1
			OR similarity(COALESCE(c.search_text, ''), ?) > 0.1
			OR similarity(COALESCE(c.source_section, ''), ?) > 0.1
			OR similarity(d.title, ?) > 0.1
		)`,
	}
	args := []interface{}{"%" + searchText + "%", "%" + searchText + "%", "%" + searchText + "%", "%" + searchText + "%", searchText, searchText, searchText, searchText}
	for _, keyword := range filter.Keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		like := "%" + keyword + "%"
		conditions = append(conditions, "(c.content ILIKE ? OR COALESCE(c.search_text, '') ILIKE ? OR COALESCE(c.source_section, '') ILIKE ? OR d.title ILIKE ?)")
		args = append(args, like, like, like, like)
	}
	query = query.Where(fmt.Sprintf("(%s)", strings.Join(conditions, " OR ")), args...)
	err := query.Order(clause.Expr{SQL: "score DESC"}).Limit(topK).Scan(&results).Error
	return results, err
}
