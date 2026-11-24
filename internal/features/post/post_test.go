package post

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/database"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPostTest(t *testing.T) (*gin.Engine, *gorm.DB, *Service, *Handler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := database.ConnectTest()
	require.NoError(t, err)

	require.NoError(t, db.Create(&database.User{
		ID: 1, Username: "admin", Email: "admin@delcom.org", Password: "x", Role: "admin",
	}).Error)
	require.NoError(t, db.Create(&database.User{
		ID: 2, Username: "author1", Email: "author1@delcom.org", Password: "x", Role: "member",
	}).Error)
	require.NoError(t, db.Create(&database.User{
		ID: 3, Username: "author2", Email: "author2@delcom.org", Password: "x", Role: "member",
	}).Error)

	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	r := gin.New()
	injectUser := func(c *gin.Context) {
		uidHeader := c.GetHeader("X-Test-User-ID")
		roleHeader := c.GetHeader("X-Test-Role")
		if uidHeader == "1" {
			c.Set(middleware.ContextUserKey, &middleware.UserClaims{UserID: 1, Username: "admin", Role: roleHeader})
		} else if uidHeader == "2" {
			c.Set(middleware.ContextUserKey, &middleware.UserClaims{UserID: 2, Username: "author1", Role: roleHeader})
		} else if uidHeader == "3" {
			c.Set(middleware.ContextUserKey, &middleware.UserClaims{UserID: 3, Username: "author2", Role: roleHeader})
		}
		c.Next()
	}

	r.GET("/api/public/categories", h.ListCategories)
	r.GET("/api/public/posts", h.ListPublicPosts)
	r.GET("/api/public/posts/:slug", h.GetPublicPostBySlug)

	r.GET("/api/posts", injectUser, h.ListMyPosts)
	r.POST("/api/posts", injectUser, h.CreatePost)
	r.GET("/api/posts/stats", injectUser, h.GetMyStats)
	r.GET("/api/posts/:id", injectUser, h.GetMyPost)
	r.PUT("/api/posts/:id", injectUser, h.UpdatePost)
	r.PATCH("/api/posts/:id/toggle-publish", injectUser, h.TogglePublish)
	r.DELETE("/api/posts/:id", injectUser, h.DeletePost)

	r.GET("/api/admin/users", h.AdminListUsers)
	r.GET("/api/admin/posts", h.AdminListPosts)
	r.GET("/api/admin/users/:id/posts", h.AdminListUserPosts)

	return r, db, svc, h
}

