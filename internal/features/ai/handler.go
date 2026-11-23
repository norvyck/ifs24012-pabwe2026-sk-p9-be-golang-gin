package ai

import (
	"net/http"

	"app/internal/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GeneratePost(c *gin.Context) {
	var req GeneratePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Topic is required")
		return
	}

	result, err := h.service.GeneratePost(req)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Failed to generate article with AI: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Article generated successfully", result)
}
