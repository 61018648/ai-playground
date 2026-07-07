package repository

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"image-ai/backend/internal/model"
)

var ErrNotFound = errors.New("not found")
var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrAlreadyUsed = errors.New("already used")
var ErrExpired = errors.New("expired")

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func randomUpperCode(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, length)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func insertAffiliateProfileTx(ctx context.Context, tx pgx.Tx, userID string) error {
	for i := 0; i < 5; i++ {
		code, err := randomUpperCode(8)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO affiliate_profiles (user_id, code)
			VALUES ($1::uuid, $2)
			ON CONFLICT (user_id) DO NOTHING
		`, userID, code)
		if isUniqueViolation(err) {
			continue
		}
		return err
	}
	return errors.New("affiliate code collision")
}

func (r *Repository) CreateVerificationCode(ctx context.Context, email, purpose, codeHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO verification_codes (email, purpose, code_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, email, purpose, codeHash, expiresAt)
	return err
}

func (r *Repository) ConsumeVerificationCode(ctx context.Context, email, purpose, codeHash string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE verification_codes
		SET consumed_at = now()
		WHERE id = (
			SELECT id FROM verification_codes
			WHERE email = $1
			  AND purpose = $2
			  AND code_hash = $3
			  AND consumed_at IS NULL
			  AND expires_at > now()
			ORDER BY created_at DESC
			LIMIT 1
		)
	`, email, purpose, codeHash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash, nickname string) (model.User, error) {
	var user model.User
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return user, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, nickname)
		VALUES ($1, $2, $3)
		RETURNING id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
	`, email, passwordHash, nickname).Scan(
		&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt,
	)
	if err != nil {
		return user, err
	}
	if err := insertAffiliateProfileTx(ctx, tx, user.ID); err != nil {
		return user, err
	}
	if err := tx.Commit(ctx); err != nil {
		return user, err
	}
	return user, nil
}

func (r *Repository) UserByEmail(ctx context.Context, email string) (model.User, string, error) {
	var user model.User
	var passwordHash string
	err := r.db.QueryRow(ctx, `
		SELECT id::text, email, password_hash, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`, email).Scan(
		&user.ID, &user.Email, &passwordHash, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return user, "", ErrNotFound
	}
	return user, passwordHash, err
}

func (r *Repository) UserByID(ctx context.Context, id string) (model.User, error) {
	var user model.User
	err := r.db.QueryRow(ctx, `
		SELECT id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user, ErrNotFound
	}
	return user, err
}

func (r *Repository) UpdateUserProfile(ctx context.Context, id, nickname, avatarURL, signature string) (model.User, error) {
	var user model.User
	err := r.db.QueryRow(ctx, `
		UPDATE users
		SET nickname = COALESCE(NULLIF($2, ''), nickname),
		    avatar_url = $3,
		    signature = $4,
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
	`, id, nickname, avatarURL, signature).Scan(
		&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return user, ErrNotFound
	}
	return user, err
}

func (r *Repository) UpdatePassword(ctx context.Context, email, passwordHash string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, updated_at = now()
		WHERE email = $1 AND deleted_at IS NULL
	`, email, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) UserPasswordHashByID(ctx context.Context, id string) (model.User, string, error) {
	var user model.User
	var passwordHash string
	err := r.db.QueryRow(ctx, `
		SELECT id::text, email, password_hash, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&user.ID, &user.Email, &passwordHash, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return user, "", ErrNotFound
	}
	return user, passwordHash, err
}

func (r *Repository) UpdatePasswordByID(ctx context.Context, id, passwordHash string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) UpdateUserEmail(ctx context.Context, id, email string) (model.User, error) {
	var user model.User
	err := r.db.QueryRow(ctx, `
		UPDATE users
		SET email = $2,
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
	`, id, email).Scan(
		&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return user, ErrNotFound
	}
	return user, err
}

