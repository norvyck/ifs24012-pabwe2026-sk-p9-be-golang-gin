package post

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"app/internal/database"

	"gorm.io/gorm"
)

var (
	ErrPostNotFound = errors.New("post not found")
	ErrForbidden    = errors.New("forbidden: you do not own this post")
	ErrUserNotFound = errors.New("user not found")

	nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9]+`)
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func Slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = nonAlphaNumRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "artikel"
	}
	return s
}

func CalculateReadingTime(content string) int {
	words := len(strings.Fields(content))
	mins := (words + 199) / 200
	if mins < 1 {
		return 1
	}
	return mins
}

func (s *Service) ListCategories() []string {
	return Categories
}

func (s *Service) ListPublicPosts(query, category string) ([]PostDTO, error) {
	posts, err := s.repo.FindPublicPosts(query, category)
	if err != nil {
		return nil, err
	}
	return mapPostsToDTO(posts), nil
}

func (s *Service) GetPublicPostBySlug(slug string) (*PostDTO, error) {
	p, err := s.repo.FindPublicPostBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	dto := toPostDTO(p)
	return &dto, nil
}

func (s *Service) ListMyPosts(userID uint) ([]PostDTO, error) {
	posts, err := s.repo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	return mapPostsToDTO(posts), nil
}

func (s *Service) CreatePost(userID uint, req CreatePostRequest) (*PostDTO, error) {
	title := strings.TrimSpace(req.Title)
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Umum"
	}

	slug, err := s.generateUniqueSlug(title, 0)
	if err != nil {
		return nil, err
	}

	var publishedAt *time.Time
	if req.IsPublished {
		now := time.Now()
		publishedAt = &now
	}

	p := &database.Post{
		UserID:         userID,
		Title:          title,
		Slug:           slug,
		Category:       category,
		Excerpt:        strings.TrimSpace(req.Excerpt),
		Content:        req.Content,
		CoverImagePath: strings.TrimSpace(req.CoverImagePath),
		IsPublished:    req.IsPublished,
		PublishedAt:    publishedAt,
	}

	if err := s.repo.Create(p); err != nil {
		return nil, err
	}

	loaded, err := s.repo.FindByID(p.ID)
	if err != nil {
		return nil, err
	}
	dto := toPostDTO(loaded)
	return &dto, nil
}

func (s *Service) GetMyPost(userID uint, role string, postID uint) (*PostDTO, error) {
	p, err := s.findAuthorizedPost(userID, role, postID)
	if err != nil {
		return nil, err
	}
	dto := toPostDTO(p)
	return &dto, nil
}

func (s *Service) UpdatePost(userID uint, role string, postID uint, req UpdatePostRequest) (*PostDTO, error) {
	p, err := s.findAuthorizedPost(userID, role, postID)
	if err != nil {
		return nil, err
	}

	if title := strings.TrimSpace(req.Title); title != "" && title != p.Title {
		slug, err := s.generateUniqueSlug(title, p.ID)
		if err != nil {
			return nil, err
		}
		p.Title = title
		p.Slug = slug
	}

	if category := strings.TrimSpace(req.Category); category != "" {
		p.Category = category
	}
	if req.Excerpt != "" {
		p.Excerpt = strings.TrimSpace(req.Excerpt)
	}
	if req.Content != "" {
		p.Content = req.Content
	}
	if req.CoverImagePath != "" {
		p.CoverImagePath = strings.TrimSpace(req.CoverImagePath)
	}
	if req.IsPublished != nil {
		p.IsPublished = *req.IsPublished
		if *req.IsPublished && p.PublishedAt == nil {
			now := time.Now()
			p.PublishedAt = &now
		} else if !*req.IsPublished {
			p.PublishedAt = nil
		}
	}

	if err := s.repo.Update(p); err != nil {
		return nil, err
	}

	dto := toPostDTO(p)
	return &dto, nil
}

func (s *Service) TogglePublish(userID uint, role string, postID uint) (*PostDTO, error) {
	p, err := s.findAuthorizedPost(userID, role, postID)
	if err != nil {
		return nil, err
	}

	p.IsPublished = !p.IsPublished
	if p.IsPublished {
		now := time.Now()
		p.PublishedAt = &now
	} else {
		p.PublishedAt = nil
	}

	if err := s.repo.Update(p); err != nil {
		return nil, err
	}

	dto := toPostDTO(p)
	return &dto, nil
}

func (s *Service) DeletePost(userID uint, role string, postID uint) error {
	if _, err := s.findAuthorizedPost(userID, role, postID); err != nil {
		return err
	}
	return s.repo.Delete(postID)
}

func (s *Service) GetMyStats(userID uint) (*PostStatsDTO, error) {
	return s.repo.GetStatsByUserID(userID)
}

func (s *Service) AdminListUsers() ([]AdminUserDTO, error) {
	return s.repo.FindAllUsersWithStats()
}

func (s *Service) AdminListPosts() ([]PostDTO, error) {
	posts, err := s.repo.FindAllPosts()
	if err != nil {
		return nil, err
	}
	return mapPostsToDTO(posts), nil
}

func (s *Service) AdminListUserPosts(targetUserID uint) ([]PostDTO, error) {
	exists, err := s.repo.UserExists(targetUserID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}
	posts, err := s.repo.FindByUserID(targetUserID)
	if err != nil {
		return nil, err
	}
	return mapPostsToDTO(posts), nil
}

func (s *Service) findAuthorizedPost(userID uint, role string, postID uint) (*database.Post, error) {
	p, err := s.repo.FindByID(postID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	if p.UserID != userID && role != "admin" {
		return nil, ErrForbidden
	}
	return p, nil
}

func (s *Service) generateUniqueSlug(title string, excludeID uint) (string, error) {
	base := Slugify(title)
	candidate := base
	for i := 2; ; i++ {
		exists, err := s.repo.SlugExists(candidate, excludeID)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func mapPostsToDTO(posts []database.Post) []PostDTO {
	result := make([]PostDTO, 0, len(posts))
	for i := range posts {
		result = append(result, toPostDTO(&posts[i]))
	}
	return result
}
