package handler

import (
	"strconv"

	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/middleware"
	"ops-kb-rag/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type K8sHandler struct {
	service *service.K8sService
}

func NewK8sHandler(service *service.K8sService) *K8sHandler {
	return &K8sHandler{service: service}
}

func (h *K8sHandler) ListClusters(c *gin.Context) {
	items, err := h.service.ListClusters(c.Request.Context())
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, items)
}

func (h *K8sHandler) CreateCluster(c *gin.Context) {
	var req dto.SaveK8sClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	item, err := h.service.CreateCluster(c.Request.Context(), req, middleware.CurrentUsername(c))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, item)
}

func (h *K8sHandler) UpdateCluster(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	var req dto.SaveK8sClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	item, err := h.service.UpdateCluster(c.Request.Context(), id, req)
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, item)
}

func (h *K8sHandler) DeleteCluster(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	if err := h.service.DeleteCluster(c.Request.Context(), id); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, gin.H{"deleted": true})
}

func (h *K8sHandler) TestCluster(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	if err := h.service.TestCluster(c.Request.Context(), id); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, dto.TestK8sClusterResponse{OK: true, Message: "连接成功"})
}

func (h *K8sHandler) DiagnosePod(c *gin.Context) {
	var req dto.K8sPodDiagnosisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	result, err := h.service.DiagnosePod(c.Request.Context(), req, middleware.CurrentUsername(c))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, result)
}

func (h *K8sHandler) DiagnoseAlert(c *gin.Context) {
	var req dto.K8sAlertDiagnosisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	result, err := h.service.DiagnoseAlert(c.Request.Context(), req, middleware.CurrentUsername(c))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, result)
}

func (h *K8sHandler) DiagnoseIngress(c *gin.Context) {
	var req dto.K8sResourceDiagnosisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	result, err := h.service.DiagnoseIngress(c.Request.Context(), req, middleware.CurrentUsername(c))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, result)
}

func (h *K8sHandler) DiagnoseService(c *gin.Context) {
	var req dto.K8sResourceDiagnosisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	result, err := h.service.DiagnoseService(c.Request.Context(), req, middleware.CurrentUsername(c))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, result)
}

func parseID(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 64)
}
