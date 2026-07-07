package handler

import (
	"context"

	"image-ai/backend/internal/model"
)

func (h *Handler) CurrentUser(ctx context.Context, userID string) (model.User, error) {
	return h.repo.UserByID(ctx, userID)
}