func doPostReq(r *gin.Engine, method, path string, body any, userID, role string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		if raw, ok := body.(string); ok {
			buf.WriteString(raw)
		} else {
			_ = json.NewEncoder(&buf).Encode(body)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-Test-User-ID", userID)
		req.Header.Set("X-Test-Role", role)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSlugifyAndReadingTime(t *testing.T) {
	assert.Equal(t, "belajar-golang-gin", Slugify("Belajar Golang & Gin!!"))
	assert.Equal(t, "artikel", Slugify("!!!"))
	assert.Equal(t, 1, CalculateReadingTime(""))
	assert.Equal(t, 1, CalculateReadingTime("hello world"))
	assert.Equal(t, 2, CalculateReadingTime(strings.Repeat("word ", 205)))
}

func TestPostFullFlow(t *testing.T) {
	r, _, _, _ := setupPostTest(t)

	// Categories
	w := doPostReq(r, http.MethodGet, "/api/public/categories", nil, "", "")
	assert.Equal(t, http.StatusOK, w.Code)

	// Unauthorized calls
	for _, tc := range []struct{ m, p string }{
		{http.MethodGet, "/api/posts"},
		{http.MethodPost, "/api/posts"},
		{http.MethodGet, "/api/posts/stats"},
		{http.MethodGet, "/api/posts/1"},
		{http.MethodPut, "/api/posts/1"},
		{http.MethodPatch, "/api/posts/1/toggle-publish"},
		{http.MethodDelete, "/api/posts/1"},
	} {
		w := doPostReq(r, tc.m, tc.p, nil, "", "")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}

	// Create bad payload
	w = doPostReq(r, http.MethodPost, "/api/posts", `{}`, "2", "member")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Create published post with default category
	w = doPostReq(r, http.MethodPost, "/api/posts", CreatePostRequest{
		Title:       "Belajar Go Gin",
		Excerpt:     "Ringkasan Go",
		Content:     "Isi lengkap artikel Golang Gin",
		IsPublished: true,
	}, "2", "member")
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp struct {
		Data PostDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	assert.Equal(t, "belajar-go-gin", createResp.Data.Slug)
	assert.Equal(t, "Umum", createResp.Data.Category)
	assert.True(t, createResp.Data.IsPublished)
	assert.NotNil(t, createResp.Data.PublishedAt)
	post1ID := createResp.Data.ID

	// Duplicate title generates unique slug (-2)
	w = doPostReq(r, http.MethodPost, "/api/posts", CreatePostRequest{
		Title:       "Belajar Go Gin",
		Category:    "Teknologi",
		Content:     "Artikel draft kedua",
		IsPublished: false,
	}, "2", "member")
	require.Equal(t, http.StatusCreated, w.Code)
	var createResp2 struct {
		Data PostDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp2))
	assert.Equal(t, "belajar-go-gin-2", createResp2.Data.Slug)
	post2ID := createResp2.Data.ID

	// Public list with search & category (must not expose user_id, must include author_name)
	w = doPostReq(r, http.MethodGet, "/api/public/posts?q=golang&category=Umum", nil, "", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "user_id")
	assert.Contains(t, w.Body.String(), `"author_name":"author1"`)

	// Public detail found & not found
	w = doPostReq(r, http.MethodGet, "/api/public/posts/belajar-go-gin", nil, "", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "user_id")
	assert.Contains(t, w.Body.String(), `"author_name":"author1"`)
	w = doPostReq(r, http.MethodGet, "/api/public/posts/tidak-ada", nil, "", "")
	assert.Equal(t, http.StatusNotFound, w.Code)

	// My posts & stats (verifies total_articles, total_selesai, belum_selesai)
	w = doPostReq(r, http.MethodGet, "/api/posts", nil, "2", "member")
	assert.Equal(t, http.StatusOK, w.Code)
	w = doPostReq(r, http.MethodGet, "/api/posts/stats", nil, "2", "member")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "total_articles")
	assert.Contains(t, w.Body.String(), "total_selesai")
	assert.Contains(t, w.Body.String(), "belum_selesai")

	// Invalid ID params
	for _, tc := range []struct{ m, p string }{
		{http.MethodGet, "/api/posts/abc"},
		{http.MethodPut, "/api/posts/0"},
		{http.MethodPatch, "/api/posts/abc/toggle-publish"},
		{http.MethodDelete, "/api/posts/abc"},
		{http.MethodGet, "/api/admin/users/abc/posts"},
	} {
		w := doPostReq(r, tc.m, tc.p, nil, "2", "member")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	}

	// GetMyPost: owner ok, other user forbidden, missing not found, admin ok
	w = doPostReq(r, http.MethodGet, fmt.Sprintf("/api/posts/%d", post1ID), nil, "2", "member")
	assert.Equal(t, http.StatusOK, w.Code)
	w = doPostReq(r, http.MethodGet, fmt.Sprintf("/api/posts/%d", post1ID), nil, "3", "member")
	assert.Equal(t, http.StatusForbidden, w.Code)
	w = doPostReq(r, http.MethodGet, "/api/posts/999", nil, "2", "member")
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = doPostReq(r, http.MethodGet, fmt.Sprintf("/api/posts/%d", post1ID), nil, "1", "admin")
	assert.Equal(t, http.StatusOK, w.Code)

	// UpdatePost: invalid json, update all fields + unpublish, then republish
	w = doPostReq(r, http.MethodPut, fmt.Sprintf("/api/posts/%d", post1ID), "{bad", "2", "member")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	pubFalse := false
	w = doPostReq(r, http.MethodPut, fmt.Sprintf("/api/posts/%d", post1ID), UpdatePostRequest{
		Title:          "Judul Baru",
		Category:       "Teknologi",
		Excerpt:        "Excerpt baru",
		Content:        "Konten baru",
		CoverImagePath: "/img/cover.png",
		IsPublished:    &pubFalse,
	}, "2", "member")
	assert.Equal(t, http.StatusOK, w.Code)

	pubTrue := true
	w = doPostReq(r, http.MethodPut, fmt.Sprintf("/api/posts/%d", post1ID), UpdatePostRequest{
		IsPublished: &pubTrue,
	}, "2", "member")
	assert.Equal(t, http.StatusOK, w.Code)

	// Update forbidden
	w = doPostReq(r, http.MethodPut, fmt.Sprintf("/api/posts/%d", post1ID), UpdatePostRequest{Title: "Hack"}, "3", "member")
	assert.Equal(t, http.StatusForbidden, w.Code)

	// TogglePublish (publish draft post2 -> true, then back to false)
	w = doPostReq(r, http.MethodPatch, fmt.Sprintf("/api/posts/%d/toggle-publish", post2ID), nil, "2", "member")
	assert.Equal(t, http.StatusOK, w.Code)
	w = doPostReq(r, http.MethodPatch, fmt.Sprintf("/api/posts/%d/toggle-publish", post2ID), nil, "2", "member")
	assert.Equal(t, http.StatusOK, w.Code)
	w = doPostReq(r, http.MethodPatch, "/api/posts/999/toggle-publish", nil, "2", "member")
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Admin endpoints (non-existent user returns 200 with empty list)
	w = doPostReq(r, http.MethodGet, "/api/admin/users", nil, "1", "admin")
	assert.Equal(t, http.StatusOK, w.Code)
	w = doPostReq(r, http.MethodGet, "/api/admin/posts", nil, "1", "admin")
	assert.Equal(t, http.StatusOK, w.Code)
	w = doPostReq(r, http.MethodGet, "/api/admin/users/2/posts", nil, "1", "admin")
	assert.Equal(t, http.StatusOK, w.Code)
	w = doPostReq(r, http.MethodGet, "/api/admin/users/999/posts", nil, "1", "admin")
	assert.Equal(t, http.StatusOK, w.Code)

	// DeletePost
	w = doPostReq(r, http.MethodDelete, fmt.Sprintf("/api/posts/%d", post2ID), nil, "3", "member")
	assert.Equal(t, http.StatusForbidden, w.Code)
	w = doPostReq(r, http.MethodDelete, fmt.Sprintf("/api/posts/%d", post2ID), nil, "2", "member")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPostErrorPaths(t *testing.T) {
	t.Run("closed db triggers 500 across handlers", func(t *testing.T) {
		r, db, _, _ := setupPostTest(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())

		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodGet, "/api/public/posts", nil, "", "").Code)
		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodGet, "/api/public/posts/any", nil, "", "").Code)
		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodGet, "/api/posts", nil, "2", "member").Code)
		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodPost, "/api/posts", CreatePostRequest{Title: "A", Content: "B"}, "2", "member").Code)
		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodGet, "/api/posts/stats", nil, "2", "member").Code)
		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodGet, "/api/posts/1", nil, "2", "member").Code)
		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodPut, "/api/posts/1", UpdatePostRequest{Title: "X"}, "2", "member").Code)
		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodPatch, "/api/posts/1/toggle-publish", nil, "2", "member").Code)
		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodDelete, "/api/posts/1", nil, "2", "member").Code)
		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodGet, "/api/admin/users", nil, "1", "admin").Code)
		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodGet, "/api/admin/posts", nil, "1", "admin").Code)
		assert.Equal(t, http.StatusInternalServerError, doPostReq(r, http.MethodGet, "/api/admin/users/2/posts", nil, "1", "admin").Code)
	})

	t.Run("repository FindAllUsersWithStats fails when posts table dropped", func(t *testing.T) {
		_, db, _, _ := setupPostTest(t)
		repo := NewRepository(db)
		require.NoError(t, db.Exec("DROP TABLE posts").Error)
		_, err := repo.FindAllUsersWithStats()
		assert.Error(t, err)
	})

	t.Run("repository GetStatsByUserID second query fails via callback", func(t *testing.T) {
		_, db, _, _ := setupPostTest(t)
		repo := NewRepository(db)
		countCalls := 0
		err := db.Callback().Query().Before("gorm:query").Register("fail_second_count", func(tx *gorm.DB) {
			countCalls++
			if countCalls == 2 {
				_ = tx.AddError(errors.New("second count failed"))
			}
		})
		require.NoError(t, err)
		_, err = repo.GetStatsByUserID(2)
		assert.Error(t, err)
	})

	t.Run("service granular repo error branches", func(t *testing.T) {
		dbErr := errors.New("db error")

		// CreatePost create error
		svc := NewService(&mockPostRepo{createErr: dbErr})
		_, err := svc.CreatePost(1, CreatePostRequest{Title: "A", Content: "B"})
		assert.Equal(t, dbErr, err)

		// CreatePost FindByID error after create
		svc = NewService(&mockPostRepo{findByIDErr: dbErr})
		_, err = svc.CreatePost(1, CreatePostRequest{Title: "A", Content: "B"})
		assert.Equal(t, dbErr, err)

		// UpdatePost slug check error
		svc = NewService(&mockPostRepo{
			postByID:     &database.Post{ID: 1, UserID: 2, Title: "Old"},
			slugCheckErr: dbErr,
		})
		_, err = svc.UpdatePost(2, "member", 1, UpdatePostRequest{Title: "New"})
		assert.Equal(t, dbErr, err)

		// UpdatePost update error
		svc = NewService(&mockPostRepo{
			postByID:  &database.Post{ID: 1, UserID: 2, Title: "Old"},
			updateErr: dbErr,
		})
		_, err = svc.UpdatePost(2, "member", 1, UpdatePostRequest{Content: "Updated"})
		assert.Equal(t, dbErr, err)

		// TogglePublish update error
		svc = NewService(&mockPostRepo{
			postByID:  &database.Post{ID: 1, UserID: 2, Title: "Old"},
			updateErr: dbErr,
		})
		_, err = svc.TogglePublish(2, "member", 1)
		assert.Equal(t, dbErr, err)

		// AdminListUserPosts FindByUserID error
		svc = NewService(&mockPostRepo{
			findByUserErr: dbErr,
		})
		_, err = svc.AdminListUserPosts(2)
		assert.Equal(t, dbErr, err)
	})
}

