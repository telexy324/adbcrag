package handler

import (
	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/middleware"
	"ops-kb-rag/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type LLMConfigHandler struct {
	service *service.LLMConfigService
}

func NewLLMConfigHandler(service *service.LLMConfigService) *LLMConfigHandler {
	return &LLMConfigHandler{service: service}
}

func (h *LLMConfigHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		dto.Error(c, 500, err.Error())
		return
	}
	dto.Success(c, items)
}

func (h *LLMConfigHandler) Create(c *gin.Context) {
	var req dto.SaveLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	item, err := h.service.Create(c.Request.Context(), req, middleware.CurrentUsername(c))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, item)
}

func (h *LLMConfigHandler) Update(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.SaveLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, item)
}

func (h *LLMConfigHandler) Delete(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, gin.H{"id": id})
}

func (h *LLMConfigHandler) SetDefault(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	item, err := h.service.SetDefault(c.Request.Context(), id)
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, item)
}

func (h *LLMConfigHandler) Test(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.TestLLMConfigRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.service.Test(c.Request.Context(), id, req.Prompt)
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, result)
}
