package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"image-ai/backend/internal/config"
	"image-ai/backend/internal/db"
	"image-ai/backend/internal/handler"
	"image-ai/backend/internal/repository"
	"image-ai/backend/internal/server"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	repo := repository.New(pool)
	h := handler.New(cfg, repo)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.NewRouter(cfg, h),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("image-ai api listening on %s", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}
