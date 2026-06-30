package handler

import (
	"strconv"

	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	service *service.ReviewService
}

func NewReviewHandler(service *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{service: service}
}

func (h *ReviewHandler) Review(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		dto.Error(c, 400, "invalid document id")
		return
	}
	var req dto.ReviewDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	doc, err := h.service.Review(c.Request.Context(), id, req.Action, c.GetHeader("X-User"), req.Comment)
	if err != nil {
		dto.Error(c, 500, err.Error())
		return
	}
	dto.Success(c, doc)
}