type mockPostRepo struct {
	postByID      *database.Post
	findByIDErr   error
	slugCheckErr  error
	createErr     error
	updateErr     error
	findByUserErr error
}

func (m *mockPostRepo) FindPublicPosts(string, string) ([]database.Post, error) { return nil, nil }
func (m *mockPostRepo) FindPublicPostBySlug(string) (*database.Post, error)     { return nil, nil }
func (m *mockPostRepo) FindByUserID(uint) ([]database.Post, error)              { return nil, m.findByUserErr }
func (m *mockPostRepo) FindByID(uint) (*database.Post, error)                   { return m.postByID, m.findByIDErr }
func (m *mockPostRepo) SlugExists(string, uint) (bool, error)                   { return false, m.slugCheckErr }
func (m *mockPostRepo) Create(*database.Post) error                             { return m.createErr }
func (m *mockPostRepo) Update(*database.Post) error                             { return m.updateErr }
func (m *mockPostRepo) Delete(uint) error                                       { return nil }
func (m *mockPostRepo) GetStatsByUserID(uint) (*PostStatsDTO, error)            { return nil, nil }
func (m *mockPostRepo) FindAllUsersWithStats() ([]AdminUserDTO, error)          { return nil, nil }
func (m *mockPostRepo) FindAllPosts() ([]database.Post, error)                  { return nil, nil }
