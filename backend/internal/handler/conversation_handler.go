package handler

import (
	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/middleware"
	"ops-kb-rag/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type ConversationHandler struct {
	service *service.ConversationService
}

func NewConversationHandler(service *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{service: service}
}

func (h *ConversationHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), middleware.CurrentUserID(c), middleware.CurrentUserRole(c), c.Query("all") == "true")
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, items)
}

func (h *ConversationHandler) Create(c *gin.Context) {
	var req dto.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	item, err := h.service.Create(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, item)
}

func (h *ConversationHandler) Messages(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	result, err := h.service.GetMessages(c.Request.Context(), middleware.CurrentUserID(c), middleware.CurrentUserRole(c), id)
	if err != nil {
		dto.Error(c, 403, err.Error())
		return
	}
	dto.Success(c, result)
}

func (h *ConversationHandler) Archive(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	if err := h.service.Archive(c.Request.Context(), middleware.CurrentUserID(c), middleware.CurrentUserRole(c), id); err != nil {
		dto.Error(c, 403, err.Error())
		return
	}
	dto.Success(c, gin.H{"archived": true})
}
