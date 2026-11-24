package post

import (
	"strings"

	"app/internal/database"

	"gorm.io/gorm"
)

type Repository interface {
	FindPublicPosts(query, category string) ([]database.Post, error)
	FindPublicPostBySlug(slug string) (*database.Post, error)
	FindByUserID(userID uint) ([]database.Post, error)
	FindByID(id uint) (*database.Post, error)
	SlugExists(slug string, excludeID uint) (bool, error)
	Create(post *database.Post) error
	Update(post *database.Post) error
	Delete(id uint) error
	GetStatsByUserID(userID uint) (*PostStatsDTO, error)
	FindAllUsersWithStats() ([]AdminUserDTO, error)
	FindAllPosts() ([]database.Post, error)
}

type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) FindPublicPosts(query, category string) ([]database.Post, error) {
	tx := r.db.Preload("User").Where("is_published = ?", true)
	if cat := strings.TrimSpace(category); cat != "" && cat != "Semua" {
		tx = tx.Where("category = ?", cat)
	}
	if q := strings.TrimSpace(query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		tx = tx.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(content) LIKE ?", like, like, like)
	}
	var posts []database.Post
	if err := tx.Order("published_at DESC, created_at DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *gormRepository) FindPublicPostBySlug(slug string) (*database.Post, error) {
	var p database.Post
	if err := r.db.Preload("User").Where("slug = ? AND is_published = ?", slug, true).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) FindByUserID(userID uint) ([]database.Post, error) {
	var posts []database.Post
	if err := r.db.Preload("User").Where("user_id = ?", userID).Order("created_at DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *gormRepository) FindByID(id uint) (*database.Post, error) {
	var p database.Post
	if err := r.db.Preload("User").First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) SlugExists(slug string, excludeID uint) (bool, error) {
	var count int64
	tx := r.db.Model(&database.Post{}).Where("slug = ?", slug)
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	if err := tx.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *gormRepository) Create(post *database.Post) error {
	return r.db.Create(post).Error
}

func (r *gormRepository) Update(post *database.Post) error {
	return r.db.Save(post).Error
}

func (r *gormRepository) Delete(id uint) error {
	return r.db.Delete(&database.Post{}, id).Error
}

func (r *gormRepository) GetStatsByUserID(userID uint) (*PostStatsDTO, error) {
	var total int64
	if err := r.db.Model(&database.Post{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, err
	}
	var published int64
	if err := r.db.Model(&database.Post{}).Where("user_id = ? AND is_published = ?", userID, true).Count(&published).Error; err != nil {
		return nil, err
	}
	draft := total - published
	return &PostStatsDTO{
		TotalArticles: total,
		TotalSelesai:  published,
		BelumSelesai:  draft,
		Total:         total,
		Published:     published,
		Draft:         draft,
	}, nil
}

func (r *gormRepository) FindAllUsersWithStats() ([]AdminUserDTO, error) {
	var users []database.User
	if err := r.db.Order("id ASC").Find(&users).Error; err != nil {
		return nil, err
	}

	result := make([]AdminUserDTO, 0, len(users))
	for _, u := range users {
		stats, err := r.GetStatsByUserID(u.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, AdminUserDTO{
			ID:                u.ID,
			Username:          u.Username,
			Email:             u.Email,
			Role:              u.Role,
			TotalArticles:     stats.TotalArticles,
			TotalPublished:    stats.Published,
			PublishedArticles: stats.Published,
			PublishedCount:    stats.Published,
			TotalSelesai:      stats.TotalSelesai,
			BelumSelesai:      stats.BelumSelesai,
			TotalPosts:        stats.Total,
			PublishedPosts:    stats.Published,
			CreatedAt:         u.CreatedAt,
		})
	}
	return result, nil
}

func (r *gormRepository) FindAllPosts() ([]database.Post, error) {
	var posts []database.Post
	if err := r.db.Preload("User").Order("created_at DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}
