package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/config"
	"app/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

func TestRun(t *testing.T) {
	origLoad := loadConfigFn
	origConnect := connectDBFn
	origMigrate := migrateDBFn
	origSeed := seedDBFn
	origListen := listenFn
	origFatal := fatalFn
	defer func() {
		loadConfigFn = origLoad
		connectDBFn = origConnect
		migrateDBFn = origMigrate
		seedDBFn = origSeed
		listenFn = origListen
		fatalFn = origFatal
		gin.SetMode(gin.TestMode)
	}()

	// Cover default listenFn error path with invalid address
	assert.Error(t, origListen(gin.New(), "invalid:addr:99999"))

	var lastFatal string
	fatalFn = func(format string, v ...any) {
		lastFatal = format
	}

	loadConfigFn = func() *config.Config {
		return &config.Config{AppEnv: "production", AppPort: "8081", AppKey: "k"}
	}

	// Connect error
	connectDBFn = func(*config.Config) (*gorm.DB, error) { return nil, errors.New("conn err") }
	Run()
	assert.Contains(t, lastFatal, "Failed to connect database")

	db, err := database.ConnectTest()
	require.NoError(t, err)
	connectDBFn = func(*config.Config) (*gorm.DB, error) { return db, nil }

	// Migrate error
	migrateDBFn = func(*gorm.DB) error { return errors.New("mig err") }
	Run()
	assert.Contains(t, lastFatal, "Failed to migrate database")
	migrateDBFn = func(*gorm.DB) error { return nil }

	// Seed error
	seedDBFn = func(*gorm.DB, *config.Config) error { return errors.New("seed err") }
	Run()
	assert.Contains(t, lastFatal, "Failed to seed database")
	seedDBFn = func(*gorm.DB, *config.Config) error { return nil }

	// Listen error
	listenFn = func(*gin.Engine, string) error { return errors.New("listen err") }
	Run()
	assert.Contains(t, lastFatal, "Server stopped with error")

	// Success
	lastFatal = ""
	listenFn = func(*gin.Engine, string) error { return nil }
	Run()
	assert.Empty(t, lastFatal)
}
