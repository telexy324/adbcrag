package dto

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, APIResponse{Code: 0, Message: "success", Data: data})
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, APIResponse{Code: status, Message: message, Data: nil})
}
