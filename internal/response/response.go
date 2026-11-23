package response

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func Success(c *gin.Context, code int, message string, data any) {
	c.JSON(code, APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

func Fail(c *gin.Context, code int, message string) {
	c.JSON(code, APIResponse{
		Status:  "fail",
		Message: message,
	})
}

func Error(c *gin.Context, code int, message string) {
	c.JSON(code, APIResponse{
		Status:  "error",
		Message: message,
	})
}

func AbortFail(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, APIResponse{
		Status:  "fail",
		Message: message,
	})
}

func AbortError(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, APIResponse{
		Status:  "error",
		Message: message,
	})
}
