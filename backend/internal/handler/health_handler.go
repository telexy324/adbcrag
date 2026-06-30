package handler

import (
	"ops-kb-rag/backend/internal/dto"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Check(c *gin.Context) {
	dto.Success(c, gin.H{"status": "ok"})
}
