package post

import (
	"time"

	"app/internal/database"
)

var Categories = []string{
	"Teknologi",
	"Pendidikan",
	"Gaya Hidup",
	"Bisnis",
	"Opini",
	"Umum",
}

type CreatePostRequest struct {
	Title          string `json:"title" binding:"required"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content" binding:"required"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    bool   `json:"is_published"`
}

type UpdatePostRequest struct {
	Title          string `json:"title"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    *bool  `json:"is_published"`
}

type AuthorInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
}

type PublicPostDTO struct {
	ID             uint        `json:"id"`
	AuthorName     string      `json:"author_name"`
	Author         *AuthorInfo `json:"author,omitempty"`
	Title          string      `json:"title"`
	Slug           string      `json:"slug"`
	Category       string      `json:"category"`
	Excerpt        string      `json:"excerpt"`
	Content        string      `json:"content"`
	CoverImagePath string      `json:"cover_image_path"`
	IsPublished    bool        `json:"is_published"`
	PublishedAt    *time.Time  `json:"published_at"`
	ReadingTime    int         `json:"reading_time"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type PostDTO struct {
	ID             uint        `json:"id"`
	UserID         uint        `json:"user_id"`
	AuthorName     string      `json:"author_name"`
	Author         *AuthorInfo `json:"author,omitempty"`
	Title          string      `json:"title"`
	Slug           string      `json:"slug"`
	Category       string      `json:"category"`
	Excerpt        string      `json:"excerpt"`
	Content        string      `json:"content"`
	CoverImagePath string      `json:"cover_image_path"`
	IsPublished    bool        `json:"is_published"`
	PublishedAt    *time.Time  `json:"published_at"`
	ReadingTime    int         `json:"reading_time"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type PostStatsDTO struct {
	TotalArticles int64 `json:"total_articles"`
	TotalSelesai  int64 `json:"total_selesai"`
	BelumSelesai  int64 `json:"belum_selesai"`
	Total         int64 `json:"total"`
	Published     int64 `json:"published"`
	Draft         int64 `json:"draft"`
}

type AdminUserDTO struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	TotalArticles  int64     `json:"total_articles"`
	TotalSelesai   int64     `json:"total_selesai"`
	BelumSelesai   int64     `json:"belum_selesai"`
	TotalPosts     int64     `json:"total_posts"`
	PublishedPosts int64     `json:"published_posts"`
	CreatedAt      time.Time `json:"created_at"`
}

func toPublicPostDTO(p *database.Post) PublicPostDTO {
	dto := PublicPostDTO{
		ID:             p.ID,
		AuthorName:     p.User.Username,
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		Content:        p.Content,
		CoverImagePath: p.CoverImagePath,
		IsPublished:    p.IsPublished,
		PublishedAt:    p.PublishedAt,
		ReadingTime:    CalculateReadingTime(p.Content),
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
	if p.User.ID != 0 {
		dto.Author = &AuthorInfo{
			ID:       p.User.ID,
			Username: p.User.Username,
		}
	}
	return dto
}

func toPostDTO(p *database.Post) PostDTO {
	dto := PostDTO{
		ID:             p.ID,
		UserID:         p.UserID,
		AuthorName:     p.User.Username,
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		Content:        p.Content,
		CoverImagePath: p.CoverImagePath,
		IsPublished:    p.IsPublished,
		PublishedAt:    p.PublishedAt,
		ReadingTime:    CalculateReadingTime(p.Content),
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
	if p.User.ID != 0 {
		dto.Author = &AuthorInfo{
			ID:       p.User.ID,
			Username: p.User.Username,
			Email:    p.User.Email,
			Role:     p.User.Role,
		}
	}
	return dto
}
