package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfigDefaults(t *testing.T) {
	keys := []string{
		"APP_ENV", "APP_PORT", "PORT", "APP_KEY",
		"ADMIN_USERNAME", "ADMIN_PASSWORD",
		"DB_CONNECTION", "DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD", "DB_SSLMODE",
		"SEARCH_ENGINE_ENDPOINT", "OPENAI_ENDPOINT", "OPENAI_KEY", "OPENAI_MODEL",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}

	cfg := LoadConfig()
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "8081", cfg.AppPort)
	assert.Equal(t, "admin", cfg.AdminUsername)
	assert.Equal(t, "admin", cfg.AdminPassword)
	assert.Equal(t, "pgsql", cfg.DBConnection)
	assert.Equal(t, "127.0.0.1", cfg.DBHost)
	assert.Equal(t, "5432", cfg.DBPort)
	assert.Equal(t, "pabwe_p9_blog", cfg.DBDatabase)
	assert.Equal(t, "postgres", cfg.DBUsername)
	assert.Equal(t, "postgres", cfg.DBPassword)
	assert.Equal(t, "disable", cfg.DBSSLMode)
	assert.Equal(t, "https://delcom.org/api/v1/search", cfg.SearchEngineEndpoint)
	assert.Equal(t, "https://delcom.org/api/v1", cfg.OpenAIEndpoint)
	assert.Equal(t, "key", cfg.OpenAIKey)
	assert.Equal(t, "nama-model", cfg.OpenAIModel)
	assert.Contains(t, cfg.DSN(), "dbname=pabwe_p9_blog")
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("APP_KEY", "custom-key")
	t.Setenv("ADMIN_USERNAME", "superadmin")
	t.Setenv("ADMIN_PASSWORD", "secret123")
	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_DATABASE", "custom_db")
	t.Setenv("DB_USERNAME", "dbuser")
	t.Setenv("DB_PASSWORD", "dbpass")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("SEARCH_ENGINE_ENDPOINT", "https://search.example.com")
	t.Setenv("OPENAI_ENDPOINT", "https://ai.example.com")
	t.Setenv("OPENAI_KEY", "sk-test")
	t.Setenv("OPENAI_MODEL", "gpt-4o-mini")

	cfg := LoadConfig()
	assert.Equal(t, "production", cfg.AppEnv)
	assert.Equal(t, "9090", cfg.AppPort)
	assert.Equal(t, "custom-key", cfg.AppKey)
	assert.Equal(t, "superadmin", cfg.AdminUsername)
	assert.Equal(t, "secret123", cfg.AdminPassword)
	assert.Equal(t, "sqlite", cfg.DBConnection)
	assert.Equal(t, "https://search.example.com", cfg.SearchEngineEndpoint)
	assert.Equal(t, "https://ai.example.com", cfg.OpenAIEndpoint)
	assert.Equal(t, "sk-test", cfg.OpenAIKey)
	assert.Equal(t, "gpt-4o-mini", cfg.OpenAIModel)
	assert.Equal(
		t,
		"host=localhost user=dbuser password=dbpass dbname=custom_db port=5433 sslmode=require TimeZone=Asia/Jakarta",
		cfg.DSN(),
	)

	t.Setenv("APP_PORT", "")
	t.Setenv("PORT", "8085")
	assert.Equal(t, "8085", LoadConfig().AppPort)
}
