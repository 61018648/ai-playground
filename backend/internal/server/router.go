package server

import (
	"errors"
	"net/http"

	"image-ai/backend/internal/config"
	"image-ai/backend/internal/handler"
	"image-ai/backend/internal/httpx"
	"image-ai/backend/internal/middleware"
	"image-ai/backend/internal/repository"
)

func NewRouter(cfg config.Config, h *handler.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/v1/auth/send-code", h.SendCode)
	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("POST /api/v1/auth/forgot-password", h.ForgotPassword)

	mux.Handle("GET /api/v1/auth/me", middleware.Auth(cfg.JWTSecret)(http.HandlerFunc(h.Me)))
	mux.Handle("PUT /api/v1/auth/profile", middleware.Auth(cfg.JWTSecret)(http.HandlerFunc(h.UpdateProfile)))
	mux.Handle("PUT /api/v1/auth/password", middleware.Auth(cfg.JWTSecret)(http.HandlerFunc(h.ChangePassword)))
	mux.Handle("PUT /api/v1/auth/email", middleware.Auth(cfg.JWTSecret)(http.HandlerFunc(h.ChangeEmail)))

	mux.HandleFunc("GET /api/v1/apps", h.ListApps)
	mux.HandleFunc("GET /api/v1/apps/{id}", h.GetApp)
	mux.HandleFunc("GET /api/v1/public/settings", h.PublicSettings)
	mux.HandleFunc("POST /api/v1/pay/epay/notify", h.EPayNotify)
	mux.HandleFunc("GET /api/v1/pay/epay/return", h.EPayReturn)

	auth := middleware.Auth(cfg.JWTSecret)
	mux.Handle("GET /api/v1/pay/plans", auth(http.HandlerFunc(h.PaymentPlans)))
	mux.Handle("POST /api/v1/pay/orders", auth(http.HandlerFunc(h.CreatePaymentOrder)))
	mux.Handle("GET /api/v1/affiliate/dashboard", auth(http.HandlerFunc(h.AffiliateDashboard)))
	mux.Handle("POST /api/v1/affiliate/withdrawals", auth(http.HandlerFunc(h.CreateAffiliateWithdrawal)))
	mux.HandleFunc("POST /api/v1/affiliate/visit/{code}", h.RecordAffiliateVisit)
	mux.Handle("POST /api/v1/generations", auth(http.HandlerFunc(h.CreateGeneration)))
	mux.Handle("POST /api/v1/generations/professional-draw", auth(http.HandlerFunc(h.CreateProfessionalDraw)))
	mux.Handle("POST /api/v1/generations/professional-draw/rewrite", auth(http.HandlerFunc(h.CreateProfessionalDrawRewrite)))
	mux.Handle("GET /api/v1/generations", auth(http.HandlerFunc(h.ListGenerations)))
	mux.Handle("GET /api/v1/generations/{id}", auth(http.HandlerFunc(h.GetGeneration)))
	mux.Handle("GET /api/v1/conversations", auth(http.HandlerFunc(h.ListConversations)))
	mux.Handle("GET /api/v1/conversations/{id}", auth(http.HandlerFunc(h.GetConversation)))
	mux.Handle("POST /api/v1/conversations/clear-draw", auth(http.HandlerFunc(h.ClearDrawConversations)))
	mux.Handle("DELETE /api/v1/conversations/clear-draw", auth(http.HandlerFunc(h.ClearDrawConversations)))
	mux.Handle("POST /api/v1/assistant/chat", auth(http.HandlerFunc(h.AssistantChat)))
	mux.Handle("GET /api/v1/balance-logs", auth(http.HandlerFunc(h.ListBalanceLogs)))
	mux.Handle("POST /api/v1/redeem-codes/redeem", auth(http.HandlerFunc(h.RedeemCode)))
	mux.Handle("GET /api/v1/media", auth(http.HandlerFunc(h.ListMediaAssets)))
	mux.Handle("POST /api/v1/media/{id}/favorite", auth(http.HandlerFunc(h.FavoriteMediaAsset)))
	mux.Handle("DELETE /api/v1/media/{id}/favorite", auth(http.HandlerFunc(h.UnfavoriteMediaAsset)))

	admin := func(next http.HandlerFunc) http.Handler {
		return auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, _ := middleware.ClaimsFromContext(r.Context())
			user, err := h.CurrentUser(r.Context(), claims.UserID)
			if errors.Is(err, repository.ErrNotFound) {
				httpx.Error(w, http.StatusUnauthorized, "用户不存在")
				return
			}
			if err != nil {
				httpx.Error(w, http.StatusInternalServerError, "校验管理员权限失败")
				return
			}
			if user.Role != "admin" || user.Status != "active" {
				httpx.Error(w, http.StatusForbidden, "需要管理员权限")
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
	mux.Handle("GET /api/v1/admin/overview", admin(h.AdminOverview))
	mux.Handle("GET /api/v1/admin/users", admin(h.AdminListUsers))
	mux.Handle("POST /api/v1/admin/users", admin(h.AdminCreateUser))
	mux.Handle("PATCH /api/v1/admin/users/{id}", admin(h.AdminUpdateUser))
	mux.Handle("PUT /api/v1/admin/users/{id}", admin(h.AdminEditUser))
	mux.Handle("POST /api/v1/admin/users/{id}/balance", admin(h.AdminAdjustUserBalance))
	mux.Handle("GET /api/v1/admin/apps", admin(h.AdminListApps))
	mux.Handle("POST /api/v1/admin/apps", admin(h.AdminCreateApp))
	mux.Handle("PUT /api/v1/admin/apps/{id}", admin(h.AdminSaveApp))
	mux.Handle("PATCH /api/v1/admin/apps/{id}", admin(h.AdminUpdateApp))
	mux.Handle("GET /api/v1/admin/generations", admin(h.AdminListGenerations))
	mux.Handle("GET /api/v1/admin/payment-orders", admin(h.AdminListPaymentOrders))
	mux.Handle("POST /api/v1/admin/payment-orders/{tradeNo}/cancel", admin(h.AdminCancelPaymentOrder))
	mux.Handle("GET /api/v1/admin/affiliates", admin(h.AdminAffiliateOverview))
	mux.Handle("GET /api/v1/admin/settings", admin(h.AdminListSettings))
	mux.Handle("PUT /api/v1/admin/settings/{key}", admin(h.AdminUpdateSetting))
	mux.Handle("GET /api/v1/admin/api-providers", admin(h.AdminListAPIProviders))
	mux.Handle("POST /api/v1/admin/api-providers", admin(h.AdminSaveAPIProvider))
	mux.Handle("DELETE /api/v1/admin/api-providers/{id}", admin(h.AdminDeleteAPIProvider))
	mux.Handle("POST /api/v1/admin/api-providers/models", admin(h.AdminFetchProviderModels))
	mux.Handle("POST /api/v1/admin/api-providers/{id}/models", admin(h.AdminFetchProviderModelsByID))
	mux.Handle("GET /api/v1/admin/invite-codes", admin(h.AdminListInviteCodes))
	mux.Handle("POST /api/v1/admin/invite-codes", admin(h.AdminCreateInviteCodes))
	mux.Handle("GET /api/v1/admin/logs/login", admin(h.AdminListLoginLogs))
	mux.Handle("GET /api/v1/admin/logs/tasks", admin(h.AdminListTaskLogs))

	return middleware.CORS(cfg.CORSOrigins)(mux)
}
