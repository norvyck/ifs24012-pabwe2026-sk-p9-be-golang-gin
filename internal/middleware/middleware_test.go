package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	t.Run("regular request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("preflight options", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestAuthAndRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"

	makeToken := func(role string, method jwt.SigningMethod, key any) string {
		claims := &UserClaims{
			UserID:   1,
			Username: "alice",
			Email:    "alice@example.com",
			Role:     role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		tok := jwt.NewWithClaims(method, claims)
		signed, err := tok.SignedString(key)
		require.NoError(t, err)
		return signed
	}

	r := gin.New()
	r.GET("/protected", Auth(secret), func(c *gin.Context) {
		claims := c.MustGet(ContextUserKey).(*UserClaims)
		c.JSON(http.StatusOK, gin.H{"user": claims.Username})
	})
	r.GET("/admin", Auth(secret), RequireRole("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/role-no-auth", RequireRole("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/role-bad-ctx", func(c *gin.Context) {
		c.Set(ContextUserKey, "not-claims")
		c.Next()
	}, RequireRole("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	t.Run("missing header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/protected", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("wrong signing method none", func(t *testing.T) {
		tok := makeToken("member", jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("valid token", func(t *testing.T) {
		tok := makeToken("member", jwt.SigningMethodHS256, []byte(secret))
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("role forbidden", func(t *testing.T) {
		tok := makeToken("member", jwt.SigningMethodHS256, []byte(secret))
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("role allowed", func(t *testing.T) {
		tok := makeToken("admin", jwt.SigningMethodHS256, []byte(secret))
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("role without auth context", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/role-no-auth", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("role with invalid context type", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/role-bad-ctx", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimiter(2, 30*time.Millisecond))
	r.GET("/limited", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/limited", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/limited", nil))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	time.Sleep(40 * time.Millisecond)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/limited", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}
