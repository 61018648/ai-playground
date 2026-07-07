package install

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"image-ai/backend/internal/config"
	"image-ai/backend/internal/db"
	"image-ai/backend/internal/security"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Options struct {
	DatabaseURL        string
	MigrationsDir      string
	AdminEmail         string
	AdminPassword      string
	AdminNickname      string
	SkipAdminReset     bool
	ResetAdminPassword bool
}

type Result struct {
	AdminEmail        string
	AppliedMigrations []string
}

func DefaultOptions(cfg config.Config) Options {
	return Options{
		DatabaseURL:    cfg.DatabaseURL,
		MigrationsDir:  getEnv("MIGRATIONS_DIR", "migrations"),
		AdminEmail:     getEnv("INSTALL_ADMIN_EMAIL", "admin@example.com"),
		AdminPassword:  getEnv("INSTALL_ADMIN_PASSWORD", "123456"),
		AdminNickname:  getEnv("INSTALL_ADMIN_NICKNAME", "admin"),
		SkipAdminReset: envBool("INSTALL_SKIP_ADMIN_PASSWORD_RESET", false),
	}
}

func NormalizeOptions(opts Options) (Options, error) {
	opts.DatabaseURL = strings.TrimSpace(opts.DatabaseURL)
	opts.MigrationsDir = strings.TrimSpace(opts.MigrationsDir)
	opts.AdminEmail = strings.ToLower(strings.TrimSpace(opts.AdminEmail))
	opts.AdminPassword = strings.TrimSpace(opts.AdminPassword)
	opts.AdminNickname = strings.TrimSpace(opts.AdminNickname)

	if opts.DatabaseURL == "" {
		return opts, errors.New("DATABASE_URL is required")
	}
	if opts.MigrationsDir == "" {
		opts.MigrationsDir = "migrations"
	}
	if opts.AdminEmail == "" {
		opts.AdminEmail = "admin@example.com"
	}
	if !strings.Contains(opts.AdminEmail, "@") {
		return opts, errors.New("INSTALL_ADMIN_EMAIL must be an email address")
	}
	if opts.AdminPassword == "" {
		opts.AdminPassword = "123456"
	}
	if opts.AdminNickname == "" {
		opts.AdminNickname = "admin"
	}
	opts.ResetAdminPassword = !opts.SkipAdminReset
	return opts, nil
}

func Install(ctx context.Context, opts Options) (Result, error) {
	opts, err := NormalizeOptions(opts)
	if err != nil {
		return Result{}, err
	}
	if err := EnsureDatabase(ctx, opts.DatabaseURL); err != nil {
		return Result{}, fmt.Errorf("ensure database: %w", err)
	}

	pool, err := db.Open(ctx, opts.DatabaseURL)
	if err != nil {
		return Result{}, fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	applied, err := ApplyMigrations(ctx, pool, opts.MigrationsDir)
	if err != nil {
		return Result{}, err
	}
	if err := SeedAdmin(ctx, pool, opts); err != nil {
		return Result{}, err
	}

	return Result{AdminEmail: opts.AdminEmail, AppliedMigrations: applied}, nil
}

func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no migration files found in %s", migrationsDir)
	}
	sort.Strings(files)

	applied := make([]string, 0, len(files))
	for _, file := range files {
		sql, err := os.ReadFile(file)
		if err != nil {
			return applied, fmt.Errorf("read migration %s: %w", file, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return applied, fmt.Errorf("run migration %s: %w", file, err)
		}
		applied = append(applied, file)
	}
	return applied, nil
}

func SeedAdmin(ctx context.Context, pool *pgxpool.Pool, opts Options) error {
	opts, err := NormalizeOptions(opts)
	if err != nil {
		return err
	}
	passwordHash, err := security.HashPassword(opts.AdminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, nickname, role, status, membership_level, balance, credits)
		VALUES ($1, $2, $3, 'admin', 'active', 'free', 0, 0)
		ON CONFLICT (email) DO UPDATE SET
		    password_hash = CASE WHEN $4 THEN EXCLUDED.password_hash ELSE users.password_hash END,
		    nickname = EXCLUDED.nickname,
		    role = 'admin',
		    status = 'active',
		    membership_level = COALESCE(NULLIF(users.membership_level, ''), 'free'),
		    deleted_at = NULL,
		    updated_at = now()
		RETURNING id::text
	`, opts.AdminEmail, passwordHash, opts.AdminNickname, opts.ResetAdminPassword).Scan(&userID); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO affiliate_profiles (user_id, code)
		VALUES ($1::uuid, upper(substr(replace($1::text, '-', ''), 1, 8)))
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return fmt.Errorf("seed admin affiliate profile: %w", err)
	}

	return tx.Commit(ctx)
}

func EnsureDatabase(ctx context.Context, databaseURL string) error {
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

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
