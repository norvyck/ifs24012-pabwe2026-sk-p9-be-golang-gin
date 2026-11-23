package database

import (
	"time"

	"app/internal/config"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"size:255;uniqueIndex;not null" json:"username"`
	Email     string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"type:text;not null" json:"-"`
	Role      string    `gorm:"size:50;not null;default:member" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Post struct {
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	User           User       `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
	Title          string     `gorm:"size:255;not null" json:"title"`
	Slug           string     `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Category       string     `gorm:"size:100;not null;default:Umum" json:"category"`
	Excerpt        string     `gorm:"type:text" json:"excerpt"`
	Content        string     `gorm:"type:text;not null" json:"content"`
	CoverImagePath string     `gorm:"type:text" json:"cover_image_path"`
	IsPublished    bool       `gorm:"not null;default:false" json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

var (
	gormOpen     = gorm.Open
	hashPassword = bcrypt.GenerateFromPassword
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if cfg.DBConnection == "sqlite" {
		dbName := cfg.DBDatabase
		if dbName == "" {
			dbName = ":memory:"
		}
		dialector = sqlite.Open(dbName)
	} else {
		dialector = postgres.Open(cfg.DSN())
	}

	return gormOpen(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func ConnectTest() (*gorm.DB, error) {
	db, err := gormOpen(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	db.Exec("PRAGMA foreign_keys = ON;")
	if err := Migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Post{})
}

func Seed(db *gorm.DB, cfg *config.Config) error {
	adminUser := cfg.AdminUsername
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := cfg.AdminPassword
	if adminPass == "" {
		adminPass = "admin"
	}

	if err := seedUser(db, adminUser, "admin@delcom.org", adminPass, "admin"); err != nil {
		return err
	}
	if err := seedUser(db, "member", "member@delcom.org", "member", "member"); err != nil {
		return err
	}
	return nil
}

func seedUser(db *gorm.DB, username, email, password, role string) error {
	var count int64
	if err := db.Model(&User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hashed, err := hashPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := User{
		Username: username,
		Email:    email,
		Password: string(hashed),
		Role:     role,
	}
	return db.Create(&user).Error
}
