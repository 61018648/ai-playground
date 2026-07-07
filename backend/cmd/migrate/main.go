package main

import (
	"context"
	"log"
	"time"

	"image-ai/backend/internal/config"
	"image-ai/backend/internal/db"
	"image-ai/backend/internal/install"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := install.EnsureDatabase(ctx, cfg.DatabaseURL); err != nil {
		log.Fatalf("ensure database: %v", err)
	}

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	files, err := install.ApplyMigrations(ctx, pool, "migrations")
	if err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	for _, file := range files {
		log.Printf("applied %s", file)
	}

	log.Println("migration completed")
}
