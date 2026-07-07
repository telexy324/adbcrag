package middleware

import (
	"strings"

	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserID   = "current_user_id"
	ContextUsername = "current_username"
	ContextUserRole = "current_user_role"
)

func Auth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			dto.Error(c, 401, "missing authorization token")
			c.Abort()
			return
		}
		claims, err := auth.Verify(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			dto.Error(c, 401, err.Error())
			c.Abort()
			return
		}
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUsername, claims.Username)
		c.Set(ContextUserRole, claims.Role)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUserRole(c) != model.UserRoleAdmin {
			dto.Error(c, 403, "admin permission required")
			c.Abort()
			return
		}
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) uint64 {
	value, _ := c.Get(ContextUserID)
	if id, ok := value.(uint64); ok {
		return id
	}
	return 0
}

func CurrentUsername(c *gin.Context) string {
	value, _ := c.Get(ContextUsername)
	if username, ok := value.(string); ok {
		return username
	}
	return ""
}

func CurrentUserRole(c *gin.Context) string {
	value, _ := c.Get(ContextUserRole)
	if role, ok := value.(string); ok {
		return role
	}
	return ""
}
