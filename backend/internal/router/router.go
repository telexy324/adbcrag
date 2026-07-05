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
	api.POST("/qa/ask", handlers.QA.Ask)
	return r
}
