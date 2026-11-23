package auth

import (
	"errors"
	"net/http"
	"strings"

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

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid registration data")
		return
	}

	res, err := h.service.Register(req)
	if err != nil {
		if errors.Is(err, ErrUsernameExists) || errors.Is(err, ErrEmailExists) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to register user")
		return
	}

	response.Success(c, http.StatusCreated, "Registration successful", res)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Username or email and password are required")
		return
	}
	if strings.TrimSpace(req.Username) == "" && strings.TrimSpace(req.Email) == "" {
		response.Fail(c, http.StatusBadRequest, "Username or email and password are required")
		return
	}

	res, err := h.service.Login(req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Fail(c, http.StatusUnauthorized, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to login")
		return
	}

	response.Success(c, http.StatusOK, "Login successful", res)
}

func (h *Handler) Logout(c *gin.Context) {
	response.Success(c, http.StatusOK, "Logout successful", nil)
}

func (h *Handler) GetProfile(c *gin.Context) {
	claims, ok := getUserClaims(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	profile, err := h.service.GetProfile(claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to load profile")
		return
	}

	response.Success(c, http.StatusOK, "Profile retrieved", profile)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	claims, ok := getUserClaims(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid profile payload")
		return
	}

	profile, err := h.service.UpdateProfile(claims.UserID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			response.Fail(c, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrUsernameExists), errors.Is(err, ErrEmailExists):
			response.Fail(c, http.StatusBadRequest, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "Failed to update profile")
		}
		return
	}

	response.Success(c, http.StatusOK, "Profile updated", profile)
}

func getUserClaims(c *gin.Context) (*middleware.UserClaims, bool) {
	val, exists := c.Get(middleware.ContextUserKey)
	if !exists {
		return nil, false
	}
	claims, ok := val.(*middleware.UserClaims)
	return claims, ok
}
