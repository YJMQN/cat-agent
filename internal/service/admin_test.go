package service

import (
	"testing"

	"cat-agent/internal/domain"
	"cat-agent/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminServiceListMemoriesWithoutUserIDReturnsAllMemories(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite memory db: %v", err)
	}

	if err := db.AutoMigrate(&domain.Memory{}); err != nil {
		t.Fatalf("migrate memory table: %v", err)
	}

	repo := repository.New(db)
	svc := NewAdminService(repo)

	if err := repo.Memory.Create(&domain.Memory{UserID: 1, Category: "profile", Key: "name", Content: "Alice", Source: "auto"}); err != nil {
		t.Fatalf("create first memory: %v", err)
	}
	if err := repo.Memory.Create(&domain.Memory{UserID: 2, Category: "preference", Key: "theme", Content: "dark", Source: "manual"}); err != nil {
		t.Fatalf("create second memory: %v", err)
	}

	memories, err := svc.ListMemories(0)
	if err != nil {
		t.Fatalf("list memories: %v", err)
	}
	if len(memories) != 2 {
		t.Fatalf("expected 2 memories, got %d", len(memories))
	}
}
