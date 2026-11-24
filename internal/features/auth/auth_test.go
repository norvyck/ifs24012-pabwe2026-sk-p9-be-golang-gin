package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/database"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthTest(t *testing.T) (*gin.Engine, *Service, *Handler, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := database.ConnectTest()
	require.NoError(t, err)

	repo := NewRepository(db)
	svc := NewService(repo, "test-secret")
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/api/auth/register", h.Register)
	r.POST("/api/auth/login", h.Login)
	r.POST("/api/auth/logout", middleware.Auth("test-secret"), h.Logout)
	r.GET("/api/auth/profile", middleware.Auth("test-secret"), h.GetProfile)
	r.PUT("/api/auth/profile", middleware.Auth("test-secret"), h.UpdateProfile)
	r.GET("/api/auth/profile-unauth", h.GetProfile)
	r.PUT("/api/auth/profile-unauth", h.UpdateProfile)

	closeDB := func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}
	return r, svc, h, closeDB
}

func postJSON(r *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
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
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthFullFlow(t *testing.T) {
	r, _, _, _ := setupAuthTest(t)

	// Invalid register payload
	w := postJSON(r, http.MethodPost, "/api/auth/register", `{"username":"a"}`, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Register user 1
	w = postJSON(r, http.MethodPost, "/api/auth/register", RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "password123",
	}, "")
	require.Equal(t, http.StatusCreated, w.Code)

	var regResp struct {
		Status string       `json:"status"`
		Data   AuthResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	assert.Equal(t, "success", regResp.Status)
	assert.NotEmpty(t, regResp.Data.Token)
	assert.Equal(t, "alice", regResp.Data.User.Username)
	token := regResp.Data.Token

	// Duplicate username (400)
	w = postJSON(r, http.MethodPost, "/api/auth/register", RegisterRequest{
		Username: "alice",
		Email:    "other@example.com",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Duplicate email (400)
	w = postJSON(r, http.MethodPost, "/api/auth/register", RegisterRequest{
		Username: "bob",
		Email:    "alice@example.com",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Short password rejected (400)
	w = postJSON(r, http.MethodPost, "/api/auth/register", RegisterRequest{
		Username: "shortpw",
		Email:    "short@example.com",
		Password: "12345",
	}, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Register user 2 for conflict tests on update
	w = postJSON(r, http.MethodPost, "/api/auth/register", RegisterRequest{
		Username: "bob",
		Email:    "bob@example.com",
		Password: "password123",
	}, "")
	require.Equal(t, http.StatusCreated, w.Code)

	// Register admin user when role is specified
	w = postJSON(r, http.MethodPost, "/api/auth/register", RegisterRequest{
		Username: "newadmin",
		Email:    "newadmin@example.com",
		Password: "password123",
		Role:     "admin",
	}, "")
	require.Equal(t, http.StatusCreated, w.Code)
	var adminRegResp struct {
		Status string       `json:"status"`
		Data   AuthResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &adminRegResp))
	assert.Equal(t, "admin", adminRegResp.Data.User.Role)

	// Login bad payload (missing password, missing identifier)
	w = postJSON(r, http.MethodPost, "/api/auth/login", `{}`, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = postJSON(r, http.MethodPost, "/api/auth/login", LoginRequest{Password: "password123"}, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Login unknown user
	w = postJSON(r, http.MethodPost, "/api/auth/login", LoginRequest{
		Username: "unknown",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Login wrong password
	w = postJSON(r, http.MethodPost, "/api/auth/login", LoginRequest{
		Username: "alice",
		Password: "wrong",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Login success with username
	w = postJSON(r, http.MethodPost, "/api/auth/login", LoginRequest{
		Username: "alice",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusOK, w.Code)

	// Login success with email in username field and in email field
	w = postJSON(r, http.MethodPost, "/api/auth/login", LoginRequest{
		Username: "alice@example.com",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusOK, w.Code)
	w = postJSON(r, http.MethodPost, "/api/auth/login", LoginRequest{
		Email:    "alice@example.com",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusOK, w.Code)

	// Logout
	w = postJSON(r, http.MethodPost, "/api/auth/logout", nil, token)
	assert.Equal(t, http.StatusOK, w.Code)

	// Get profile success
	w = postJSON(r, http.MethodGet, "/api/auth/profile", nil, token)
	assert.Equal(t, http.StatusOK, w.Code)

	// Get/Update profile without claims
	w = postJSON(r, http.MethodGet, "/api/auth/profile-unauth", nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = postJSON(r, http.MethodPut, "/api/auth/profile-unauth", UpdateProfileRequest{}, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Update profile invalid JSON & invalid email
	w = postJSON(r, http.MethodPut, "/api/auth/profile", "{invalid", token)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = postJSON(r, http.MethodPut, "/api/auth/profile", UpdateProfileRequest{Email: "not-an-email"}, token)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Update profile duplicate username (400)
	w = postJSON(r, http.MethodPut, "/api/auth/profile", UpdateProfileRequest{Username: "bob"}, token)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Update profile duplicate email (400)
	w = postJSON(r, http.MethodPut, "/api/auth/profile", UpdateProfileRequest{Email: "bob@example.com"}, token)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Update profile success (username, email, password)
	w = postJSON(r, http.MethodPut, "/api/auth/profile", UpdateProfileRequest{
		Username: "alice_new",
		Email:    "alice_new@example.com",
		Password: "newpassword",
	}, token)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify login with new credentials
	w = postJSON(r, http.MethodPost, "/api/auth/login", LoginRequest{
		Username: "alice_new",
		Password: "newpassword",
	}, "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthServiceAndHandlerErrorPaths(t *testing.T) {
	t.Run("user not found on profile endpoints", func(t *testing.T) {
		r, svc, _, _ := setupAuthTest(t)
		ghostToken, err := svc.generateToken(&database.User{ID: 999, Username: "ghost", Role: "member"})
		require.NoError(t, err)

		w := postJSON(r, http.MethodGet, "/api/auth/profile", nil, ghostToken)
		assert.Equal(t, http.StatusNotFound, w.Code)

		w = postJSON(r, http.MethodPut, "/api/auth/profile", UpdateProfileRequest{Username: "newname"}, ghostToken)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("hash and sign errors", func(t *testing.T) {
		r, svc, _, _ := setupAuthTest(t)

		// Hash error on register
		origHash := svc.hashFn
		svc.hashFn = func([]byte, int) ([]byte, error) {
			return nil, errors.New("hash failed")
		}
		w := postJSON(r, http.MethodPost, "/api/auth/register", RegisterRequest{
			Username: "user1", Email: "user1@example.com", Password: "password",
		}, "")
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		svc.hashFn = origHash

		// Sign error on register
		origSign := svc.signFn
		svc.signFn = func(*jwt.Token, any) (string, error) {
			return "", errors.New("sign failed")
		}
		w = postJSON(r, http.MethodPost, "/api/auth/register", RegisterRequest{
			Username: "user2", Email: "user2@example.com", Password: "password",
		}, "")
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Sign error on login (user2 was saved before sign failed)
		w = postJSON(r, http.MethodPost, "/api/auth/login", LoginRequest{
			Username: "user2", Password: "password",
		}, "")
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		svc.signFn = origSign

		// Login valid to get token, then test hash error on update
		w = postJSON(r, http.MethodPost, "/api/auth/login", LoginRequest{
			Username: "user2", Password: "password",
		}, "")
		require.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data AuthResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		svc.hashFn = func([]byte, int) ([]byte, error) {
			return nil, errors.New("hash update failed")
		}
		w = postJSON(r, http.MethodPut, "/api/auth/profile", UpdateProfileRequest{Password: "newpass"}, resp.Data.Token)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		svc.hashFn = origHash
	})

	t.Run("repository errors via mock", func(t *testing.T) {
		dbErr := errors.New("db failure")

		// Error on FindByUsername during register
		svc := NewService(&mockAuthRepo{findByUsernameErr: dbErr}, "secret")
		_, err := svc.Register(RegisterRequest{Username: "a", Email: "a@b.com", Password: "pass"})
		assert.Equal(t, dbErr, err)

		// Error on FindByEmail during register
		svc = NewService(&mockAuthRepo{
			findByUsernameErr: errors.New("notfound"),
		}, "secret")
		_ = svc

		// Use closed DB for genuine repo error tests
		r, s2, _, closeDB := setupAuthTest(t)
		tok, err := s2.generateToken(&database.User{ID: 1, Username: "x", Role: "member"})
		require.NoError(t, err)
		closeDB()

		w := postJSON(r, http.MethodPost, "/api/auth/register", RegisterRequest{
			Username: "closed", Email: "closed@example.com", Password: "password",
		}, "")
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		w = postJSON(r, http.MethodPost, "/api/auth/login", LoginRequest{
			Username: "closed", Password: "password",
		}, "")
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		w = postJSON(r, http.MethodGet, "/api/auth/profile", nil, tok)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		w = postJSON(r, http.MethodPut, "/api/auth/profile", UpdateProfileRequest{Username: "validname"}, tok)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("granular repo errors in service", func(t *testing.T) {
		dbErr := errors.New("db error")

		// Login FindByEmail error when FindByUsername returns not found
		svcEmailErr := NewService(&mockAuthRepo{
			findByUsernameNotFound: true,
			findByEmailErr:         dbErr,
		}, "secret")
		_, err := svcEmailErr.Login(LoginRequest{Username: "x@y.com", Password: "password123"})
		assert.Equal(t, dbErr, err)

		// Register FindByEmail error
		svc := NewService(&mockAuthRepo{
			findByUsernameNotFound: true,
			findByEmailErr:         dbErr,
		}, "secret")
		_, err = svc.Register(RegisterRequest{Username: "u", Email: "u@e.com", Password: "pw"})
		assert.Equal(t, dbErr, err)

		// Register Create error
		svc = NewService(&mockAuthRepo{
			findByUsernameNotFound: true,
			findByEmailNotFound:    true,
			createErr:              dbErr,
		}, "secret")
		_, err = svc.Register(RegisterRequest{Username: "u", Email: "u@e.com", Password: "pw"})
		assert.Equal(t, dbErr, err)

		// UpdateProfile FindByUsername error
		svc = NewService(&mockAuthRepo{
			userByID:          &database.User{ID: 1, Username: "old", Email: "old@e.com"},
			findByUsernameErr: dbErr,
		}, "secret")
		_, err = svc.UpdateProfile(1, UpdateProfileRequest{Username: "new"})
		assert.Equal(t, dbErr, err)

		// UpdateProfile FindByEmail error
		svc = NewService(&mockAuthRepo{
			userByID:       &database.User{ID: 1, Username: "old", Email: "old@e.com"},
			findByEmailErr: dbErr,
		}, "secret")
		_, err = svc.UpdateProfile(1, UpdateProfileRequest{Email: "new@e.com"})
		assert.Equal(t, dbErr, err)

		// UpdateProfile Update error
		svc = NewService(&mockAuthRepo{
			userByID:  &database.User{ID: 1, Username: "old", Email: "old@e.com"},
			updateErr: dbErr,
		}, "secret")
		_, err = svc.UpdateProfile(1, UpdateProfileRequest{})
		assert.Equal(t, dbErr, err)
	})
}

type mockAuthRepo struct {
	userByID               *database.User
	findByUsernameNotFound bool
	findByEmailNotFound    bool
	findByUsernameErr      error
	findByEmailErr         error
	createErr              error
	updateErr              error
}

func (m *mockAuthRepo) Create(user *database.User) error { return m.createErr }
func (m *mockAuthRepo) FindByUsername(username string) (*database.User, error) {
	if m.findByUsernameNotFound {
		return nil, errors.Join(errors.New("wrapped"), databaseErrNotFound())
	}
	return nil, m.findByUsernameErr
}
func (m *mockAuthRepo) FindByEmail(email string) (*database.User, error) {
	if m.findByEmailNotFound {
		return nil, databaseErrNotFound()
	}
	return nil, m.findByEmailErr
}
func (m *mockAuthRepo) FindByID(id uint) (*database.User, error) { return m.userByID, nil }
func (m *mockAuthRepo) Update(user *database.User) error         { return m.updateErr }

func databaseErrNotFound() error {
	db, _ := database.ConnectTest()
	var u database.User
	return db.First(&u, 99999).Error
}
