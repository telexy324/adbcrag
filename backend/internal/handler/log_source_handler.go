package handler

import (
	"strconv"

	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type LogSourceHandler struct {
	service *service.LogSourceService
}

func NewLogSourceHandler(service *service.LogSourceService) *LogSourceHandler {
	return &LogSourceHandler{service: service}
}

func (h *LogSourceHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		dto.Error(c, 500, err.Error())
		return
	}
	dto.Success(c, items)
}

func (h *LogSourceHandler) Create(c *gin.Context) {
	var req dto.SaveLogSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	item, err := h.service.Create(c.Request.Context(), req, c.GetHeader("X-User"))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, item)
}

func (h *LogSourceHandler) Update(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.SaveLogSourceRequest
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

func (h *LogSourceHandler) Delete(c *gin.Context) {
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

func (h *LogSourceHandler) Test(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Test(c.Request.Context(), id); err != nil {
		dto.Success(c, dto.TestLogSourceResponse{OK: false, Message: err.Error()})
		return
	}
	dto.Success(c, dto.TestLogSourceResponse{OK: true, Message: "连接成功"})
}

func parseUintParam(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		dto.Error(c, 400, "invalid "+name)
		return 0, false
	}
	return id, true
}
