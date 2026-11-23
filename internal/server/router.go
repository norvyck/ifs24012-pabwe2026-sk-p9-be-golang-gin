package server

import (
	"net/http"
	"time"

	"app/docs"
	"app/internal/config"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimiter(300, time.Minute))

	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, cfg.AppKey)
	authHandler := auth.NewHandler(authService)

	postRepo := post.NewRepository(db)
	postService := post.NewService(postRepo)
	postHandler := post.NewHandler(postService)

	searchClient := ai.NewHTTPSearchClient(cfg.SearchEngineEndpoint, cfg.OpenAIKey)
	openAIClient := ai.NewHTTPOpenAIClient(cfg.OpenAIEndpoint, cfg.OpenAIKey, cfg.OpenAIModel)
	aiService := ai.NewService(searchClient, openAIClient)
	aiHandler := ai.NewHandler(aiService)

	// Swagger & OpenAPI spec
	r.GET("/api.json", func(c *gin.Context) {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(docs.GetSwaggerJSON(c.Request.Host, scheme)))
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api.json")))

	api := r.Group("/api")
	{
		// Public endpoints
		public := api.Group("/public")
		{
			public.GET("/categories", postHandler.ListCategories)
			public.GET("/posts", postHandler.ListPublicPosts)
			public.GET("/posts/:slug", postHandler.GetPublicPostBySlug)
		}

		// Auth endpoints
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/logout", middleware.Auth(cfg.AppKey), authHandler.Logout)
			authGroup.GET("/profile", middleware.Auth(cfg.AppKey), authHandler.GetProfile)
			authGroup.PUT("/profile", middleware.Auth(cfg.AppKey), authHandler.UpdateProfile)
		}

		// Author post endpoints
		postsGroup := api.Group("/posts", middleware.Auth(cfg.AppKey))
		{
			postsGroup.GET("", postHandler.ListMyPosts)
			postsGroup.POST("", postHandler.CreatePost)
			postsGroup.GET("/stats", postHandler.GetMyStats)
			postsGroup.POST("/ai-generate", aiHandler.GeneratePost)
			postsGroup.GET("/:id", postHandler.GetMyPost)
			postsGroup.PUT("/:id", postHandler.UpdatePost)
			postsGroup.PATCH("/:id/toggle-publish", postHandler.TogglePublish)
			postsGroup.DELETE("/:id", postHandler.DeletePost)
		}

		// Admin endpoints
		adminGroup := api.Group("/admin", middleware.Auth(cfg.AppKey), middleware.RequireRole("admin"))
		{
			adminGroup.GET("/users", postHandler.AdminListUsers)
			adminGroup.GET("/posts", postHandler.AdminListPosts)
			adminGroup.GET("/users/:id/posts", postHandler.AdminListUserPosts)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, "Endpoint not found")
	})

	return r
}
