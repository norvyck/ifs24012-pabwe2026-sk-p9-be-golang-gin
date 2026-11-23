package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/config"
	"app/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := database.ConnectTest()
	require.NoError(t, err)

	cfg := &config.Config{
		AppKey:               "test-secret-key",
		SearchEngineEndpoint: "http://localhost:9999/search",
		OpenAIEndpoint:       "http://localhost:9999/v1",
		OpenAIKey:            "test-key",
		OpenAIModel:          "test-model",
	}

	r := NewRouter(db, cfg)

	t.Run("public categories route", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/categories", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("api.json http and https forwarded proto", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api.json", nil)
		req.Host = "localhost:8081"
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"http"`)

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/api.json", nil)
		req2.Host = "blog.delcom.org"
		req2.Header.Set("X-Forwarded-Proto", "https")
		r.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Contains(t, w2.Body.String(), `"https"`)
	})

	t.Run("swagger ui route", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("404 NoRoute handler", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/does-not-exist", nil))
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Endpoint not found")
	})
}
