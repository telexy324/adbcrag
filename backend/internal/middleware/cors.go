package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	config := cors.Config{
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Authorization", "X-User"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}
	origins := corsAllowOrigins()
	if len(origins) == 1 && origins[0] == "*" {
		config.AllowAllOrigins = true
	} else {
		config.AllowOrigins = origins
		config.AllowCredentials = true
	}
	return cors.New(config)
}

func corsAllowOrigins() []string {
	value := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if value == "" {
		return []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			origins = append(origins, part)
		}
	}
	if len(origins) == 0 {
		return []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}
	return origins
}
