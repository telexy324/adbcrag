package router

import (
	"ops-kb-rag/backend/internal/handler"
	"ops-kb-rag/backend/internal/middleware"
	"ops-kb-rag/backend/internal/service"

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
	Auth            *handler.AuthHandler
	Conversation    *handler.ConversationHandler
	AuthService     *service.AuthService
}

func New(handlers Handlers) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Logger(), middleware.Recover(), middleware.CORS())
	api := r.Group("/api")
	api.GET("/health", handlers.Health.Check)
	api.POST("/auth/login", handlers.Auth.Login)

	protected := api.Group("")
	protected.Use(middleware.Auth(handlers.AuthService))
	protected.GET("/auth/me", handlers.Auth.Me)
	protected.POST("/auth/change-password", handlers.Auth.ChangePassword)
	protected.GET("/dashboard/stats", handlers.Document.Stats)
	protected.POST("/documents/upload", handlers.Document.Upload)
	protected.GET("/documents", handlers.Document.List)
	protected.GET("/documents/:id", handlers.Document.Detail)
	protected.POST("/logs/preview", handlers.LogAnalysis.Preview)
	protected.POST("/log-analysis", handlers.LogAnalysis.Analyze)
	protected.GET("/log-sources", handlers.LogSource.List)
	protected.POST("/qa/ask", handlers.QA.Ask)
	protected.GET("/conversations", handlers.Conversation.List)
	protected.POST("/conversations", handlers.Conversation.Create)
	protected.GET("/conversations/:id/messages", handlers.Conversation.Messages)
	protected.DELETE("/conversations/:id", handlers.Conversation.Archive)
	protected.POST("/k8s/diagnosis/alert", handlers.K8s.DiagnoseAlert)
	protected.POST("/k8s/diagnosis/pod", handlers.K8s.DiagnosePod)
	protected.POST("/k8s/diagnosis/ingress", handlers.K8s.DiagnoseIngress)
	protected.POST("/k8s/diagnosis/service", handlers.K8s.DiagnoseService)
	protected.GET("/k8s/clusters", handlers.K8s.ListClusters)

	admin := protected.Group("")
	admin.Use(middleware.RequireAdmin())
	admin.GET("/users", handlers.Auth.ListUsers)
	admin.POST("/users", handlers.Auth.CreateUser)
	admin.PUT("/users/:id", handlers.Auth.UpdateUser)
	admin.POST("/users/:id/reset-password", handlers.Auth.ResetPassword)
	admin.POST("/documents/:id/review", handlers.Review.Review)
	admin.GET("/quality-criteria", handlers.QualityCriteria.List)
	admin.POST("/quality-criteria", handlers.QualityCriteria.Create)
	admin.PUT("/quality-criteria/:id", handlers.QualityCriteria.Update)
	admin.DELETE("/quality-criteria/:id", handlers.QualityCriteria.Delete)
	admin.POST("/quality-criteria/:id/default", handlers.QualityCriteria.SetDefault)
	admin.POST("/log-sources", handlers.LogSource.Create)
	admin.PUT("/log-sources/:id", handlers.LogSource.Update)
	admin.DELETE("/log-sources/:id", handlers.LogSource.Delete)
	admin.POST("/log-sources/:id/test", handlers.LogSource.Test)
	admin.GET("/llm-configs", handlers.LLMConfig.List)
	admin.GET("/llm-configs/default", handlers.LLMConfig.Active)
	admin.POST("/llm-configs", handlers.LLMConfig.Create)
	admin.PUT("/llm-configs/:id", handlers.LLMConfig.Update)
	admin.DELETE("/llm-configs/:id", handlers.LLMConfig.Delete)
	admin.POST("/llm-configs/:id/default", handlers.LLMConfig.SetDefault)
	admin.POST("/llm-configs/:id/test", handlers.LLMConfig.Test)
	admin.POST("/k8s/clusters", handlers.K8s.CreateCluster)
	admin.PUT("/k8s/clusters/:id", handlers.K8s.UpdateCluster)
	admin.DELETE("/k8s/clusters/:id", handlers.K8s.DeleteCluster)
	admin.POST("/k8s/clusters/:id/test", handlers.K8s.TestCluster)
	return r
}