func (r *Repository) ListApps(ctx context.Context) ([]model.App, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id::text, COALESCE(a.provider_id::text, ''), COALESCE(p.name, ''), a.code, a.name, a.app_type, a.category, a.description, a.icon, a.icon_color, a.cover_url,
		       prompt_template, input_schema, output_schema, price_free::text, price_v1::text, price_v2::text,
		       a.visibility, a.status, a.sort_order, a.created_at, a.updated_at
		FROM apps a
		LEFT JOIN api_providers p ON p.id = a.provider_id
		WHERE a.status = 'active' AND a.visibility = 'public'
		ORDER BY a.sort_order ASC, a.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []model.App
	for rows.Next() {
		var app model.App
		var providerID string
		if err := rows.Scan(
			&app.ID, &providerID, &app.ProviderName, &app.Code, &app.Name, &app.AppType, &app.Category, &app.Description, &app.Icon, &app.IconColor, &app.CoverURL,
			&app.PromptTemplate, &app.InputSchema, &app.OutputSchema, &app.PriceFree, &app.PriceV1, &app.PriceV2,
			&app.Visibility, &app.Status, &app.SortOrder,
			&app.CreatedAt, &app.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if providerID != "" {
			app.ProviderID = &providerID
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func (r *Repository) AppByID(ctx context.Context, id string) (model.App, error) {
	var app model.App
	var providerID string
	err := r.db.QueryRow(ctx, `
		SELECT a.id::text, COALESCE(a.provider_id::text, ''), COALESCE(p.name, ''), a.code, a.name, a.app_type, a.category, a.description, a.icon, a.icon_color, a.cover_url,
		       prompt_template, input_schema, output_schema, price_free::text, price_v1::text, price_v2::text,
		       a.visibility, a.status, a.sort_order, a.created_at, a.updated_at
		FROM apps a
		LEFT JOIN api_providers p ON p.id = a.provider_id
		WHERE a.id = $1 AND a.status = 'active'
	`, id).Scan(
		&app.ID, &providerID, &app.ProviderName, &app.Code, &app.Name, &app.AppType, &app.Category, &app.Description, &app.Icon, &app.IconColor, &app.CoverURL,
		&app.PromptTemplate, &app.InputSchema, &app.OutputSchema, &app.PriceFree, &app.PriceV1, &app.PriceV2,
		&app.Visibility, &app.Status, &app.SortOrder,
		&app.CreatedAt, &app.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return app, ErrNotFound
	}
	if providerID != "" {
		app.ProviderID = &providerID
	}
	return app, err
}

func (r *Repository) AppByCode(ctx context.Context, code string) (model.App, error) {
	var app model.App
	var providerID string
	err := r.db.QueryRow(ctx, `
		SELECT a.id::text, COALESCE(a.provider_id::text, ''), COALESCE(p.name, ''), a.code, a.name, a.app_type, a.category, a.description, a.icon, a.icon_color, a.cover_url,
		       prompt_template, input_schema, output_schema, price_free::text, price_v1::text, price_v2::text,
		       a.visibility, a.status, a.sort_order, a.created_at, a.updated_at
		FROM apps a
		LEFT JOIN api_providers p ON p.id = a.provider_id
		WHERE a.code = $1 AND a.status = 'active'
	`, code).Scan(
		&app.ID, &providerID, &app.ProviderName, &app.Code, &app.Name, &app.AppType, &app.Category, &app.Description, &app.Icon, &app.IconColor, &app.CoverURL,
		&app.PromptTemplate, &app.InputSchema, &app.OutputSchema, &app.PriceFree, &app.PriceV1, &app.PriceV2,
		&app.Visibility, &app.Status, &app.SortOrder, &app.CreatedAt, &app.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return app, ErrNotFound
	}
	if providerID != "" {
		app.ProviderID = &providerID
	}
	return app, err
}

func (r *Repository) CreateGeneration(ctx context.Context, userID string, appID *string, prompt, negativePrompt string, params json.RawMessage, modelName string) (model.GenerationJob, error) {
	if len(params) == 0 {
		params = json.RawMessage(`{}`)
	}
	var job model.GenerationJob
	err := r.db.QueryRow(ctx, `
		INSERT INTO generation_jobs (user_id, app_id, prompt, negative_prompt, params, model, status, progress)
		VALUES ($1, $2, $3, $4, $5, $6, 'queued', 0)
		RETURNING id::text, user_id::text, app_id::text, prompt, negative_prompt, params, model, status, progress,
		          error_message, seed, created_at, started_at, finished_at
	`, userID, appID, prompt, negativePrompt, params, modelName).Scan(
		&job.ID, &job.UserID, &job.AppID, &job.Prompt, &job.NegativePrompt, &job.Params, &job.Model, &job.Status, &job.Progress,
		&job.ErrorMessage, &job.Seed, &job.CreatedAt, &job.StartedAt, &job.FinishedAt,
	)
	return job, err
}

func (r *Repository) EnabledAPIProviderByCategory(ctx context.Context, category string) (model.APIProvider, error) {
	var provider model.APIProvider
	err := r.db.QueryRow(ctx, `
		SELECT id::text, name, category, provider, base_url, api_key, model, enabled, sort_order, created_at, updated_at
		FROM api_providers
		WHERE enabled = true AND category = $1
		ORDER BY sort_order ASC, created_at DESC
		LIMIT 1
	`, category).Scan(
		&provider.ID, &provider.Name, &provider.Category, &provider.Provider, &provider.BaseURL, &provider.APIKey, &provider.Model,
		&provider.Enabled, &provider.SortOrder, &provider.CreatedAt, &provider.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return provider, ErrNotFound
	}
	return provider, err
}

func (r *Repository) CompleteGenerationPlaceholder(ctx context.Context, jobID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var prompt string
	err = tx.QueryRow(ctx, `
		UPDATE generation_jobs
		SET status = 'succeeded', progress = 100, started_at = COALESCE(started_at, now()), finished_at = now()
		WHERE id = $1
		RETURNING prompt
	`, jobID).Scan(&prompt)
	if err != nil {
		return err
	}

	assetURL := "https://placehold.co/1024x1024/png?text=Image+AI"
	_, err = tx.Exec(ctx, `
		INSERT INTO generation_assets (job_id, kind, url, thumbnail_url, width, height, mime_type, sort_order, meta)
		VALUES ($1, 'image', $2, $2, 1024, 1024, 'image/png', 0, jsonb_build_object('provider', 'placeholder', 'prompt', $3::text))
	`, jobID, assetURL, prompt)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) completeGenerationPlaceholderTx(ctx context.Context, tx pgx.Tx, jobID string) error {
	var prompt string
	err := tx.QueryRow(ctx, `
		UPDATE generation_jobs
		SET status = 'succeeded', progress = 100, started_at = COALESCE(started_at, now()), finished_at = now()
		WHERE id = $1
		RETURNING prompt
	`, jobID).Scan(&prompt)
	if err != nil {
		return err
	}

	assetURL := "https://placehold.co/1024x1024/png?text=Image+AI"
	_, err = tx.Exec(ctx, `
		INSERT INTO generation_assets (job_id, kind, url, thumbnail_url, width, height, mime_type, sort_order, meta)
		VALUES ($1, 'image', $2, $2, 1024, 1024, 'image/png', 0, jsonb_build_object('provider', 'placeholder', 'prompt', $3::text))
		ON CONFLICT DO NOTHING
	`, jobID, assetURL, prompt)
	return err
}

func (r *Repository) ListConversations(ctx context.Context, userID string, limit int) ([]model.Conversation, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := r.db.Query(ctx, `
		SELECT c.id::text, c.user_id::text, COALESCE(c.app_id::text, ''), COALESCE(a.name, ''), c.kind, c.title, c.created_at, c.updated_at
		FROM conversations c
		LEFT JOIN apps a ON a.id = c.app_id
		WHERE c.user_id = $1
		ORDER BY c.updated_at DESC, c.created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []model.Conversation
	for rows.Next() {
		var conversation model.Conversation
		var appID string
		if err := rows.Scan(&conversation.ID, &conversation.UserID, &appID, &conversation.AppName, &conversation.Kind, &conversation.Title, &conversation.CreatedAt, &conversation.UpdatedAt); err != nil {
			return nil, err
		}
		if appID != "" {
			conversation.AppID = &appID
		}
		conversations = append(conversations, conversation)
	}
	return conversations, rows.Err()
}

func (r *Repository) ConversationByID(ctx context.Context, userID, conversationID string) (model.ConversationDetail, error) {
	var detail model.ConversationDetail
	var appID string
	err := r.db.QueryRow(ctx, `
		SELECT c.id::text, c.user_id::text, COALESCE(c.app_id::text, ''), COALESCE(a.name, ''), c.kind, c.title, c.created_at, c.updated_at
		FROM conversations c
		LEFT JOIN apps a ON a.id = c.app_id
		WHERE c.user_id = $1 AND c.id = $2
	`, userID, conversationID).Scan(
		&detail.Conversation.ID, &detail.Conversation.UserID, &appID, &detail.Conversation.AppName, &detail.Conversation.Kind,
		&detail.Conversation.Title, &detail.Conversation.CreatedAt, &detail.Conversation.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return detail, ErrNotFound
	}
	if err != nil {
		return detail, err
	}
	if appID != "" {
		detail.Conversation.AppID = &appID
	}

	rows, err := r.db.Query(ctx, `
		SELECT id::text, conversation_id::text, role, content, meta, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`, conversationID)
	if err != nil {
		return detail, err
	}
	defer rows.Close()

	for rows.Next() {
		var message model.ConversationMessage
		if err := rows.Scan(&message.ID, &message.ConversationID, &message.Role, &message.Content, &message.Meta, &message.CreatedAt); err != nil {
			return detail, err
		}
		detail.Messages = append(detail.Messages, message)
	}
	if err := rows.Err(); err != nil {
		return detail, err
	}

	var jobID string
	for _, message := range detail.Messages {
		var meta struct {
			JobID string `json:"jobId"`
		}
		if len(message.Meta) > 0 && json.Unmarshal(message.Meta, &meta) == nil && meta.JobID != "" {
			jobID = meta.JobID
		}
	}
	if jobID != "" {
		job, err := r.GenerationByID(ctx, userID, jobID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return detail, err
		}
		if err == nil {
			detail.Job = &job
		}
	}

	return detail, nil
}

func (r *Repository) ListGenerations(ctx context.Context, userID string, limit int) ([]model.GenerationJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := r.db.Query(ctx, `
		SELECT j.id::text, j.user_id::text, j.app_id::text, COALESCE(a.name, ''), j.prompt, j.negative_prompt,
		       j.params, j.model, j.status, j.progress, j.error_message, j.seed, j.created_at, j.started_at, j.finished_at
		FROM generation_jobs j
		LEFT JOIN apps a ON a.id = j.app_id
		WHERE j.user_id = $1
		ORDER BY j.created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.GenerationJob
	for rows.Next() {
		var job model.GenerationJob
		if err := rows.Scan(
			&job.ID, &job.UserID, &job.AppID, &job.AppName, &job.Prompt, &job.NegativePrompt, &job.Params,
			&job.Model, &job.Status, &job.Progress, &job.ErrorMessage, &job.Seed, &job.CreatedAt, &job.StartedAt, &job.FinishedAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return r.attachAssets(ctx, jobs)
}

func (r *Repository) GenerationByID(ctx context.Context, userID, jobID string) (model.GenerationJob, error) {
	var job model.GenerationJob
	err := r.db.QueryRow(ctx, `
		SELECT j.id::text, j.user_id::text, j.app_id::text, COALESCE(a.name, ''), j.prompt, j.negative_prompt,
		       j.params, j.model, j.status, j.progress, j.error_message, j.seed, j.created_at, j.started_at, j.finished_at
		FROM generation_jobs j
		LEFT JOIN apps a ON a.id = j.app_id
		WHERE j.user_id = $1 AND j.id = $2
	`, userID, jobID).Scan(
		&job.ID, &job.UserID, &job.AppID, &job.AppName, &job.Prompt, &job.NegativePrompt, &job.Params,
		&job.Model, &job.Status, &job.Progress, &job.ErrorMessage, &job.Seed, &job.CreatedAt, &job.StartedAt, &job.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return job, ErrNotFound
	}
	if err != nil {
		return job, err
	}
	jobs, err := r.attachAssets(ctx, []model.GenerationJob{job})
	if err != nil {
		return job, err
	}
	return jobs[0], nil
}

func (r *Repository) CreateChargedDrawConversation(ctx context.Context, userID string, app model.App, prompt string, params json.RawMessage, modelName, price string) (model.DrawConversationResult, error) {
	var result model.DrawConversationResult
	if len(params) == 0 {
		params = json.RawMessage(`{}`)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var user model.User
	err = tx.QueryRow(ctx, `
		UPDATE users
		SET balance = balance - NULLIF($2, '')::numeric,
		    updated_at = now()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND balance >= NULLIF($2, '')::numeric
		RETURNING id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
	`, userID, price).Scan(
		&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrInsufficientBalance
	}
	if err != nil {
		return result, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO balance_logs (user_id, change_type, amount, balance_before, balance_after, note)
		SELECT $1, 'decrease', NULLIF($2, '')::numeric, NULLIF($3, '')::numeric + NULLIF($2, '')::numeric, NULLIF($3, '')::numeric, $4
	`, userID, price, user.Balance, "专业绘画扣费"); err != nil {
		return result, err
	}

	var conversationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (user_id, app_id, kind, title)
		VALUES ($1, $2, 'draw', $3)
		RETURNING id::text
	`, userID, app.ID, prompt).Scan(&conversationID); err != nil {
		return result, err
	}

	var job model.GenerationJob
	err = tx.QueryRow(ctx, `
		INSERT INTO generation_jobs (user_id, app_id, prompt, negative_prompt, params, model, status, progress)
		VALUES ($1, $2, $3, '', $4, $5, 'queued', 0)
		RETURNING id::text, user_id::text, app_id::text, prompt, negative_prompt, params, model, status, progress,
		          error_message, seed, created_at, started_at, finished_at
	`, userID, app.ID, prompt, params, modelName).Scan(
		&job.ID, &job.UserID, &job.AppID, &job.Prompt, &job.NegativePrompt, &job.Params, &job.Model, &job.Status, &job.Progress,
		&job.ErrorMessage, &job.Seed, &job.CreatedAt, &job.StartedAt, &job.FinishedAt,
	)
	if err != nil {
		return result, err
	}
	job.AppName = app.Name

	meta, _ := json.Marshal(map[string]any{
		"jobId":   job.ID,
		"charged": price,
		"model":   modelName,
		"params":  json.RawMessage(params),
		"status":  job.Status,
	})
	userMessage, err := insertMessage(ctx, tx, conversationID, "user", prompt, json.RawMessage(meta))
	if err != nil {
		return result, err
	}
	assistantMessage, err := insertMessage(ctx, tx, conversationID, "assistant", "任务已提交，正在生成专业绘画结果。", json.RawMessage(meta))
	if err != nil {
		return result, err
	}
	if _, err := tx.Exec(ctx, `
			UPDATE conversations
			SET updated_at = now()
			WHERE id = $1
		`, conversationID); err != nil {
		return result, err
	}

	if err := tx.Commit(ctx); err != nil {
		return result, err
	}
	result.ConversationID = conversationID
	result.User = user
	result.Job = job
	result.Messages = []model.ConversationMessage{userMessage, assistantMessage}
	result.Charged = price
	return result, nil
}

func (r *Repository) CreateChargedDrawMessage(ctx context.Context, userID, conversationID string, app model.App, prompt, effectivePrompt string, params json.RawMessage, modelName, price string) (model.DrawConversationResult, error) {
	var result model.DrawConversationResult
	if len(params) == 0 {
		params = json.RawMessage(`{}`)
	}
	if strings.TrimSpace(effectivePrompt) == "" {
		effectivePrompt = prompt
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var existingID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM conversations
		WHERE id = $1 AND user_id = $2 AND kind = 'draw'
		FOR UPDATE
	`, conversationID, userID).Scan(&existingID); errors.Is(err, pgx.ErrNoRows) {
		return result, ErrNotFound
	} else if err != nil {
		return result, err
	}

	var user model.User
	err = tx.QueryRow(ctx, `
		UPDATE users
		SET balance = balance - NULLIF($2, '')::numeric,
		    updated_at = now()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND balance >= NULLIF($2, '')::numeric
		RETURNING id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
	`, userID, price).Scan(
		&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrInsufficientBalance
	}
	if err != nil {
		return result, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO balance_logs (user_id, change_type, amount, balance_before, balance_after, note)
		SELECT $1, 'decrease', NULLIF($2, '')::numeric, NULLIF($3, '')::numeric + NULLIF($2, '')::numeric, NULLIF($3, '')::numeric, $4
	`, userID, price, user.Balance, "专业绘画扣费"); err != nil {
		return result, err
	}

	var job model.GenerationJob
	err = tx.QueryRow(ctx, `
		INSERT INTO generation_jobs (user_id, app_id, prompt, negative_prompt, params, model, status, progress)
		VALUES ($1, $2, $3, '', $4, $5, 'queued', 0)
		RETURNING id::text, user_id::text, app_id::text, prompt, negative_prompt, params, model, status, progress,
		          error_message, seed, created_at, started_at, finished_at
	`, userID, app.ID, effectivePrompt, params, modelName).Scan(
		&job.ID, &job.UserID, &job.AppID, &job.Prompt, &job.NegativePrompt, &job.Params, &job.Model, &job.Status, &job.Progress,
		&job.ErrorMessage, &job.Seed, &job.CreatedAt, &job.StartedAt, &job.FinishedAt,
	)
	if err != nil {
		return result, err
	}
	job.AppName = app.Name

	meta, _ := json.Marshal(map[string]any{
		"jobId":           job.ID,
		"charged":         price,
		"model":           modelName,
		"params":          json.RawMessage(params),
		"status":          job.Status,
		"effectivePrompt": effectivePrompt,
	})
	userMessage, err := insertMessage(ctx, tx, conversationID, "user", prompt, json.RawMessage(meta))
	if err != nil {
		return result, err
	}
	assistantMessage, err := insertMessage(ctx, tx, conversationID, "assistant", "任务已提交，正在生成专业绘画结果。", json.RawMessage(meta))
	if err != nil {
		return result, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversations
		SET updated_at = now()
		WHERE id = $1
	`, conversationID); err != nil {
		return result, err
	}

	if err := tx.Commit(ctx); err != nil {
		return result, err
	}
	result.ConversationID = conversationID
	result.User = user
	result.Job = job
	result.Messages = []model.ConversationMessage{userMessage, assistantMessage}
	result.Charged = price
	return result, nil
}

func (r *Repository) MarkProfessionalDrawRunning(ctx context.Context, jobID, conversationID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE generation_jobs
		SET status = 'running', progress = 35, started_at = COALESCE(started_at, now())
		WHERE id = $1 AND status = 'queued'
	`, jobID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE messages
		SET meta = jsonb_set(meta, '{status}', '"running"', true)
		WHERE conversation_id = $1 AND role = 'assistant' AND meta->>'jobId' = $2
	`, conversationID, jobID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversations
		SET updated_at = now()
		WHERE id = $1
	`, conversationID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) CompleteProfessionalDrawJob(ctx context.Context, jobID, conversationID, imageURL, modelName string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE generation_jobs
		SET status = 'succeeded', progress = 100, started_at = COALESCE(started_at, now()), finished_at = now()
		WHERE id = $1
	`, jobID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO generation_assets (job_id, kind, url, thumbnail_url, width, height, mime_type, sort_order, meta)
		VALUES ($1, 'image', $2, $2, 1024, 1024, 'image/png', 0, jsonb_build_object('provider', 'openai', 'model', $3::text))
	`, jobID, toDataURL(imageURL), modelName); err != nil {
		return err
	}
	meta, _ := json.Marshal(map[string]any{
		"jobId":    jobID,
		"model":    modelName,
		"assetUrl": toDataURL(imageURL),
		"status":   "succeeded",
	})
	if _, err := tx.Exec(ctx, `
		UPDATE messages
		SET content = '已完成生成，图片结果如下。', meta = meta || $3::jsonb
		WHERE conversation_id = $1 AND role = 'assistant' AND meta->>'jobId' = $2
	`, conversationID, jobID, json.RawMessage(meta)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversations
		SET updated_at = now()
		WHERE id = $1
	`, conversationID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func toDataURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if strings.HasPrefix(value, "data:") || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return fmt.Sprintf("data:image/png;base64,%s", value)
}

func (r *Repository) FailProfessionalDrawJob(ctx context.Context, jobID, conversationID, message string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE generation_jobs
		SET status = 'failed', progress = 100, error_message = $2, started_at = COALESCE(started_at, now()), finished_at = now()
		WHERE id = $1
	`, jobID, message); err != nil {
		return err
	}
	meta, _ := json.Marshal(map[string]any{
		"jobId":  jobID,
		"status": "failed",
		"error":  message,
	})
	if _, err := tx.Exec(ctx, `
		UPDATE messages
		SET content = $3, meta = meta || $4::jsonb
		WHERE conversation_id = $1 AND role = 'assistant' AND meta->>'jobId' = $2
	`, conversationID, jobID, "生成失败："+message, json.RawMessage(meta)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversations
		SET updated_at = now()
		WHERE id = $1
	`, conversationID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type messageInserter interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func insertMessage(ctx context.Context, tx messageInserter, conversationID, role, content string, meta json.RawMessage) (model.ConversationMessage, error) {
	var message model.ConversationMessage
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, role, content, meta)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, conversation_id::text, role, content, meta, created_at
	`, conversationID, role, content, meta).Scan(
		&message.ID, &message.ConversationID, &message.Role, &message.Content, &message.Meta, &message.CreatedAt,
	)
	return message, err
}

func (r *Repository) AddAssistantChatMessage(ctx context.Context, userID, conversationID, prompt, answer, modelName string, attachments json.RawMessage) (model.AssistantChatResult, error) {
	var result model.AssistantChatResult
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if strings.TrimSpace(conversationID) == "" {
		title := prompt
		if len([]rune(title)) > 36 {
			title = string([]rune(title)[:36])
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO conversations (user_id, kind, title)
			VALUES ($1, 'assistant', $2)
			RETURNING id::text, user_id::text, kind, title, created_at, updated_at
		`, userID, title).Scan(
			&result.Conversation.ID, &result.Conversation.UserID, &result.Conversation.Kind,
			&result.Conversation.Title, &result.Conversation.CreatedAt, &result.Conversation.UpdatedAt,
		)
		if err != nil {
			return result, err
		}
		conversationID = result.Conversation.ID
	} else {
		err = tx.QueryRow(ctx, `
			SELECT id::text, user_id::text, kind, title, created_at, updated_at
			FROM conversations
			WHERE id = $1 AND user_id = $2 AND kind = 'assistant'
			FOR UPDATE
		`, conversationID, userID).Scan(
			&result.Conversation.ID, &result.Conversation.UserID, &result.Conversation.Kind,
			&result.Conversation.Title, &result.Conversation.CreatedAt, &result.Conversation.UpdatedAt,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return result, ErrNotFound
		}
		if err != nil {
			return result, err
		}
	}

	if len(attachments) == 0 {
		attachments = json.RawMessage(`[]`)
	}
	userMeta, _ := json.Marshal(map[string]any{
		"model":       modelName,
		"kind":        "assistant",
		"attachments": json.RawMessage(attachments),
	})
	assistantMeta, _ := json.Marshal(map[string]any{
		"model": modelName,
		"kind":  "assistant",
	})
	userMessage, err := insertMessage(ctx, tx, conversationID, "user", prompt, json.RawMessage(userMeta))
	if err != nil {
		return result, err
	}
	assistantMessage, err := insertMessage(ctx, tx, conversationID, "assistant", answer, json.RawMessage(assistantMeta))
	if err != nil {
		return result, err
	}
	err = tx.QueryRow(ctx, `
		UPDATE conversations
		SET updated_at = now()
		WHERE id = $1
		RETURNING id::text, user_id::text, kind, title, created_at, updated_at
	`, conversationID).Scan(
		&result.Conversation.ID, &result.Conversation.UserID, &result.Conversation.Kind,
		&result.Conversation.Title, &result.Conversation.CreatedAt, &result.Conversation.UpdatedAt,
	)
	if err != nil {
		return result, err
	}
	if err := tx.Commit(ctx); err != nil {
		return result, err
	}
	result.Messages = []model.ConversationMessage{userMessage, assistantMessage}
	return result, nil
}

func (r *Repository) ClearDrawConversations(ctx context.Context, userID string) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM conversations
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) attachAssets(ctx context.Context, jobs []model.GenerationJob) ([]model.GenerationJob, error) {
	if len(jobs) == 0 {
		return jobs, nil
	}
	ids := make([]string, 0, len(jobs))
	index := map[string]int{}
	for i, job := range jobs {
		ids = append(ids, job.ID)
		index[job.ID] = i
	}

	rows, err := r.db.Query(ctx, `
		SELECT id::text, job_id::text, kind, url, thumbnail_url, width, height, mime_type, sort_order, meta, created_at
		FROM generation_assets
		WHERE job_id::text = ANY($1)
		ORDER BY sort_order ASC, created_at ASC
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var asset model.Asset
		if err := rows.Scan(
			&asset.ID, &asset.JobID, &asset.Kind, &asset.URL, &asset.ThumbnailURL, &asset.Width, &asset.Height,
			&asset.MimeType, &asset.SortOrder, &asset.Meta, &asset.CreatedAt,
		); err != nil {
			return nil, err
		}
		if i, ok := index[asset.JobID]; ok {
			jobs[i].Assets = append(jobs[i].Assets, asset)
		}
	}
	return jobs, rows.Err()
}

func (r *Repository) ListMediaAssets(ctx context.Context, userID string, favoriteOnly bool, limit int) ([]model.MediaAsset, error) {
	if limit <= 0 || limit > 200 {
		limit = 60
	}
	rows, err := r.db.Query(ctx, `
		SELECT a.id::text, a.job_id::text, a.kind, a.url, a.thumbnail_url, a.width, a.height, a.mime_type, a.sort_order, a.meta, a.created_at,
		       j.prompt, COALESCE(apps.name, ''), j.model, j.status, (f.asset_id IS NOT NULL) AS is_favorite, j.created_at
		FROM generation_assets a
		JOIN generation_jobs j ON j.id = a.job_id
		LEFT JOIN apps ON apps.id = j.app_id
		LEFT JOIN favorites f ON f.asset_id = a.id AND f.user_id = $1
		WHERE j.user_id = $1
		  AND a.kind = 'image'
		  AND ($2 = false OR f.asset_id IS NOT NULL)
		ORDER BY a.created_at DESC
		LIMIT $3
	`, userID, favoriteOnly, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []model.MediaAsset
	for rows.Next() {
		var asset model.MediaAsset
		if err := rows.Scan(
			&asset.ID, &asset.JobID, &asset.Kind, &asset.URL, &asset.ThumbnailURL, &asset.Width, &asset.Height,
			&asset.MimeType, &asset.SortOrder, &asset.Meta, &asset.CreatedAt,
			&asset.Prompt, &asset.AppName, &asset.Model, &asset.JobStatus, &asset.IsFavorite, &asset.GeneratedAt,
		); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func (r *Repository) SetFavorite(ctx context.Context, userID, assetID string, favorite bool) error {
	var exists bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM generation_assets a
			JOIN generation_jobs j ON j.id = a.job_id
			WHERE a.id = $1 AND j.user_id = $2
		)
	`, assetID, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	if favorite {
		_, err := r.db.Exec(ctx, `
			INSERT INTO favorites (user_id, asset_id)
			VALUES ($1, $2)
			ON CONFLICT (user_id, asset_id) DO NOTHING
		`, userID, assetID)
		return err
	}
	_, err := r.db.Exec(ctx, `
		DELETE FROM favorites
		WHERE user_id = $1 AND asset_id = $2
	`, userID, assetID)
	return err
}

func (r *Repository) AdminStats(ctx context.Context) (model.AdminStats, error) {
	var stats model.AdminStats
	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM users WHERE deleted_at IS NULL),
			(SELECT count(*) FROM apps),
			(SELECT count(*) FROM generation_jobs),
			(SELECT count(*) FROM generation_assets),
			(SELECT count(*) FROM generation_jobs WHERE created_at >= date_trunc('day', now()))
	`).Scan(
		&stats.UsersTotal,
		&stats.AppsTotal,
		&stats.GenerationsTotal,
		&stats.AssetsTotal,
		&stats.TodayGenerations,
	)
	return stats, err
}

func (r *Repository) AdminListUsers(ctx context.Context, limit int) ([]model.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *Repository) AdminCreateUser(ctx context.Context, user model.User, passwordHash string) (model.User, error) {
	if user.Role == "" {
		user.Role = "user"
	}
	if user.Status == "" {
		user.Status = "active"
	}
	if user.MembershipLevel == "" {
		user.MembershipLevel = "free"
	}
	if user.Balance == "" {
		user.Balance = "0"
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, nickname, role, status, membership_level, balance, credits)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::numeric, $8)
		RETURNING id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
	`, user.Email, passwordHash, user.Nickname, user.Role, user.Status, user.MembershipLevel, user.Balance, user.Credits).Scan(
		&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt,
	)
	return user, err
}

func (r *Repository) AdminUpdateUser(ctx context.Context, id, role, status string) (model.User, error) {
	var user model.User
	err := r.db.QueryRow(ctx, `
		UPDATE users
		SET role = COALESCE(NULLIF($2, ''), role),
		    status = COALESCE(NULLIF($3, ''), status),
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
	`, id, role, status).Scan(&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return user, ErrNotFound
	}
	return user, err
}

func (r *Repository) AdminEditUser(ctx context.Context, user model.User, passwordHash string) (model.User, error) {
	err := r.db.QueryRow(ctx, `
		UPDATE users
		SET email = COALESCE(NULLIF($2, ''), email),
		    nickname = COALESCE(NULLIF($3, ''), nickname),
		    role = COALESCE(NULLIF($4, ''), role),
		    status = COALESCE(NULLIF($5, ''), status),
		    membership_level = COALESCE(NULLIF($6, ''), membership_level),
		    credits = COALESCE($7, credits),
		    password_hash = COALESCE(NULLIF($8, ''), password_hash),
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
	`, user.ID, user.Email, user.Nickname, user.Role, user.Status, user.MembershipLevel, user.Credits, passwordHash).Scan(
		&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return user, ErrNotFound
	}
	return user, err
}

func (r *Repository) AdminAdjustUserBalance(ctx context.Context, userID, operatorID, changeType, amount, note string) (model.BalanceLog, model.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.BalanceLog{}, model.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var before string
	err = tx.QueryRow(ctx, `
		SELECT balance::text
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, userID).Scan(&before)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.BalanceLog{}, model.User{}, ErrNotFound
	}
	if err != nil {
		return model.BalanceLog{}, model.User{}, err
	}

	operator := &operatorID
	if operatorID == "" {
		operator = nil
	}

	var log model.BalanceLog
	err = tx.QueryRow(ctx, `
		WITH updated AS (
			UPDATE users
			SET balance = CASE
					WHEN $3 = 'increase' THEN balance + NULLIF($4, '')::numeric
					WHEN $3 = 'decrease' THEN balance - NULLIF($4, '')::numeric
					WHEN $3 = 'set' THEN NULLIF($4, '')::numeric
					ELSE balance
				END,
				updated_at = now()
			WHERE id = $1 AND deleted_at IS NULL
			  AND ($3 <> 'decrease' OR balance >= NULLIF($4, '')::numeric)
			RETURNING id, balance
		)
		INSERT INTO balance_logs (user_id, operator_id, change_type, amount, balance_before, balance_after, note)
		SELECT id, $2, $3, NULLIF($4, '')::numeric, NULLIF($5, '')::numeric, balance, $6
		FROM updated
		RETURNING id::text, user_id::text, operator_id::text, change_type, amount::text, balance_before::text, balance_after::text, note, created_at
	`, userID, operator, changeType, amount, before, note).Scan(
		&log.ID, &log.UserID, &log.OperatorID, &log.ChangeType, &log.Amount, &log.BalanceBefore, &log.BalanceAfter, &log.Note, &log.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.BalanceLog{}, model.User{}, ErrInsufficientBalance
	}
	if err != nil {
		return model.BalanceLog{}, model.User{}, err
	}

	meta, _ := json.Marshal(map[string]string{
		"targetUserId":  userID,
		"changeType":    changeType,
		"amount":        amount,
		"balanceBefore": log.BalanceBefore,
		"balanceAfter":  log.BalanceAfter,
		"note":          note,
	})
	if _, err := tx.Exec(ctx, `
		INSERT INTO task_logs (user_id, action, status, message, meta)
		VALUES ($1, 'user.balance.adjust', 'succeeded', $2, $3)
	`, operator, "管理员调整用户余额", json.RawMessage(meta)); err != nil {
		return model.BalanceLog{}, model.User{}, err
	}

	var user model.User
	err = tx.QueryRow(ctx, `
		SELECT id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt)
	if err != nil {
		return model.BalanceLog{}, model.User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.BalanceLog{}, model.User{}, err
	}
	return log, user, nil
}

func (r *Repository) AdminListApps(ctx context.Context) ([]model.App, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id::text, COALESCE(a.provider_id::text, ''), COALESCE(p.name, ''), a.code, a.name, a.app_type, a.category, a.description, a.icon, a.icon_color, a.cover_url,
		       prompt_template, input_schema, output_schema, price_free::text, price_v1::text, price_v2::text,
		       a.visibility, a.status, a.sort_order, a.created_at, a.updated_at
		FROM apps a
		LEFT JOIN api_providers p ON p.id = a.provider_id
		ORDER BY a.sort_order ASC, a.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []model.App
	for rows.Next() {
		var app model.App
		var providerID string
		if err := rows.Scan(
			&app.ID, &providerID, &app.ProviderName, &app.Code, &app.Name, &app.AppType, &app.Category, &app.Description, &app.Icon, &app.IconColor, &app.CoverURL,
			&app.PromptTemplate, &app.InputSchema, &app.OutputSchema, &app.PriceFree, &app.PriceV1, &app.PriceV2,
			&app.Visibility, &app.Status, &app.SortOrder,
			&app.CreatedAt, &app.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if providerID != "" {
			app.ProviderID = &providerID
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func (r *Repository) AdminUpsertApp(ctx context.Context, app model.App) (model.App, error) {
	if len(app.InputSchema) == 0 {
		app.InputSchema = json.RawMessage(`{}`)
	}
	if len(app.OutputSchema) == 0 {
		app.OutputSchema = json.RawMessage(`{}`)
	}
	if app.Visibility == "" {
		app.Visibility = "public"
	}
	if app.Status == "" {
		app.Status = "active"
	}
	if app.AppType == "" {
		app.AppType = "image"
	}
	if app.PriceFree == "" {
		app.PriceFree = "0"
	}
	if app.PriceV1 == "" {
		app.PriceV1 = "0"
	}
	if app.PriceV2 == "" {
		app.PriceV2 = "0"
	}
	if app.ID == "" {
		var providerID string
		err := r.db.QueryRow(ctx, `
			INSERT INTO apps (
				provider_id, code, name, app_type, category, description, icon, icon_color, cover_url,
				prompt_template, input_schema, output_schema, price_free, price_v1, price_v2, visibility, status, sort_order
			)
			VALUES (NULLIF($1, '')::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NULLIF($13, '')::numeric, NULLIF($14, '')::numeric, NULLIF($15, '')::numeric, $16, $17, $18)
			RETURNING id::text, COALESCE(provider_id::text, ''), code, name, app_type, category, description, icon, icon_color, cover_url,
			          prompt_template, input_schema, output_schema, price_free::text, price_v1::text, price_v2::text,
			          visibility, status, sort_order, created_at, updated_at
		`, appProviderID(app), app.Code, app.Name, app.AppType, app.Category, app.Description, app.Icon, app.IconColor, app.CoverURL,
			app.PromptTemplate, app.InputSchema, app.OutputSchema, app.PriceFree, app.PriceV1, app.PriceV2,
			app.Visibility, app.Status, app.SortOrder).Scan(
			&app.ID, &providerID, &app.Code, &app.Name, &app.AppType, &app.Category, &app.Description, &app.Icon, &app.IconColor, &app.CoverURL,
			&app.PromptTemplate, &app.InputSchema, &app.OutputSchema, &app.PriceFree, &app.PriceV1, &app.PriceV2,
			&app.Visibility, &app.Status, &app.SortOrder,
			&app.CreatedAt, &app.UpdatedAt,
		)
		if providerID != "" {
			app.ProviderID = &providerID
		}
		return app, err
	}

	var providerID string
	err := r.db.QueryRow(ctx, `
		UPDATE apps
		SET provider_id = NULLIF($2, '')::uuid,
		    code = $3,
		    name = $4,
		    app_type = $5,
		    category = $6,
		    description = $7,
		    icon = $8,
		    icon_color = $9,
		    cover_url = $10,
		    prompt_template = $11,
		    input_schema = $12,
		    output_schema = $13,
		    price_free = NULLIF($14, '')::numeric,
		    price_v1 = NULLIF($15, '')::numeric,
		    price_v2 = NULLIF($16, '')::numeric,
		    visibility = $17,
		    status = $18,
		    sort_order = $19,
		    updated_at = now()
		WHERE id = $1
		RETURNING id::text, COALESCE(provider_id::text, ''), code, name, app_type, category, description, icon, icon_color, cover_url,
		          prompt_template, input_schema, output_schema, price_free::text, price_v1::text, price_v2::text,
		          visibility, status, sort_order, created_at, updated_at
	`, app.ID, appProviderID(app), app.Code, app.Name, app.AppType, app.Category, app.Description, app.Icon, app.IconColor, app.CoverURL,
		app.PromptTemplate, app.InputSchema, app.OutputSchema, app.PriceFree, app.PriceV1, app.PriceV2,
		app.Visibility, app.Status, app.SortOrder).Scan(
		&app.ID, &providerID, &app.Code, &app.Name, &app.AppType, &app.Category, &app.Description, &app.Icon, &app.IconColor, &app.CoverURL,
		&app.PromptTemplate, &app.InputSchema, &app.OutputSchema, &app.PriceFree, &app.PriceV1, &app.PriceV2,
		&app.Visibility, &app.Status, &app.SortOrder,
		&app.CreatedAt, &app.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return app, ErrNotFound
	}
	if providerID != "" {
		app.ProviderID = &providerID
	}
	return app, err
}

func appProviderID(app model.App) string {
	if app.ProviderID == nil {
		return ""
	}
	return *app.ProviderID
}

func (r *Repository) AdminUpdateApp(ctx context.Context, id, status, visibility string, sortOrder *int) (model.App, error) {
	var app model.App
	var providerID string
	err := r.db.QueryRow(ctx, `
		UPDATE apps
		SET status = COALESCE(NULLIF($2, ''), status),
		    visibility = COALESCE(NULLIF($3, ''), visibility),
		    sort_order = COALESCE($4, sort_order),
		    updated_at = now()
		WHERE id = $1
		RETURNING id::text, COALESCE(provider_id::text, ''), code, name, app_type, category, description, icon, icon_color, cover_url,
		          prompt_template, input_schema, output_schema, price_free::text, price_v1::text, price_v2::text,
		          visibility, status, sort_order, created_at, updated_at
	`, id, status, visibility, sortOrder).Scan(
		&app.ID, &providerID, &app.Code, &app.Name, &app.AppType, &app.Category, &app.Description, &app.Icon, &app.IconColor, &app.CoverURL,
		&app.PromptTemplate, &app.InputSchema, &app.OutputSchema, &app.PriceFree, &app.PriceV1, &app.PriceV2,
		&app.Visibility, &app.Status, &app.SortOrder,
		&app.CreatedAt, &app.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return app, ErrNotFound
	}
	if providerID != "" {
		app.ProviderID = &providerID
	}
	return app, err
}

func (r *Repository) AdminListGenerations(ctx context.Context, limit int) ([]model.AdminGenerationJob, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT j.id::text, j.user_id::text, u.email, j.app_id::text, COALESCE(a.name, ''), j.prompt, j.negative_prompt,
		       j.params, j.model, j.status, j.progress, j.error_message, j.seed, j.created_at, j.started_at, j.finished_at
		FROM generation_jobs j
		JOIN users u ON u.id = j.user_id
		LEFT JOIN apps a ON a.id = j.app_id
		ORDER BY j.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.AdminGenerationJob
	for rows.Next() {
		var job model.AdminGenerationJob
		if err := rows.Scan(
			&job.ID, &job.UserID, &job.UserEmail, &job.AppID, &job.AppName, &job.Prompt, &job.NegativePrompt,
			&job.Params, &job.Model, &job.Status, &job.Progress, &job.ErrorMessage, &job.Seed,
			&job.CreatedAt, &job.StartedAt, &job.FinishedAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	baseJobs := make([]model.GenerationJob, len(jobs))
	for i := range jobs {
		baseJobs[i] = jobs[i].GenerationJob
	}
	baseJobs, err = r.attachAssets(ctx, baseJobs)
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		jobs[i].GenerationJob = baseJobs[i]
	}
	return jobs, nil
}

func (r *Repository) AdminListSettings(ctx context.Context) ([]model.SiteSetting, error) {
	rows, err := r.db.Query(ctx, `
		SELECT key, value, updated_at
		FROM site_settings
		ORDER BY key ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []model.SiteSetting
	for rows.Next() {
		var setting model.SiteSetting
		if err := rows.Scan(&setting.Key, &setting.Value, &setting.UpdatedAt); err != nil {
			return nil, err
		}
		settings = append(settings, setting)
	}
	return settings, rows.Err()
}

func (r *Repository) SiteSetting(ctx context.Context, key string) (model.SiteSetting, error) {
	var setting model.SiteSetting
	err := r.db.QueryRow(ctx, `
		SELECT key, value, updated_at
		FROM site_settings
		WHERE key = $1
	`, key).Scan(&setting.Key, &setting.Value, &setting.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return setting, ErrNotFound
	}
	return setting, err
}

func (r *Repository) AdminUpdateSetting(ctx context.Context, key string, value json.RawMessage) (model.SiteSetting, error) {
	var setting model.SiteSetting
	err := r.db.QueryRow(ctx, `
		INSERT INTO site_settings (key, value, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
		RETURNING key, value, updated_at
	`, key, value).Scan(&setting.Key, &setting.Value, &setting.UpdatedAt)
	return setting, err
}

func (r *Repository) SiteSettingByKey(ctx context.Context, key string) (model.SiteSetting, error) {
	var setting model.SiteSetting
	err := r.db.QueryRow(ctx, `
		SELECT key, value, updated_at
		FROM site_settings
		WHERE key = $1
	`, key).Scan(&setting.Key, &setting.Value, &setting.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return setting, ErrNotFound
	}
	return setting, err
}

func (r *Repository) CreatePaymentOrder(ctx context.Context, order model.PaymentOrder) (model.PaymentOrder, error) {
	if order.Provider == "" {
		order.Provider = "epay"
	}
	if order.Status == "" {
		order.Status = "pending"
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO payment_orders (
			trade_no, user_id, provider, order_type, plan_code, plan_name, amount, credits, membership_level, status, pay_url, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::numeric, $8, $9, $10, $11, now() + interval '15 minutes')
		RETURNING id::text, trade_no, user_id::text, provider, order_type, plan_code, plan_name, amount::text,
		          credits, membership_level, status, pay_url, paid_at, cancelled_at, expires_at, created_at, updated_at
	`, order.TradeNo, order.UserID, order.Provider, order.OrderType, order.PlanCode, order.PlanName, order.Amount,
		order.Credits, order.MembershipLevel, order.Status, order.PayURL).Scan(
		&order.ID, &order.TradeNo, &order.UserID, &order.Provider, &order.OrderType, &order.PlanCode, &order.PlanName, &order.Amount,
		&order.Credits, &order.MembershipLevel, &order.Status, &order.PayURL, &order.PaidAt, &order.CancelledAt, &order.ExpiresAt, &order.CreatedAt, &order.UpdatedAt,
	)
	return order, err
}

func (r *Repository) CompletePaymentOrder(ctx context.Context, tradeNo string) (model.PaymentOrder, model.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.PaymentOrder{}, model.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var order model.PaymentOrder
	err = tx.QueryRow(ctx, `
		SELECT id::text, trade_no, user_id::text, provider, order_type, plan_code, plan_name, amount::text,
		       credits, membership_level, status, pay_url, paid_at, cancelled_at, expires_at, created_at, updated_at
		FROM payment_orders
		WHERE trade_no = $1
		FOR UPDATE
	`, tradeNo).Scan(
		&order.ID, &order.TradeNo, &order.UserID, &order.Provider, &order.OrderType, &order.PlanCode, &order.PlanName, &order.Amount,
		&order.Credits, &order.MembershipLevel, &order.Status, &order.PayURL, &order.PaidAt, &order.CancelledAt, &order.ExpiresAt, &order.CreatedAt, &order.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return order, model.User{}, ErrNotFound
	}
	if err != nil {
		return order, model.User{}, err
	}

	var user model.User
	if order.Status != "paid" {
		switch order.OrderType {
		case "credits":
			var before string
			err = tx.QueryRow(ctx, `
				UPDATE users
				SET balance = balance + $2::numeric,
				    updated_at = now()
				WHERE id = $1 AND deleted_at IS NULL
				RETURNING balance::text
			`, order.UserID, order.Credits).Scan(&user.Balance)
			if err != nil {
				return order, model.User{}, err
			}
			err = tx.QueryRow(ctx, `
				SELECT (balance - $2::numeric)::text
				FROM users
				WHERE id = $1
			`, order.UserID, order.Credits).Scan(&before)
			if err != nil {
				return order, model.User{}, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO balance_logs (user_id, change_type, amount, balance_before, balance_after, note)
				VALUES ($1, 'increase', $2, NULLIF($3, '')::numeric, NULLIF($4, '')::numeric, $5)
			`, order.UserID, order.Credits, before, user.Balance, "积分充值："+order.PlanName); err != nil {
				return order, model.User{}, err
			}
		case "membership":
			if _, err := tx.Exec(ctx, `
				UPDATE users
				SET membership_level = $2, updated_at = now()
				WHERE id = $1 AND deleted_at IS NULL
			`, order.UserID, order.MembershipLevel); err != nil {
				return order, model.User{}, err
			}
		}

		err = tx.QueryRow(ctx, `
			UPDATE payment_orders
			SET status = 'paid', paid_at = now(), updated_at = now()
			WHERE trade_no = $1
			RETURNING id::text, trade_no, user_id::text, provider, order_type, plan_code, plan_name, amount::text,
			          credits, membership_level, status, pay_url, paid_at, cancelled_at, expires_at, created_at, updated_at
		`, tradeNo).Scan(
			&order.ID, &order.TradeNo, &order.UserID, &order.Provider, &order.OrderType, &order.PlanCode, &order.PlanName, &order.Amount,
			&order.Credits, &order.MembershipLevel, &order.Status, &order.PayURL, &order.PaidAt, &order.CancelledAt, &order.ExpiresAt, &order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return order, model.User{}, err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO affiliate_commissions (
				referrer_id, referred_user_id, payment_order_id, order_amount, product_type,
				status, commission_rate, commission_amount
			)
			SELECT ref.referrer_id, $1, $2, NULLIF($3, '')::numeric, $4,
			       'settled', profile.commission_rate, round(NULLIF($3, '')::numeric * profile.commission_rate / 100, 2)
			FROM affiliate_referrals ref
			JOIN affiliate_profiles profile ON profile.user_id = ref.referrer_id
			WHERE ref.referred_user_id = $1
			  AND ref.referrer_id <> $1
			ON CONFLICT (payment_order_id) DO NOTHING
		`, order.UserID, order.ID, order.Amount, order.OrderType); err != nil {
			return order, model.User{}, err
		}
	}

	err = tx.QueryRow(ctx, `
		SELECT id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, order.UserID).Scan(&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt)
	if err != nil {
		return order, model.User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return order, model.User{}, err
	}
	return order, user, nil
}

func (r *Repository) EnsureAffiliateProfile(ctx context.Context, userID string) (model.AffiliateProfile, error) {
	var profile model.AffiliateProfile
	for i := 0; i < 5; i++ {
		code, err := randomUpperCode(8)
		if err != nil {
			return profile, err
		}
		err = r.db.QueryRow(ctx, `
		INSERT INTO affiliate_profiles (user_id, code)
		VALUES ($1::uuid, $2)
		ON CONFLICT (user_id) DO UPDATE SET updated_at = affiliate_profiles.updated_at
		RETURNING user_id::text, code, level, commission_rate::text, visits, created_at, updated_at
	`, userID, code).Scan(
			&profile.UserID, &profile.Code, &profile.Level, &profile.CommissionRate, &profile.Visits, &profile.CreatedAt, &profile.UpdatedAt,
		)
		if isUniqueViolation(err) {
			continue
		}
		return profile, err
	}
	return profile, errors.New("affiliate code collision")
}

func (r *Repository) RecordAffiliateReferral(ctx context.Context, code, referredUserID string) error {
	if code == "" {
		return nil
	}
	tag, err := r.db.Exec(ctx, `
		INSERT INTO affiliate_referrals (referrer_id, referred_user_id)
		SELECT user_id, $2
		FROM affiliate_profiles
		WHERE upper(code) = upper($1) AND user_id::text <> $2
		ON CONFLICT (referred_user_id) DO NOTHING
	`, code, referredUserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	_, err = r.db.Exec(ctx, `
		UPDATE affiliate_profiles
		SET updated_at = now()
		WHERE upper(code) = upper($1)
	`, code)
	return err
}

func (r *Repository) RecordAffiliateVisit(ctx context.Context, code string) error {
	if code == "" {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		UPDATE affiliate_profiles
		SET visits = visits + 1, updated_at = now()
		WHERE upper(code) = upper($1)
	`, code)
	return err
}

func (r *Repository) AffiliateDashboard(ctx context.Context, userID string) (model.AffiliateDashboard, error) {
	profile, err := r.EnsureAffiliateProfile(ctx, userID)
	if err != nil {
		return model.AffiliateDashboard{}, err
	}
	dashboard := model.AffiliateDashboard{Profile: profile}
	err = r.db.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT sum(commission_amount) FROM affiliate_commissions WHERE referrer_id = $1), 0)::text,
			(COALESCE((SELECT sum(commission_amount) FROM affiliate_commissions WHERE referrer_id = $1 AND status = 'settled'), 0)
			  - COALESCE((SELECT sum(amount) FROM affiliate_withdrawals WHERE user_id = $1 AND status IN ('pending', 'paid')), 0))::text,
			COALESCE((SELECT sum(amount) FROM affiliate_withdrawals WHERE user_id = $1 AND status = 'pending'), 0)::text,
			(SELECT count(*) FROM affiliate_commissions WHERE referrer_id = $1)::int,
			(SELECT count(*) FROM affiliate_referrals WHERE referrer_id = $1)::int
	`, userID).Scan(
		&dashboard.TotalCommission, &dashboard.AvailableAmount, &dashboard.WithdrawingAmount,
		&dashboard.PaidOrderCount, &dashboard.InvitedUserCount,
	)
	if err != nil {
		return dashboard, err
	}
	dashboard.Commissions, err = r.ListAffiliateCommissions(ctx, userID, 100)
	if err != nil {
		return dashboard, err
	}
	dashboard.Withdrawals, err = r.ListAffiliateWithdrawals(ctx, userID, 100)
	if err != nil {
		return dashboard, err
	}
	dashboard.InviteUsers, err = r.ListAffiliateInviteUsers(ctx, userID, 100)
	if err != nil {
		return dashboard, err
	}
	return dashboard, nil
}

func (r *Repository) ListAffiliateCommissions(ctx context.Context, userID string, limit int) ([]model.AffiliateCommission, error) {
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT c.id::text, c.referrer_id::text, c.referred_user_id::text, COALESCE(u.email, ''), c.payment_order_id::text,
		       c.order_amount::text, c.product_type, c.status, c.commission_rate::text, c.commission_amount::text, c.created_at
		FROM affiliate_commissions c
		LEFT JOIN users u ON u.id = c.referred_user_id
		WHERE c.referrer_id = $1
		ORDER BY c.created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commissions []model.AffiliateCommission
	for rows.Next() {
		var commission model.AffiliateCommission
		if err := rows.Scan(
			&commission.ID, &commission.ReferrerID, &commission.ReferredUserID, &commission.ReferredEmail, &commission.PaymentOrderID,
			&commission.OrderAmount, &commission.ProductType, &commission.Status, &commission.CommissionRate, &commission.CommissionAmount, &commission.CreatedAt,
		); err != nil {
			return nil, err
		}
		commissions = append(commissions, commission)
	}
	return commissions, rows.Err()
}

func (r *Repository) ListAffiliateWithdrawals(ctx context.Context, userID string, limit int) ([]model.AffiliateWithdrawal, error) {
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id::text, user_id::text, amount::text, status, note, created_at, updated_at
		FROM affiliate_withdrawals
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []model.AffiliateWithdrawal
	for rows.Next() {
		var withdrawal model.AffiliateWithdrawal
		if err := rows.Scan(&withdrawal.ID, &withdrawal.UserID, &withdrawal.Amount, &withdrawal.Status, &withdrawal.Note, &withdrawal.CreatedAt, &withdrawal.UpdatedAt); err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, rows.Err()
}

func (r *Repository) ListAffiliateInviteUsers(ctx context.Context, userID string, limit int) ([]model.AffiliateInviteUser, error) {
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT u.id::text, u.email, COALESCE(u.nickname, ''), ref.created_at
		FROM affiliate_referrals ref
		JOIN users u ON u.id = ref.referred_user_id
		WHERE ref.referrer_id = $1
		ORDER BY ref.created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.AffiliateInviteUser
	for rows.Next() {
		var user model.AffiliateInviteUser
		if err := rows.Scan(&user.ID, &user.Email, &user.Nickname, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *Repository) CreateAffiliateWithdrawal(ctx context.Context, userID, amount, note string) (model.AffiliateWithdrawal, error) {
	var withdrawal model.AffiliateWithdrawal
	err := r.db.QueryRow(ctx, `
		WITH balance AS (
			SELECT COALESCE((SELECT sum(commission_amount) FROM affiliate_commissions WHERE referrer_id = $1 AND status = 'settled'), 0)
			     - COALESCE((SELECT sum(amount) FROM affiliate_withdrawals WHERE user_id = $1 AND status IN ('pending', 'paid')), 0) AS available
		)
		INSERT INTO affiliate_withdrawals (user_id, amount, note)
		SELECT $1, NULLIF($2, '')::numeric, $3
		FROM balance
		WHERE available >= NULLIF($2, '')::numeric
		RETURNING id::text, user_id::text, amount::text, status, note, created_at, updated_at
	`, userID, amount, note).Scan(
		&withdrawal.ID, &withdrawal.UserID, &withdrawal.Amount, &withdrawal.Status, &withdrawal.Note, &withdrawal.CreatedAt, &withdrawal.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return withdrawal, ErrInsufficientBalance
	}
	return withdrawal, err
}

func (r *Repository) AdminAffiliateOverview(ctx context.Context, limit int) (model.AdminAffiliateOverview, error) {
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	var overview model.AdminAffiliateOverview
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id::text, p.code, p.level, p.commission_rate::text, p.visits, p.created_at, p.updated_at,
		       u.email, u.nickname,
		       COALESCE(sum(c.commission_amount), 0)::text,
		       (COALESCE(sum(c.commission_amount) FILTER (WHERE c.status = 'settled'), 0)
		         - COALESCE((SELECT sum(w.amount) FROM affiliate_withdrawals w WHERE w.user_id = p.user_id AND w.status IN ('pending', 'paid')), 0))::text,
		       COALESCE((SELECT sum(w.amount) FROM affiliate_withdrawals w WHERE w.user_id = p.user_id AND w.status = 'pending'), 0)::text,
		       count(c.id)::int,
		       (SELECT count(*) FROM affiliate_referrals ref WHERE ref.referrer_id = p.user_id)::int
		FROM affiliate_profiles p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN affiliate_commissions c ON c.referrer_id = p.user_id
		WHERE u.deleted_at IS NULL
		GROUP BY p.user_id, p.code, p.level, p.commission_rate, p.visits, p.created_at, p.updated_at, u.email, u.nickname
		ORDER BY p.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return overview, err
	}
	defer rows.Close()

	for rows.Next() {
		var profile model.AdminAffiliateProfile
		if err := rows.Scan(
			&profile.UserID, &profile.Code, &profile.Level, &profile.CommissionRate, &profile.Visits, &profile.CreatedAt, &profile.UpdatedAt,
			&profile.Email, &profile.Nickname, &profile.TotalCommission, &profile.AvailableAmount, &profile.WithdrawingAmount,
			&profile.PaidOrderCount, &profile.InvitedUserCount,
		); err != nil {
			return overview, err
		}
		overview.Profiles = append(overview.Profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return overview, err
	}

	overview.Commissions, err = r.adminAffiliateCommissions(ctx, limit)
	if err != nil {
		return overview, err
	}
	overview.Withdrawals, err = r.adminAffiliateWithdrawals(ctx, limit)
	if err != nil {
		return overview, err
	}
	return overview, nil
}

func (r *Repository) adminAffiliateCommissions(ctx context.Context, limit int) ([]model.AffiliateCommission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id::text, c.referrer_id::text, c.referred_user_id::text, u.email, c.payment_order_id::text,
		       c.order_amount::text, c.product_type, c.status, c.commission_rate::text, c.commission_amount::text, c.created_at
		FROM affiliate_commissions c
		LEFT JOIN users u ON u.id = c.referred_user_id
		ORDER BY c.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commissions []model.AffiliateCommission
	for rows.Next() {
		var commission model.AffiliateCommission
		if err := rows.Scan(
			&commission.ID, &commission.ReferrerID, &commission.ReferredUserID, &commission.ReferredEmail, &commission.PaymentOrderID,
			&commission.OrderAmount, &commission.ProductType, &commission.Status, &commission.CommissionRate, &commission.CommissionAmount, &commission.CreatedAt,
		); err != nil {
			return nil, err
		}
		commissions = append(commissions, commission)
	}
	return commissions, rows.Err()
}

func (r *Repository) adminAffiliateWithdrawals(ctx context.Context, limit int) ([]model.AffiliateWithdrawal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, user_id::text, amount::text, status, note, created_at, updated_at
		FROM affiliate_withdrawals
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []model.AffiliateWithdrawal
	for rows.Next() {
		var withdrawal model.AffiliateWithdrawal
		if err := rows.Scan(&withdrawal.ID, &withdrawal.UserID, &withdrawal.Amount, &withdrawal.Status, &withdrawal.Note, &withdrawal.CreatedAt, &withdrawal.UpdatedAt); err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, rows.Err()
}

func (r *Repository) AdminListPaymentOrders(ctx context.Context, limit int) ([]model.AdminPaymentOrder, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
    SELECT p.id::text, p.trade_no, p.user_id::text, p.provider, p.order_type, p.plan_code, p.plan_name, p.amount::text,
           p.credits, p.membership_level, p.status, p.pay_url, p.paid_at, p.cancelled_at, p.expires_at, p.created_at, p.updated_at,
           u.email, u.nickname
    FROM payment_orders p
    JOIN users u ON u.id = p.user_id
    ORDER BY p.created_at DESC
    LIMIT $1
  `, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.AdminPaymentOrder
	for rows.Next() {
		var order model.AdminPaymentOrder
		if err := rows.Scan(
			&order.ID, &order.TradeNo, &order.UserID, &order.Provider, &order.OrderType, &order.PlanCode, &order.PlanName, &order.Amount,
			&order.Credits, &order.MembershipLevel, &order.Status, &order.PayURL, &order.PaidAt, &order.CancelledAt, &order.ExpiresAt, &order.CreatedAt, &order.UpdatedAt,
			&order.Email, &order.Nickname,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (r *Repository) AdminCancelPaymentOrder(ctx context.Context, tradeNo string) (model.PaymentOrder, error) {
	var order model.PaymentOrder
	err := r.db.QueryRow(ctx, `
    UPDATE payment_orders
    SET status = 'cancelled', cancelled_at = now(), updated_at = now()
    WHERE trade_no = $1 AND status = 'pending'
    RETURNING id::text, trade_no, user_id::text, provider, order_type, plan_code, plan_name, amount::text,
              credits, membership_level, status, pay_url, paid_at, cancelled_at, expires_at, created_at, updated_at
  `, tradeNo).Scan(
		&order.ID, &order.TradeNo, &order.UserID, &order.Provider, &order.OrderType, &order.PlanCode, &order.PlanName, &order.Amount,
		&order.Credits, &order.MembershipLevel, &order.Status, &order.PayURL, &order.PaidAt, &order.CancelledAt, &order.ExpiresAt, &order.CreatedAt, &order.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return order, ErrNotFound
	}
	return order, err
}

func (r *Repository) AdminListAPIProviders(ctx context.Context) ([]model.APIProvider, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, name, category, provider, base_url, api_key, model, enabled, sort_order, created_at, updated_at
		FROM api_providers
		ORDER BY category ASC, sort_order ASC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []model.APIProvider
	for rows.Next() {
		var provider model.APIProvider
		if err := rows.Scan(
			&provider.ID, &provider.Name, &provider.Category, &provider.Provider, &provider.BaseURL, &provider.APIKey, &provider.Model,
			&provider.Enabled, &provider.SortOrder, &provider.CreatedAt, &provider.UpdatedAt,
		); err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return providers, rows.Err()
}

func (r *Repository) AdminAPIProviderByID(ctx context.Context, id string) (model.APIProvider, error) {
	var provider model.APIProvider
	err := r.db.QueryRow(ctx, `
		SELECT id::text, name, category, provider, base_url, api_key, model, enabled, sort_order, created_at, updated_at
		FROM api_providers
		WHERE id = $1
	`, id).Scan(
		&provider.ID, &provider.Name, &provider.Category, &provider.Provider, &provider.BaseURL, &provider.APIKey, &provider.Model,
		&provider.Enabled, &provider.SortOrder, &provider.CreatedAt, &provider.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return provider, ErrNotFound
	}
	return provider, err
}

func (r *Repository) AdminUpsertAPIProvider(ctx context.Context, provider model.APIProvider) (model.APIProvider, error) {
	if provider.Category == "" {
		provider.Category = "general"
	}
	if provider.ID == "" {
		err := r.db.QueryRow(ctx, `
			INSERT INTO api_providers (name, category, provider, base_url, api_key, model, enabled, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id::text, name, category, provider, base_url, api_key, model, enabled, sort_order, created_at, updated_at
		`, provider.Name, provider.Category, provider.Provider, provider.BaseURL, provider.APIKey, provider.Model, provider.Enabled, provider.SortOrder).Scan(
			&provider.ID, &provider.Name, &provider.Category, &provider.Provider, &provider.BaseURL, &provider.APIKey, &provider.Model,
			&provider.Enabled, &provider.SortOrder, &provider.CreatedAt, &provider.UpdatedAt,
		)
		return provider, err
	}

	err := r.db.QueryRow(ctx, `
		UPDATE api_providers
		SET name = $2,
		    category = $3,
		    provider = $4,
		    base_url = $5,
		    api_key = COALESCE(NULLIF($6, ''), api_key),
		    model = $7,
		    enabled = $8,
		    sort_order = $9,
		    updated_at = now()
		WHERE id = $1
		RETURNING id::text, name, category, provider, base_url, api_key, model, enabled, sort_order, created_at, updated_at
	`, provider.ID, provider.Name, provider.Category, provider.Provider, provider.BaseURL, provider.APIKey, provider.Model, provider.Enabled, provider.SortOrder).Scan(
		&provider.ID, &provider.Name, &provider.Category, &provider.Provider, &provider.BaseURL, &provider.APIKey, &provider.Model,
		&provider.Enabled, &provider.SortOrder, &provider.CreatedAt, &provider.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return provider, ErrNotFound
	}
	return provider, err
}

func (r *Repository) AdminDeleteAPIProvider(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM api_providers
		WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) RecordLoginLog(ctx context.Context, userID *string, email string, success bool, ip, userAgent, message string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO login_logs (user_id, email, success, ip, user_agent, message)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, email, success, ip, userAgent, message)
	return err
}

func (r *Repository) AdminListLoginLogs(ctx context.Context, limit int) ([]model.LoginLog, error) {
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id::text, user_id::text, email, success, ip, user_agent, message, created_at
		FROM login_logs
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.LoginLog
	for rows.Next() {
		var log model.LoginLog
		if err := rows.Scan(&log.ID, &log.UserID, &log.Email, &log.Success, &log.IP, &log.UserAgent, &log.Message, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *Repository) RecordTaskLog(ctx context.Context, jobID, userID *string, action, status, message string, meta json.RawMessage) error {
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO task_logs (job_id, user_id, action, status, message, meta)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, jobID, userID, action, status, message, meta)
	return err
}

func (r *Repository) AdminListTaskLogs(ctx context.Context, limit int) ([]model.TaskLog, error) {
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id::text, job_id::text, user_id::text, action, status, message, meta, created_at
		FROM task_logs
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.TaskLog
	for rows.Next() {
		var log model.TaskLog
		if err := rows.Scan(&log.ID, &log.JobID, &log.UserID, &log.Action, &log.Status, &log.Message, &log.Meta, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *Repository) ListBalanceLogs(ctx context.Context, userID string, limit int) ([]model.BalanceLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id::text, user_id::text, operator_id::text, change_type, amount::text, balance_before::text, balance_after::text, note, created_at
		FROM balance_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.BalanceLog
	for rows.Next() {
		var log model.BalanceLog
		if err := rows.Scan(&log.ID, &log.UserID, &log.OperatorID, &log.ChangeType, &log.Amount, &log.BalanceBefore, &log.BalanceAfter, &log.Note, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *Repository) AdminListInviteCodes(ctx context.Context, limit int) ([]model.InviteCode, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.db.Query(ctx, `
		SELECT c.id::text, c.code, c.amount::text, c.max_uses, c.used_count, c.note,
		       COALESCE(c.created_by::text, ''), COALESCE(c.used_by::text, ''), COALESCE(u.email, ''),
		       c.used_at, c.expires_at, c.created_at
		FROM invite_codes c
		LEFT JOIN users u ON u.id = c.used_by
		ORDER BY c.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []model.InviteCode
	for rows.Next() {
		var code model.InviteCode
		var createdBy string
		var usedBy string
		if err := rows.Scan(
			&code.ID, &code.Code, &code.Amount, &code.MaxUses, &code.UsedCount, &code.Note,
			&createdBy, &usedBy, &code.UsedByEmail, &code.UsedAt, &code.ExpiresAt, &code.CreatedAt,
		); err != nil {
			return nil, err
		}
		if createdBy != "" {
			code.CreatedBy = &createdBy
		}
		if usedBy != "" {
			code.UsedBy = &usedBy
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

func (r *Repository) AdminCreateInviteCodes(ctx context.Context, codes []model.InviteCode) ([]model.InviteCode, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created := make([]model.InviteCode, 0, len(codes))
	for _, item := range codes {
		var code model.InviteCode
		var createdBy string
		err := tx.QueryRow(ctx, `
			INSERT INTO invite_codes (code, amount, max_uses, note, created_by, expires_at)
			VALUES ($1, NULLIF($2, '')::numeric, 1, $3, $4, $5)
			RETURNING id::text, code, amount::text, max_uses, used_count, note, COALESCE(created_by::text, ''), expires_at, created_at
		`, item.Code, item.Amount, item.Note, item.CreatedBy, item.ExpiresAt).Scan(
			&code.ID, &code.Code, &code.Amount, &code.MaxUses, &code.UsedCount, &code.Note, &createdBy, &code.ExpiresAt, &code.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if createdBy != "" {
			code.CreatedBy = &createdBy
		}
		created = append(created, code)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *Repository) RedeemInviteCode(ctx context.Context, userID, rawCode string) (model.InviteCode, model.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.InviteCode{}, model.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	codeText := strings.ToUpper(strings.TrimSpace(rawCode))
	var code model.InviteCode
	var createdBy string
	var usedBy string
	err = tx.QueryRow(ctx, `
		SELECT id::text, code, amount::text, max_uses, used_count, note,
		       COALESCE(created_by::text, ''), COALESCE(used_by::text, ''), used_at, expires_at, created_at
		FROM invite_codes
		WHERE upper(code) = $1 AND amount > 0
		FOR UPDATE
	`, codeText).Scan(
		&code.ID, &code.Code, &code.Amount, &code.MaxUses, &code.UsedCount, &code.Note,
		&createdBy, &usedBy, &code.UsedAt, &code.ExpiresAt, &code.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return code, model.User{}, ErrNotFound
	}
	if err != nil {
		return code, model.User{}, err
	}
	if createdBy != "" {
		code.CreatedBy = &createdBy
	}
	if usedBy != "" {
		code.UsedBy = &usedBy
	}
	if code.UsedCount >= 1 || code.UsedBy != nil {
		return code, model.User{}, ErrAlreadyUsed
	}
	if code.ExpiresAt != nil && code.ExpiresAt.Before(time.Now()) {
		return code, model.User{}, ErrExpired
	}

	var before string
	if err := tx.QueryRow(ctx, `
		SELECT balance::text
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, userID).Scan(&before); errors.Is(err, pgx.ErrNoRows) {
		return code, model.User{}, ErrNotFound
	} else if err != nil {
		return code, model.User{}, err
	}

	var user model.User
	err = tx.QueryRow(ctx, `
		UPDATE users
		SET balance = balance + NULLIF($2, '')::numeric,
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id::text, email, nickname, avatar_url, COALESCE(signature, ''), role, status, membership_level, balance::text, credits, created_at
	`, userID, code.Amount).Scan(
		&user.ID, &user.Email, &user.Nickname, &user.AvatarURL, &user.Signature, &user.Role, &user.Status, &user.MembershipLevel, &user.Balance, &user.Credits, &user.CreatedAt,
	)
	if err != nil {
		return code, model.User{}, err
	}

	if err := tx.QueryRow(ctx, `
		UPDATE invite_codes
		SET used_count = 1,
		    used_by = $2,
		    used_at = now()
		WHERE id = $1
		RETURNING used_by::text, used_at
	`, code.ID, userID).Scan(&code.UsedBy, &code.UsedAt); err != nil {
		return code, model.User{}, err
	}
	code.UsedCount = 1
	code.UsedByEmail = user.Email

	if _, err := tx.Exec(ctx, `
		INSERT INTO balance_logs (user_id, change_type, amount, balance_before, balance_after, note)
		VALUES ($1, 'increase', NULLIF($2, '')::numeric, NULLIF($3, '')::numeric, NULLIF($4, '')::numeric, $5)
	`, userID, code.Amount, before, user.Balance, "兑换码充值："+code.Code); err != nil {
		return code, model.User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return code, model.User{}, err
	}
	return code, user, nil
}
