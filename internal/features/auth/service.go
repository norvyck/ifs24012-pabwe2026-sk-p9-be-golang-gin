package auth

import (
	"errors"
	"strings"
	"time"

	"app/internal/database"
	"app/internal/middleware"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUsernameExists     = errors.New("username already registered")
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
)

type Service struct {
	repo      Repository
	secretKey string
	hashFn    func([]byte, int) ([]byte, error)
	signFn    func(*jwt.Token, any) (string, error)
}

func NewService(repo Repository, secretKey string) *Service {
	return &Service{
		repo:      repo,
		secretKey: secretKey,
		hashFn:    bcrypt.GenerateFromPassword,
		signFn: func(t *jwt.Token, key any) (string, error) {
			return t.SignedString(key)
		},
	}
}

func (s *Service) Register(req RegisterRequest) (*AuthResponse, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(strings.ToLower(req.Email))

	if _, err := s.repo.FindByUsername(username); err == nil {
		return nil, ErrUsernameExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if _, err := s.repo.FindByEmail(email); err == nil {
		return nil, ErrEmailExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashed, err := s.hashFn([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	role := "member"
	if strings.EqualFold(strings.TrimSpace(req.Role), "admin") {
		role = "admin"
	}

	user := &database.User{
		Username: username,
		Email:    email,
		Password: string(hashed),
		Role:     role,
	}
	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  toUserDTO(user),
	}, nil
}

func (s *Service) Login(req LoginRequest) (*AuthResponse, error) {
	identifier := strings.TrimSpace(req.Username)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Email)
	}

	user, err := s.repo.FindByUsername(identifier)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		user, err = s.repo.FindByEmail(strings.ToLower(identifier))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrInvalidCredentials
			}
			return nil, err
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  toUserDTO(user),
	}, nil
}

func (s *Service) GetProfile(userID uint) (*UserDTO, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	dto := toUserDTO(user)
	return &dto, nil
}

func (s *Service) UpdateProfile(userID uint, req UpdateProfileRequest) (*UserDTO, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if username := strings.TrimSpace(req.Username); username != "" && username != user.Username {
		if _, err := s.repo.FindByUsername(username); err == nil {
			return nil, ErrUsernameExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		user.Username = username
	}

	if email := strings.TrimSpace(strings.ToLower(req.Email)); email != "" && email != user.Email {
		if _, err := s.repo.FindByEmail(email); err == nil {
			return nil, ErrEmailExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		user.Email = email
	}

	if req.Password != "" {
		hashed, err := s.hashFn([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(hashed)
	}

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}

	dto := toUserDTO(user)
	return &dto, nil
}

func (s *Service) generateToken(user *database.User) (string, error) {
	claims := &middleware.UserClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return s.signFn(token, []byte(s.secretKey))
}
