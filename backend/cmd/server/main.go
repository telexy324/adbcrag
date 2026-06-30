package main

import (
	"log"

	"ops-kb-rag/backend/internal/client"
	"ops-kb-rag/backend/internal/config"
	"ops-kb-rag/backend/internal/handler"
	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/repository"
	"ops-kb-rag/backend/internal/router"
	"ops-kb-rag/backend/internal/service"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		log.Fatalf("enable pgvector: %v", err)
	}
	if err := db.AutoMigrate(&model.KBDocument{}, &model.KBChunk{}, &model.QARecord{}, &model.KBReviewRecord{}); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	deepSeek := client.NewDeepSeekClient(cfg.DeepSeekBaseURL, cfg.DeepSeekAPIKey, cfg.DeepSeekModel)
	embeddingClient := client.NewEmbeddingClient(cfg.EmbeddingBaseURL, cfg.EmbeddingAPIKey, cfg.EmbeddingModel, cfg.EmbeddingDim)

	docRepo := repository.NewDocumentRepository(db)
	chunkRepo := repository.NewChunkRepository(db)
	qaRepo := repository.NewQARepository(db)
	reviewRepo := repository.NewReviewRepository(db)

	embeddingSvc := service.NewEmbeddingService(embeddingClient)
	documentSvc := service.NewDocumentService(
		cfg, docRepo, chunkRepo, service.NewParserService(), service.NewChunkService(), embeddingSvc, service.NewQualityService(deepSeek),
	)
	ragSvc := service.NewRAGService(cfg, embeddingSvc, chunkRepo, qaRepo, deepSeek)
	reviewSvc := service.NewReviewService(docRepo, reviewRepo)

	r := router.New(router.Handlers{
		Health: handler.NewHealthHandler(), Document: handler.NewDocumentHandler(documentSvc),
		QA: handler.NewQAHandler(ragSvc), Review: handler.NewReviewHandler(reviewSvc),
	})
	log.Printf("server listening on :%s", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
