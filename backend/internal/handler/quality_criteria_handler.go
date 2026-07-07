package handler

import (
	"strconv"

	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/middleware"
	"ops-kb-rag/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type QualityCriteriaHandler struct {
	service *service.QualityCriteriaService
}

func NewQualityCriteriaHandler(service *service.QualityCriteriaService) *QualityCriteriaHandler {
	return &QualityCriteriaHandler{service: service}
}

func (h *QualityCriteriaHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		dto.Error(c, 500, err.Error())
		return
	}
	dto.Success(c, items)
}

func (h *QualityCriteriaHandler) Create(c *gin.Context) {
	var req dto.SaveQualityCriteriaRequest
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

func (h *QualityCriteriaHandler) Update(c *gin.Context) {
	id, ok := parseCriteriaID(c)
	if !ok {
		return
	}
	var req dto.SaveQualityCriteriaRequest
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

func (h *QualityCriteriaHandler) Delete(c *gin.Context) {
	id, ok := parseCriteriaID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, gin.H{"id": id})
}

func (h *QualityCriteriaHandler) SetDefault(c *gin.Context) {
	id, ok := parseCriteriaID(c)
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

func parseCriteriaID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		dto.Error(c, 400, "invalid criteria id")
		return 0, false
	}
	return id, true
}
