package handler

import (
	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/middleware"
	"ops-kb-rag/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	result, err := h.service.Login(c.Request.Context(), req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		dto.Error(c, 401, err.Error())
		return
	}
	dto.Success(c, result)
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		dto.Error(c, 404, err.Error())
		return
	}
	dto.Success(c, user)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), middleware.CurrentUserID(c), req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, gin.H{"changed": true})
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	users, err := h.service.ListUsers(c.Request.Context())
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, users)
}

func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req dto.SaveUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	user, err := h.service.CreateUser(c.Request.Context(), req, middleware.CurrentUserID(c))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, user)
}

func (h *AuthHandler) UpdateUser(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	var req dto.SaveUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	user, err := h.service.UpdateUser(c.Request.Context(), id, req)
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, user)
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), id, req.Password); err != nil {
		dto.Error(c, 400, err.Error())
		return
	}
	dto.Success(c, gin.H{"reset": true})
}
