package main

import (
	"context"
	"os"

	"ops-kb-rag/backend/internal/client"
	"ops-kb-rag/backend/internal/config"
	"ops-kb-rag/backend/internal/handler"
	"ops-kb-rag/backend/internal/logger"
	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/repository"
	"ops-kb-rag/backend/internal/router"
	"ops-kb-rag/backend/internal/security"
	"ops-kb-rag/backend/internal/service"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel, cfg.LogFormat, cfg.AppEnv)

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		fatal("connect database", err)
	}
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		fatal("enable pg_trgm", err)
	}
	if err := db.AutoMigrate(&model.KBDocument{}, &model.KBChunk{}, &model.QARecord{}, &model.KBReviewRecord{}, &model.QualityCriteria{}, &model.LogSource{}, &model.LogAnalysisTask{}, &model.LLMConfig{}, &model.K8sCluster{}, &model.K8sDiagnosisTask{}); err != nil {
		fatal("auto migrate", err)
	}
	if err := ensureSearchIndexes(db); err != nil {
		fatal("create search indexes", err)
	}

	docRepo := repository.NewDocumentRepository(db)
	chunkRepo := repository.NewChunkRepository(db)
	qaRepo := repository.NewQARepository(db)
	reviewRepo := repository.NewReviewRepository(db)
	criteriaRepo := repository.NewQualityCriteriaRepository(db)
	logSourceRepo := repository.NewLogSourceRepository(db)
	logAnalysisRepo := repository.NewLogAnalysisRepository(db)
	llmConfigRepo := repository.NewLLMConfigRepository(db)
	k8sClusterRepo := repository.NewK8sClusterRepository(db)
	k8sDiagnosisRepo := repository.NewK8sDiagnosisRepository(db)
	credentialCrypto := security.NewCredentialCrypto(cfg.CredentialKey)
	esClient := client.NewElasticsearchClient()
	sshClient := client.NewSSHLogClient()
	k8sClient := client.NewK8sClient()
	llmConfigSvc := service.NewLLMConfigService(cfg, llmConfigRepo, credentialCrypto)
	llmClient := service.NewDynamicLLMClient(llmConfigSvc)

	documentSvc := service.NewDocumentService(
		cfg, docRepo, chunkRepo, service.NewParserService(), service.NewChunkService(), service.NewQualityService(llmClient), service.NewRetrievalMetadataService(llmClient),
	)
	ragSvc := service.NewRAGService(cfg, chunkRepo, qaRepo, llmClient)
	reviewSvc := service.NewReviewService(docRepo, reviewRepo)
	criteriaSvc := service.NewQualityCriteriaService(criteriaRepo)
	logSourceSvc := service.NewLogSourceService(cfg, logSourceRepo, credentialCrypto, esClient, sshClient)
	logAnalysisSvc := service.NewLogAnalysisService(cfg, logSourceSvc, logAnalysisRepo, chunkRepo, esClient, sshClient, llmClient)
	k8sSvc := service.NewK8sService(cfg, k8sClusterRepo, k8sDiagnosisRepo, chunkRepo, credentialCrypto, k8sClient, llmClient)

	r := router.New(router.Handlers{
		Health: handler.NewHealthHandler(), Document: handler.NewDocumentHandler(documentSvc),
		QA: handler.NewQAHandler(ragSvc), Review: handler.NewReviewHandler(reviewSvc),
		QualityCriteria: handler.NewQualityCriteriaHandler(criteriaSvc),
		LogSource:       handler.NewLogSourceHandler(logSourceSvc),
		LogAnalysis:     handler.NewLogAnalysisHandler(logAnalysisSvc),
		LLMConfig:       handler.NewLLMConfigHandler(llmConfigSvc),
		K8s:             handler.NewK8sHandler(k8sSvc),
	})
	logger.Info(context.Background(), "server listening", "port", cfg.AppPort, "env", cfg.AppEnv, "log_level", cfg.LogLevel, "log_format", cfg.LogFormat)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		fatal("run server", err)
	}
}

func fatal(msg string, err error) {
	logger.Error(context.Background(), msg, "error", err)
	os.Exit(1)
}

func ensureSearchIndexes(db *gorm.DB) error {
	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_kb_document_status ON kb_document(status)",
		"CREATE INDEX IF NOT EXISTS idx_kb_document_filters ON kb_document(system_name, component_name, doc_type)",
		"CREATE INDEX IF NOT EXISTS idx_kb_chunk_document_id ON kb_chunk(document_id)",
		"CREATE INDEX IF NOT EXISTS idx_kb_chunk_content_trgm ON kb_chunk USING gin (content gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_kb_chunk_search_text_trgm ON kb_chunk USING gin (search_text gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_kb_chunk_source_section_trgm ON kb_chunk USING gin (source_section gin_trgm_ops)",
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
