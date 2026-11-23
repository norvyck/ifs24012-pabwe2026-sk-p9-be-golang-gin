package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"app/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const ContextUserKey = "user"

type UserClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func Auth(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.AbortFail(c, http.StatusUnauthorized, "Authorization token is required")
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		claims := &UserClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		})
		if err != nil || !token.Valid {
			response.AbortFail(c, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		c.Set(ContextUserKey, claims)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		val, exists := c.Get(ContextUserKey)
		if !exists {
			response.AbortFail(c, http.StatusUnauthorized, "Unauthorized access")
			return
		}
		claims, ok := val.(*UserClaims)
		if !ok {
			response.AbortFail(c, http.StatusUnauthorized, "Invalid user context")
			return
		}
		if _, ok := allowed[claims.Role]; !ok {
			response.AbortFail(c, http.StatusForbidden, "Forbidden: insufficient role permissions")
			return
		}
		c.Next()
	}
}

type rateEntry struct {
	count   int
	resetAt time.Time
}

func RateLimiter(maxRequests int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*rateEntry)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		entry, exists := clients[ip]
		if !exists || now.After(entry.resetAt) {
			clients[ip] = &rateEntry{
				count:   1,
				resetAt: now.Add(window),
			}
			mu.Unlock()
			c.Next()
			return
		}

		entry.count++
		if entry.count > maxRequests {
			mu.Unlock()
			response.AbortFail(c, http.StatusTooManyRequests, "Too many requests, please try again later")
			return
		}
		mu.Unlock()
		c.Next()
	}
}
