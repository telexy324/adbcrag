package router

import (
	"ops-kb-rag/backend/internal/handler"
	"ops-kb-rag/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Health          *handler.HealthHandler
	Document        *handler.DocumentHandler
	QA              *handler.QAHandler
	Review          *handler.ReviewHandler
	QualityCriteria *handler.QualityCriteriaHandler
	LogSource       *handler.LogSourceHandler
	LogAnalysis     *handler.LogAnalysisHandler
	LLMConfig       *handler.LLMConfigHandler
	K8s             *handler.K8sHandler
}

func New(handlers Handlers) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Logger(), middleware.Recover(), middleware.CORS())
	api := r.Group("/api")
	api.GET("/health", handlers.Health.Check)
	api.GET("/dashboard/stats", handlers.Document.Stats)
	api.POST("/documents/upload", handlers.Document.Upload)
	api.GET("/documents", handlers.Document.List)
	api.GET("/documents/:id", handlers.Document.Detail)
	api.POST("/documents/:id/review", handlers.Review.Review)
	api.GET("/quality-criteria", handlers.QualityCriteria.List)
	api.POST("/quality-criteria", handlers.QualityCriteria.Create)
	api.PUT("/quality-criteria/:id", handlers.QualityCriteria.Update)
	api.DELETE("/quality-criteria/:id", handlers.QualityCriteria.Delete)
	api.POST("/quality-criteria/:id/default", handlers.QualityCriteria.SetDefault)
	api.GET("/log-sources", handlers.LogSource.List)
	api.POST("/log-sources", handlers.LogSource.Create)
	api.PUT("/log-sources/:id", handlers.LogSource.Update)
	api.DELETE("/log-sources/:id", handlers.LogSource.Delete)
	api.POST("/log-sources/:id/test", handlers.LogSource.Test)
	api.POST("/logs/preview", handlers.LogAnalysis.Preview)
	api.POST("/log-analysis", handlers.LogAnalysis.Analyze)
	api.GET("/llm-configs", handlers.LLMConfig.List)
	api.POST("/llm-configs", handlers.LLMConfig.Create)
	api.PUT("/llm-configs/:id", handlers.LLMConfig.Update)
	api.DELETE("/llm-configs/:id", handlers.LLMConfig.Delete)
	api.POST("/llm-configs/:id/default", handlers.LLMConfig.SetDefault)
	api.POST("/llm-configs/:id/test", handlers.LLMConfig.Test)
	api.POST("/qa/ask", handlers.QA.Ask)
	api.GET("/k8s/clusters", handlers.K8s.ListClusters)
	api.POST("/k8s/clusters", handlers.K8s.CreateCluster)
	api.PUT("/k8s/clusters/:id", handlers.K8s.UpdateCluster)
	api.DELETE("/k8s/clusters/:id", handlers.K8s.DeleteCluster)
	api.POST("/k8s/clusters/:id/test", handlers.K8s.TestCluster)
	api.POST("/k8s/diagnosis/alert", handlers.K8s.DiagnoseAlert)
	api.POST("/k8s/diagnosis/pod", handlers.K8s.DiagnosePod)
	api.POST("/k8s/diagnosis/ingress", handlers.K8s.DiagnoseIngress)
	api.POST("/k8s/diagnosis/service", handlers.K8s.DiagnoseService)
	return r
}
