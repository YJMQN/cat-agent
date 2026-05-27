package repository

import (
	"testing"

	"cat-agent/internal/domain"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite memory db: %v", err)
	}

	if err := db.AutoMigrate(&domain.User{}); err != nil {
		t.Fatalf("migrate user table: %v", err)
	}

	return db
}

func TestInitDefaultAdminCreatesValidDefaultPassword(t *testing.T) {
	db := setupTestDB(t)

	initDefaultAdmin(db)

	var user domain.User
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatalf("find admin user: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("admin123")); err != nil {
		t.Fatalf("default admin password should be valid: %v", err)
	}
}

func TestInitDefaultAdminRepairsKnownBadSeedHash(t *testing.T) {
	db := setupTestDB(t)

	badHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	if err := db.Create(&domain.User{Username: "admin", Password: badHash, Role: "admin"}).Error; err != nil {
		t.Fatalf("create seeded admin: %v", err)
	}

	initDefaultAdmin(db)

	var user domain.User
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatalf("find admin user after repair: %v", err)
	}

	if user.Password == badHash {
		t.Fatalf("expected admin password hash to be repaired")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("admin123")); err != nil {
		t.Fatalf("repaired admin password should validate: %v", err)
	}
}
