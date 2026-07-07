package handler

import (
	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/middleware"
	"ops-kb-rag/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type LogAnalysisHandler struct {
	service *service.LogAnalysisService
}

func NewLogAnalysisHandler(service *service.LogAnalysisService) *LogAnalysisHandler {
	return &LogAnalysisHandler{service: service}
}

func (h *LogAnalysisHandler) Preview(c *gin.Context) {
	var req dto.LogPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	result, err := h.service.Preview(c.Request.Context(), req)
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, result)
}

func (h *LogAnalysisHandler) Analyze(c *gin.Context) {
	var req dto.LogAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	result, err := h.service.Analyze(c.Request.Context(), req, middleware.CurrentUsername(c))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, result)
}
