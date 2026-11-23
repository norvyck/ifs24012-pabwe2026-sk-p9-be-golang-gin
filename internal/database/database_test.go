package database

import (
	"errors"
	"testing"

	"app/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestConnect(t *testing.T) {
	t.Run("sqlite with empty database name defaults to memory", func(t *testing.T) {
		db, err := Connect(&config.Config{
			DBConnection: "sqlite",
			DBDatabase:   "",
		})
		require.NoError(t, err)
		assert.NotNil(t, db)
	})

	t.Run("postgres uses gormOpen", func(t *testing.T) {
		origOpen := gormOpen
		defer func() { gormOpen = origOpen }()

		gormOpen = func(dialector gorm.Dialector, opts ...gorm.Option) (*gorm.DB, error) {
			return nil, errors.New("connection refused")
		}

		db, err := Connect(&config.Config{
			DBConnection: "pgsql",
			DBHost:       "127.0.0.1",
			DBPort:       "5432",
			DBDatabase:   "pabwe_p9_blog",
			DBUsername:   "postgres",
			DBPassword:   "postgres",
			DBSSLMode:    "disable",
		})
		assert.Nil(t, db)
		assert.EqualError(t, err, "connection refused")
	})
}

func TestConnectTestAndMigrate(t *testing.T) {
	db, err := ConnectTest()
	require.NoError(t, err)
	assert.NotNil(t, db)

	t.Run("open error", func(t *testing.T) {
		origOpen := gormOpen
		defer func() { gormOpen = origOpen }()
		gormOpen = func(dialector gorm.Dialector, opts ...gorm.Option) (*gorm.DB, error) {
			return nil, errors.New("open failed")
		}
		_, err := ConnectTest()
		assert.EqualError(t, err, "open failed")
	})

	t.Run("migrate error on closed db", func(t *testing.T) {
		origOpen := gormOpen
		defer func() { gormOpen = origOpen }()
		gormOpen = func(dialector gorm.Dialector, opts ...gorm.Option) (*gorm.DB, error) {
			d, err := origOpen(dialector, opts...)
			require.NoError(t, err)
			sqlDB, err := d.DB()
			require.NoError(t, err)
			_ = sqlDB.Close()
			return d, nil
		}
		_, err := ConnectTest()
		assert.Error(t, err)
	})
}

func TestSeed(t *testing.T) {
	db, err := ConnectTest()
	require.NoError(t, err)

	cfg := &config.Config{}
	require.NoError(t, Seed(db, cfg))

	var users []User
	require.NoError(t, db.Order("id asc").Find(&users).Error)
	assert.Len(t, users, 2)
	assert.Equal(t, "admin", users[0].Username)
	assert.Equal(t, "admin", users[0].Role)
	assert.Equal(t, "member", users[1].Username)
	assert.Equal(t, "member", users[1].Role)

	// Idempotent second call
	require.NoError(t, Seed(db, cfg))
	var count int64
	require.NoError(t, db.Model(&User{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)

	// Promotes existing admin user back to admin role if demoted
	require.NoError(t, db.Model(&User{}).Where("username = ?", "admin").Update("role", "member").Error)
	require.NoError(t, Seed(db, cfg))
	var promoted User
	require.NoError(t, db.Where("username = ?", "admin").First(&promoted).Error)
	assert.Equal(t, "admin", promoted.Role)

	// Syncs admin password if changed in config
	require.NoError(t, Seed(db, &config.Config{AdminUsername: "admin", AdminPassword: "newadminpass"}))
	require.NoError(t, Seed(db, cfg))

	t.Run("hash error on admin password sync", func(t *testing.T) {
		origHash := hashPassword
		defer func() { hashPassword = origHash }()
		hashPassword = func(password []byte, cost int) ([]byte, error) {
			return nil, errors.New("sync hash error")
		}
		err = Seed(db, &config.Config{AdminUsername: "admin", AdminPassword: "differentpass"})
		assert.EqualError(t, err, "sync hash error")
	})

	t.Run("hash error on admin seed", func(t *testing.T) {
		freshDB, err := ConnectTest()
		require.NoError(t, err)
		origHash := hashPassword
		defer func() { hashPassword = origHash }()
		hashPassword = func(password []byte, cost int) ([]byte, error) {
			return nil, errors.New("hash error")
		}
		err = Seed(freshDB, &config.Config{AdminUsername: "admin", AdminPassword: "admin"})
		assert.EqualError(t, err, "hash error")
	})

	t.Run("hash error on member seed", func(t *testing.T) {
		freshDB, err := ConnectTest()
		require.NoError(t, err)
		origHash := hashPassword
		defer func() { hashPassword = origHash }()
		calls := 0
		hashPassword = func(password []byte, cost int) ([]byte, error) {
			calls++
			if calls == 2 {
				return nil, errors.New("member hash error")
			}
			return origHash(password, cost)
		}
		err = Seed(freshDB, &config.Config{AdminUsername: "admin", AdminPassword: "admin"})
		assert.EqualError(t, err, "member hash error")
	})

	t.Run("db error on count", func(t *testing.T) {
		closedDB, err := ConnectTest()
		require.NoError(t, err)
		sqlDB, err := closedDB.DB()
		require.NoError(t, err)
		_ = sqlDB.Close()
		err = Seed(closedDB, &config.Config{})
		assert.Error(t, err)
	})
}
