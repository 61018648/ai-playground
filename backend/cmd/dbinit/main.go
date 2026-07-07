package main

import (
	"context"
	"log"
	"time"

	"image-ai/backend/internal/config"
	"image-ai/backend/internal/install"
)

func main() {
	cfg := config.Load()
	opts := install.DefaultOptions(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := install.Install(ctx, opts)
	if err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}

	for _, file := range result.AppliedMigrations {
		log.Printf("applied %s", file)
	}
	log.Printf("admin ready: %s", result.AdminEmail)
	log.Println("database initialization completed")
}
