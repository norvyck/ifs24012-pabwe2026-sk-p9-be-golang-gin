package post

import (
	"errors"
	"net/http"
	"strconv"

	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListCategories(c *gin.Context) {
	response.Success(c, http.StatusOK, "Categories retrieved", h.service.ListCategories())
}

func (h *Handler) ListPublicPosts(c *gin.Context) {
	q := c.Query("q")
	category := c.Query("category")

	posts, err := h.service.ListPublicPosts(q, category)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to load public posts")
		return
	}
	response.Success(c, http.StatusOK, "Public posts retrieved", posts)
}

func (h *Handler) GetPublicPostBySlug(c *gin.Context) {
	slug := c.Param("slug")
	p, err := h.service.GetPublicPostBySlug(slug)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to load post")
		return
	}
	response.Success(c, http.StatusOK, "Post retrieved", p)
}

func (h *Handler) ListMyPosts(c *gin.Context) {
	claims, ok := getUserClaims(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	posts, err := h.service.ListMyPosts(claims.UserID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to load posts")
		return
	}
	response.Success(c, http.StatusOK, "Author posts retrieved", posts)
}

func (h *Handler) CreatePost(c *gin.Context) {
	claims, ok := getUserClaims(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Title and content are required")
		return
	}

	p, err := h.service.CreatePost(claims.UserID, req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create post")
		return
	}
	response.Success(c, http.StatusCreated, "Post created", p)
}

func (h *Handler) GetMyStats(c *gin.Context) {
	claims, ok := getUserClaims(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	stats, err := h.service.GetMyStats(claims.UserID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to load post statistics")
		return
	}
	response.Success(c, http.StatusOK, "Post statistics retrieved", stats)
}

func (h *Handler) GetMyPost(c *gin.Context) {
	claims, ok := getUserClaims(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	p, err := h.service.GetMyPost(claims.UserID, claims.Role, id)
	if err != nil {
		handlePostError(c, err, "Failed to load post")
		return
	}
	response.Success(c, http.StatusOK, "Post retrieved", p)
}

func (h *Handler) UpdatePost(c *gin.Context) {
	claims, ok := getUserClaims(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid post update payload")
		return
	}

	p, err := h.service.UpdatePost(claims.UserID, claims.Role, id, req)
	if err != nil {
		handlePostError(c, err, "Failed to update post")
		return
	}
	response.Success(c, http.StatusOK, "Post updated", p)
}

func (h *Handler) TogglePublish(c *gin.Context) {
	claims, ok := getUserClaims(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	p, err := h.service.TogglePublish(claims.UserID, claims.Role, id)
	if err != nil {
		handlePostError(c, err, "Failed to toggle publish status")
		return
	}
	response.Success(c, http.StatusOK, "Post publication status updated", p)
}

func (h *Handler) DeletePost(c *gin.Context) {
	claims, ok := getUserClaims(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.service.DeletePost(claims.UserID, claims.Role, id); err != nil {
		handlePostError(c, err, "Failed to delete post")
		return
	}
	response.Success(c, http.StatusOK, "Post deleted", nil)
}

func (h *Handler) AdminListUsers(c *gin.Context) {
	users, err := h.service.AdminListUsers()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to load users")
		return
	}
	response.Success(c, http.StatusOK, "Users retrieved", users)
}

func (h *Handler) AdminListPosts(c *gin.Context) {
	posts, err := h.service.AdminListPosts()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to load all posts")
		return
	}
	response.Success(c, http.StatusOK, "All posts retrieved", posts)
}

func (h *Handler) AdminListUserPosts(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	posts, err := h.service.AdminListUserPosts(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to load user posts")
		return
	}
	response.Success(c, http.StatusOK, "User posts retrieved", posts)
}

func getUserClaims(c *gin.Context) (*middleware.UserClaims, bool) {
	val, exists := c.Get(middleware.ContextUserKey)
	if !exists {
		return nil, false
	}
	claims, ok := val.(*middleware.UserClaims)
	return claims, ok
}

func parseUintParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || parsed == 0 {
		response.Fail(c, http.StatusBadRequest, "Invalid ID parameter")
		return 0, false
	}
	return uint(parsed), true
}

func handlePostError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, ErrPostNotFound):
		response.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Fail(c, http.StatusForbidden, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, fallback)
	}
}
