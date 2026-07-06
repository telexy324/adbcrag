package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"ops-kb-rag/backend/internal/logger"

	"github.com/gin-gonic/gin"
)

func Recover() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error(
			c.Request.Context(),
			"panic recovered",
			"error", fmt.Sprint(recovered),
			"stack", string(debug.Stack()),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
