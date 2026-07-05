package handler

import (
	"strconv"
	"strings"

	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/repository"
	"ops-kb-rag/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	service *service.DocumentService
}

func NewDocumentHandler(service *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{service: service}
}

func (h *DocumentHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		dto.Error(c, 400, "file is required")
		return
	}
	defer file.Close()
	result, err := h.service.Upload(c.Request.Context(), service.UploadInput{
		File:            file,
		FileHeader:      header,
		Title:           c.PostForm("title"),
		SystemName:      c.PostForm("systemName"),
		ComponentName:   c.PostForm("componentName"),
		DocType:         c.PostForm("docType"),
		Tags:            c.PostForm("tags"),
		QualityCriteria: c.PostForm("qualityCriteria"),
		CreatedBy:       c.GetHeader("X-User"),
	})
	if err != nil {
		dto.Error(c, uploadErrorStatus(err), err.Error())
		return
	}
	dto.Success(c, result)
}

func uploadErrorStatus(err error) int {
	message := strings.ToLower(err.Error())
	for _, marker := range []string{
		"unsupported file type",
		"file size exceeds",
		"legacy .doc parsing is not supported",
		"office parser",
		"invalid .docx",
		"not a valid office",
		"corrupted",
		"incompletely uploaded",
		"checksum error",
		"parser for",
	} {
		if strings.Contains(message, marker) {
			return 400
		}
	}
	return 500
}

func (h *DocumentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	result, err := h.service.List(c.Request.Context(), repository.DocumentFilter{
		Page: page, PageSize: pageSize, Status: c.Query("status"), SystemName: c.Query("systemName"),
		ComponentName: c.Query("componentName"), DocType: c.Query("docType"),
	})
	if err != nil {
		dto.Error(c, 500, err.Error())
		return
	}
	dto.Success(c, result)
}

func (h *DocumentHandler) Detail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		dto.Error(c, 400, "invalid document id")
		return
	}
	result, err := h.service.Detail(c.Request.Context(), id)
	if err != nil {
		dto.Error(c, 404, err.Error())
		return
	}
	dto.Success(c, result)
}

func (h *DocumentHandler) Stats(c *gin.Context) {
	result, err := h.service.Stats(c.Request.Context())
	if err != nil {
		dto.Error(c, 500, err.Error())
		return
	}
	dto.Success(c, result)
}
