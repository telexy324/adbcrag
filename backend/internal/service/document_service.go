package service

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"

	"ops-kb-rag/backend/internal/config"
	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/repository"
	"ops-kb-rag/backend/internal/util"

	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
)

type DocumentService struct {
	cfg       *config.Config
	docs      *repository.DocumentRepository
	chunks    *repository.ChunkRepository
	parser    *ParserService
	chunker   *ChunkService
	embedding *EmbeddingService
	quality   *QualityService
}

func NewDocumentService(cfg *config.Config, docs *repository.DocumentRepository, chunks *repository.ChunkRepository, parser *ParserService, chunker *ChunkService, embedding *EmbeddingService, quality *QualityService) *DocumentService {
	return &DocumentService{cfg: cfg, docs: docs, chunks: chunks, parser: parser, chunker: chunker, embedding: embedding, quality: quality}
}

type UploadInput struct {
	File          multipart.File
	FileHeader    *multipart.FileHeader
	Title         string
	SystemName    string
	ComponentName string
	DocType       string
	Tags          string
	CreatedBy     string
}

func (s *DocumentService) Upload(ctx context.Context, input UploadInput) (*dto.UploadDocumentResponse, error) {
	ext, err := util.ValidateUpload(input.FileHeader, s.cfg.MaxUploadMB)
	if err != nil {
		return nil, err
	}
	path, err := util.SaveUploadedFile(input.File, input.FileHeader, s.cfg.LocalFileDir)
	if err != nil {
		return nil, err
	}
	content, err := s.parser.Parse(path)
	if err != nil {
		return nil, err
	}
	quality, qualityJSON, err := s.quality.Check(ctx, content)
	if err != nil {
		return nil, err
	}
	title := input.Title
	if title == "" {
		title = input.FileHeader.Filename
	}
	doc := &model.KBDocument{
		Title:         title,
		FileName:      input.FileHeader.Filename,
		FilePath:      path,
		FileType:      ext,
		SystemName:    input.SystemName,
		ComponentName: input.ComponentName,
		DocType:       input.DocType,
		Tags:          input.Tags,
		Summary:       quality.Summary,
		Status:        StatusByQuality(quality.Score),
		QualityScore:  quality.Score,
		QualityResult: datatypes.JSON(qualityJSON),
		CreatedBy:     input.CreatedBy,
	}
	if err := s.docs.Create(ctx, doc); err != nil {
		return nil, err
	}
	textChunks := s.chunker.Split(doc.Title, content)
	dbChunks := make([]model.KBChunk, 0, len(textChunks))
	for _, textChunk := range textChunks {
		vector, err := s.embedding.Embed(ctx, textChunk.Content)
		if err != nil {
			return nil, fmt.Errorf("embed chunk %d: %w", textChunk.Index, err)
		}
		dbChunks = append(dbChunks, model.KBChunk{
			DocumentID:    doc.ID,
			ChunkIndex:    textChunk.Index,
			Content:       textChunk.Content,
			SourceTitle:   textChunk.SourceTitle,
			SourceSection: textChunk.SourceSection,
			TokenCount:    textChunk.TokenCount,
			Embedding:     pgvector.NewVector(vector),
		})
	}
	if err := s.chunks.CreateBatch(ctx, dbChunks); err != nil {
		return nil, err
	}
	var qr interface{}
	_ = json.Unmarshal(qualityJSON, &qr)
	return &dto.UploadDocumentResponse{ID: doc.ID, Title: doc.Title, Status: doc.Status, QualityScore: doc.QualityScore, QualityResult: qr}, nil
}

func (s *DocumentService) List(ctx context.Context, filter repository.DocumentFilter) (*dto.DocumentListResponse, error) {
	docs, total, err := s.docs.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	items := make([]dto.DocumentListItem, 0, len(docs))
	for _, doc := range docs {
		items = append(items, dto.DocumentListItem{
			ID: doc.ID, Title: doc.Title, SystemName: doc.SystemName, ComponentName: doc.ComponentName,
			DocType: doc.DocType, Status: doc.Status, QualityScore: doc.QualityScore, UpdatedAt: doc.UpdatedAt,
		})
	}
	return &dto.DocumentListResponse{Items: items, Total: total}, nil
}

func (s *DocumentService) Detail(ctx context.Context, id uint64) (*dto.DocumentDetailResponse, error) {
	doc, err := s.docs.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	var qr interface{}
	_ = json.Unmarshal(doc.QualityResult, &qr)
	return &dto.DocumentDetailResponse{
		ID: doc.ID, Title: doc.Title, FileName: doc.FileName, SystemName: doc.SystemName, ComponentName: doc.ComponentName,
		DocType: doc.DocType, Tags: doc.Tags, Summary: doc.Summary, Status: doc.Status, QualityScore: doc.QualityScore,
		QualityResult: qr, UpdatedAt: doc.UpdatedAt,
	}, nil
}

func (s *DocumentService) Stats(ctx context.Context) (map[string]interface{}, error) {
	stats, avg, err := s.docs.Stats(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"documentTotal": stats["total"], "publishedTotal": stats[model.DocumentStatusPublished], "reviewingTotal": stats[model.DocumentStatusReviewing], "averageQualityScore": avg}, nil
}
