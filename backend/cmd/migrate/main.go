package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"image-ai/backend/internal/config"
	"image-ai/backend/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := ensureDatabase(ctx, cfg.DatabaseURL); err != nil {
		log.Fatalf("ensure database: %v", err)
	}

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		log.Fatalf("list migrations: %v", err)
	}
	sort.Strings(files)

	for _, file := range files {
		sql, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			log.Fatalf("run migration %s: %v", file, err)
		}
		log.Printf("applied %s", file)
	}

	log.Println("migration completed")
}

func ensureDatabase(ctx context.Context, databaseURL string) error {
	targetURL, err := url.Parse(databaseURL)
	if err != nil {
		return err
	}
	dbName := strings.TrimPrefix(targetURL.Path, "/")
	if dbName == "" {
		return nil
	}

	adminURL := *targetURL
	adminURL.Path = "/postgres"
	pool, err := pgxpool.New(ctx, adminURL.String())
	if err != nil {
		return err
	}
	defer pool.Close()

	var exists bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err = pool.Exec(ctx, "CREATE DATABASE "+quoteIdent(dbName))
	return err
}

func quoteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
