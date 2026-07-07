package handler

import (
	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/middleware"
	"ops-kb-rag/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type QAHandler struct {
	service *service.RAGService
}

func NewQAHandler(service *service.RAGService) *QAHandler {
	return &QAHandler{service: service}
}

func (h *QAHandler) Ask(c *gin.Context) {
	var req dto.AskQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	result, err := h.service.Ask(c.Request.Context(), req, middleware.CurrentUserID(c), middleware.CurrentUsername(c), middleware.CurrentUserRole(c))
	if err != nil {
		dto.Error(c, 500, err.Error())
		return
	}
	dto.Success(c, result)
}
