package handler

import (
	"net/http"

	"image-ai/backend/internal/httpx"
)

func (h *Handler) PublicSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.AdminListSettings(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取站点配置失败")
		return
	}
	httpx.JSON(w, http.StatusOK, settings)
}
