package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"image-ai/backend/internal/httpx"
	"image-ai/backend/internal/middleware"
	"image-ai/backend/internal/model"
	"image-ai/backend/internal/repository"
	"image-ai/backend/internal/security"
)

func (h *Handler) AdminOverview(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.AdminStats(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取后台统计失败")
		return
	}
	httpx.JSON(w, http.StatusOK, stats)
}

func (h *Handler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	users, err := h.repo.AdminListUsers(r.Context(), limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取用户列表失败")
		return
	}
	httpx.JSON(w, http.StatusOK, users)
}

func (h *Handler) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email           string `json:"email"`
		Password        string `json:"password"`
		Nickname        string `json:"nickname"`
		Role            string `json:"role"`
		Status          string `json:"status"`
		MembershipLevel string `json:"membershipLevel"`
		Credits         int    `json:"credits"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确："+err.Error())
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)
	req.Role = strings.TrimSpace(req.Role)
	req.Status = strings.TrimSpace(req.Status)
	req.MembershipLevel = strings.TrimSpace(req.MembershipLevel)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		httpx.Error(w, http.StatusBadRequest, "邮箱格式不正确")
		return
	}
	if len(req.Password) < 8 {
		httpx.Error(w, http.StatusBadRequest, "密码至少 8 位")
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if req.MembershipLevel == "" {
		req.MembershipLevel = "free"
	}
	if req.Role != "user" && req.Role != "admin" {
		httpx.Error(w, http.StatusBadRequest, "角色不正确")
		return
	}
	if req.Status != "active" && req.Status != "disabled" {
		httpx.Error(w, http.StatusBadRequest, "状态不正确")
		return
	}
	if !validMembershipLevel(req.MembershipLevel) {
		httpx.Error(w, http.StatusBadRequest, "会员等级不正确")
		return
	}
	passwordHash, err := security.HashPassword(req.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "密码加密失败")
		return
	}
	user, err := h.repo.AdminCreateUser(r.Context(), model.User{
		Email:           req.Email,
		Nickname:        strings.TrimSpace(req.Nickname),
		Role:            req.Role,
		Status:          req.Status,
		MembershipLevel: req.MembershipLevel,
		Credits:         req.Credits,
	}, passwordHash)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "创建用户失败")
		return
	}
	httpx.JSON(w, http.StatusCreated, user)
}

func (h *Handler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role   string `json:"role"`
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确："+err.Error())
		return
	}
	req.Role = strings.TrimSpace(req.Role)
	req.Status = strings.TrimSpace(req.Status)
	if req.Role != "" && req.Role != "user" && req.Role != "admin" {
		httpx.Error(w, http.StatusBadRequest, "角色不正确")
		return
	}
	if req.Status != "" && req.Status != "active" && req.Status != "disabled" {
		httpx.Error(w, http.StatusBadRequest, "状态不正确")
		return
	}
	user, err := h.repo.AdminUpdateUser(r.Context(), pathTail(r.URL.Path), req.Role, req.Status)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "用户不存在")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "更新用户失败")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) AdminEditUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email           string `json:"email"`
		Nickname        string `json:"nickname"`
		Password        string `json:"password"`
		Role            string `json:"role"`
		Status          string `json:"status"`
		MembershipLevel string `json:"membershipLevel"`
		Credits         int    `json:"credits"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确："+err.Error())
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email != "" && !strings.Contains(req.Email, "@") {
		httpx.Error(w, http.StatusBadRequest, "邮箱格式不正确")
		return
	}
	if req.Role != "" && req.Role != "user" && req.Role != "admin" {
		httpx.Error(w, http.StatusBadRequest, "角色不正确")
		return
	}
	if req.Status != "" && req.Status != "active" && req.Status != "disabled" {
		httpx.Error(w, http.StatusBadRequest, "状态不正确")
		return
	}
	req.MembershipLevel = strings.TrimSpace(req.MembershipLevel)
	if req.MembershipLevel != "" && !validMembershipLevel(req.MembershipLevel) {
		httpx.Error(w, http.StatusBadRequest, "会员等级不正确")
		return
	}
	var passwordHash string
	if strings.TrimSpace(req.Password) != "" {
		if len(strings.TrimSpace(req.Password)) < 8 {
			httpx.Error(w, http.StatusBadRequest, "密码至少 8 位")
			return
		}
		hash, err := security.HashPassword(strings.TrimSpace(req.Password))
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "密码加密失败")
			return
		}
		passwordHash = hash
	}
	user, err := h.repo.AdminEditUser(r.Context(), model.User{
		ID:              pathTail(r.URL.Path),
		Email:           req.Email,
		Nickname:        strings.TrimSpace(req.Nickname),
		Role:            strings.TrimSpace(req.Role),
		Status:          strings.TrimSpace(req.Status),
		MembershipLevel: req.MembershipLevel,
		Credits:         req.Credits,
	}, passwordHash)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "用户不存在")
		return
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		httpx.Error(w, http.StatusConflict, "邮箱已存在")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "保存用户失败")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) AdminAdjustUserBalance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type   string `json:"type"`
		Amount string `json:"amount"`
		Note   string `json:"note"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确："+err.Error())
		return
	}
	req.Type = strings.TrimSpace(req.Type)
	req.Amount = strings.TrimSpace(req.Amount)
	if req.Type != "increase" && req.Type != "decrease" && req.Type != "set" {
		httpx.Error(w, http.StatusBadRequest, "余额调整类型不正确")
		return
	}
	if req.Amount == "" {
		httpx.Error(w, http.StatusBadRequest, "请输入调整金额")
		return
	}
	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil || amount <= 0 {
		httpx.Error(w, http.StatusBadRequest, "金额必须大于 0")
		return
	}
	claims, _ := middleware.ClaimsFromContext(r.Context())
	userID := pathTail(strings.TrimSuffix(r.URL.Path, "/balance"))
	log, user, err := h.repo.AdminAdjustUserBalance(r.Context(), userID, claims.UserID, req.Type, req.Amount, strings.TrimSpace(req.Note))
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "用户不存在")
		return
	}
	if errors.Is(err, repository.ErrInsufficientBalance) {
		httpx.Error(w, http.StatusBadRequest, "余额不足，不能减少到负数")
		return
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		httpx.Error(w, http.StatusBadRequest, "金额格式不正确")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "调整余额失败")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"log": log, "user": user})
}

func (h *Handler) AdminListApps(w http.ResponseWriter, r *http.Request) {
	apps, err := h.repo.AdminListApps(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取应用列表失败")
		return
	}
	httpx.JSON(w, http.StatusOK, apps)
}

func (h *Handler) AdminCreateApp(w http.ResponseWriter, r *http.Request) {
	app, ok := h.decodeAdminApp(w, r)
	if !ok {
		return
	}
	created, err := h.repo.AdminUpsertApp(r.Context(), app)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "创建应用失败")
		return
	}
	httpx.JSON(w, http.StatusCreated, created)
}

func (h *Handler) AdminSaveApp(w http.ResponseWriter, r *http.Request) {
	app, ok := h.decodeAdminApp(w, r)
	if !ok {
		return
	}
	app.ID = pathTail(r.URL.Path)
	saved, err := h.repo.AdminUpsertApp(r.Context(), app)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "应用不存在")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "保存应用失败")
		return
	}
	httpx.JSON(w, http.StatusOK, saved)
}

func (h *Handler) decodeAdminApp(w http.ResponseWriter, r *http.Request) (model.App, bool) {
	var req struct {
		ID             string          `json:"id"`
		ProviderID     json.RawMessage `json:"providerId"`
		Code           string          `json:"code"`
		Name           string          `json:"name"`
		AppType        string          `json:"appType"`
		Category       string          `json:"category"`
		Description    string          `json:"description"`
		Icon           string          `json:"icon"`
		IconColor      string          `json:"iconColor"`
		CoverURL       string          `json:"coverUrl"`
		PromptTemplate string          `json:"promptTemplate"`
		InputSchema    json.RawMessage `json:"inputSchema"`
		OutputSchema   json.RawMessage `json:"outputSchema"`
		PriceFree      string          `json:"priceFree"`
		PriceV1        string          `json:"priceV1"`
		PriceV2        string          `json:"priceV2"`
		Visibility     string          `json:"visibility"`
		Status         string          `json:"status"`
		SortOrder      int             `json:"sortOrder"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确："+err.Error())
		return model.App{}, false
	}
	req.Code = strings.TrimSpace(req.Code)
	providerID := stringFromJSONSelectValue(req.ProviderID)
	req.Name = strings.TrimSpace(req.Name)
	req.AppType = strings.TrimSpace(req.AppType)
	req.Category = strings.TrimSpace(req.Category)
	req.Visibility = strings.TrimSpace(req.Visibility)
	req.Status = strings.TrimSpace(req.Status)
	if req.Code == "" || req.Name == "" || req.Category == "" {
		httpx.Error(w, http.StatusBadRequest, "应用标识、名称和分类必填")
		return model.App{}, false
	}
	if req.AppType == "" {
		req.AppType = "image"
	}
	if req.AppType != "image" && req.AppType != "text" {
		httpx.Error(w, http.StatusBadRequest, "应用类型不正确")
		return model.App{}, false
	}
	if providerID == "" {
		httpx.Error(w, http.StatusBadRequest, "应用必须选择一条接口配置")
		return model.App{}, false
	}
	provider, err := h.repo.AdminAPIProviderByID(r.Context(), providerID)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusBadRequest, "选择的接口配置不存在")
		return model.App{}, false
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "读取接口配置失败")
		return model.App{}, false
	}
	expectedCategory := expectedAppProviderCategory(req.AppType)
	if provider.Category != expectedCategory {
		httpx.Error(w, http.StatusBadRequest, "应用类型与接口分类不匹配")
		return model.App{}, false
	}
	if req.Visibility == "" {
		req.Visibility = "public"
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if req.Visibility != "public" && req.Visibility != "private" {
		httpx.Error(w, http.StatusBadRequest, "可见性不正确")
		return model.App{}, false
	}
	if req.Status != "active" && req.Status != "disabled" {
		httpx.Error(w, http.StatusBadRequest, "状态不正确")
		return model.App{}, false
	}
	if len(req.InputSchema) == 0 {
		req.InputSchema = json.RawMessage(`{}`)
	}
	if len(req.OutputSchema) == 0 {
		req.OutputSchema = json.RawMessage(`{}`)
	}
	if !json.Valid(req.InputSchema) || !json.Valid(req.OutputSchema) {
		httpx.Error(w, http.StatusBadRequest, "Schema 必须是合法 JSON")
		return model.App{}, false
	}
	return model.App{
		ID:             strings.TrimSpace(req.ID),
		ProviderID:     &providerID,
		Code:           req.Code,
		Name:           req.Name,
		AppType:        req.AppType,
		Category:       req.Category,
		Description:    strings.TrimSpace(req.Description),
		Icon:           strings.TrimSpace(req.Icon),
		IconColor:      strings.TrimSpace(req.IconColor),
		CoverURL:       strings.TrimSpace(req.CoverURL),
		PromptTemplate: strings.TrimSpace(req.PromptTemplate),
		InputSchema:    req.InputSchema,
		OutputSchema:   req.OutputSchema,
		PriceFree:      strings.TrimSpace(req.PriceFree),
		PriceV1:        strings.TrimSpace(req.PriceV1),
		PriceV2:        strings.TrimSpace(req.PriceV2),
		Visibility:     req.Visibility,
		Status:         req.Status,
		SortOrder:      req.SortOrder,
	}, true
}

func (h *Handler) AdminUpdateApp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status     string `json:"status"`
		Visibility string `json:"visibility"`
		SortOrder  *int   `json:"sortOrder"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确："+err.Error())
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	req.Visibility = strings.TrimSpace(req.Visibility)
	if req.Status != "" && req.Status != "active" && req.Status != "disabled" {
		httpx.Error(w, http.StatusBadRequest, "状态不正确")
		return
	}
	if req.Visibility != "" && req.Visibility != "public" && req.Visibility != "private" {
		httpx.Error(w, http.StatusBadRequest, "可见性不正确")
		return
	}
	app, err := h.repo.AdminUpdateApp(r.Context(), pathTail(r.URL.Path), req.Status, req.Visibility, req.SortOrder)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "应用不存在")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "更新应用失败")
		return
	}
	httpx.JSON(w, http.StatusOK, app)
}

func (h *Handler) AdminListGenerations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	jobs, err := h.repo.AdminListGenerations(r.Context(), limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取生成任务失败")
		return
	}
	httpx.JSON(w, http.StatusOK, jobs)
}

func (h *Handler) AdminListSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.AdminListSettings(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取站点配置失败")
		return
	}
	httpx.JSON(w, http.StatusOK, settings)
}

func (h *Handler) AdminUpdateSetting(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value json.RawMessage `json:"value"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确："+err.Error())
		return
	}
	if !json.Valid(req.Value) {
		httpx.Error(w, http.StatusBadRequest, "配置必须是合法 JSON")
		return
	}
	key := pathTail(r.URL.Path)
	if key != "seo" && key != "auth" && key != "smtp" && key != "payment" && key != "professional_draw" {
		httpx.Error(w, http.StatusBadRequest, "配置项不正确")
		return
	}
	setting, err := h.repo.AdminUpdateSetting(r.Context(), key, req.Value)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "保存站点配置失败")
		return
	}
	httpx.JSON(w, http.StatusOK, setting)
}

func (h *Handler) AdminListAPIProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.repo.AdminListAPIProviders(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取接口配置失败")
		return
	}
	httpx.JSON(w, http.StatusOK, providers)
}

func normalizeAPIProviderCategory(category string) string {
	switch strings.TrimSpace(category) {
	case "通用生图":
		return "general"
	case "通用文本":
		return "general_text"
	default:
		return strings.TrimSpace(category)
	}
}

func expectedAppProviderCategory(appType string) string {
	if strings.TrimSpace(appType) == "text" {
		return "general_text"
	}
	return "general"
}

func stringFromJSONSelectValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return strings.TrimSpace(value)
	}
	var item struct {
		Value string `json:"value"`
		ID    string `json:"id"`
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return ""
	}
	if strings.TrimSpace(item.Value) != "" {
		return strings.TrimSpace(item.Value)
	}
	return strings.TrimSpace(item.ID)
}

func (h *Handler) AdminSaveAPIProvider(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Category  string `json:"category"`
		Provider  string `json:"provider"`
		BaseURL   string `json:"baseUrl"`
		APIKey    string `json:"apiKey"`
		Model     string `json:"model"`
		Enabled   bool   `json:"enabled"`
		SortOrder int    `json:"sortOrder"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确："+err.Error())
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Category = strings.TrimSpace(req.Category)
	req.Category = normalizeAPIProviderCategory(req.Category)
	req.Provider = strings.TrimSpace(req.Provider)
	if req.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "接口名称必填")
		return
	}
	if req.Category == "" {
		req.Category = "general"
	}
	if req.Category != "general" && req.Category != "general_text" {
		httpx.Error(w, http.StatusBadRequest, "接口分类不正确")
		return
	}
	req.Provider = "openai"
	if strings.TrimSpace(req.BaseURL) == "" {
		httpx.Error(w, http.StatusBadRequest, "OpenAI Base URL 蹇呭～")
		return
	}
	if strings.TrimSpace(req.APIKey) == "" {
		if strings.TrimSpace(req.ID) == "" {
			httpx.Error(w, http.StatusBadRequest, "OpenAI API Key 蹇呭～")
			return
		}
		existing, err := h.repo.AdminAPIProviderByID(r.Context(), strings.TrimSpace(req.ID))
		if errors.Is(err, repository.ErrNotFound) || strings.TrimSpace(existing.APIKey) == "" {
			httpx.Error(w, http.StatusBadRequest, "OpenAI API Key 蹇呭～")
			return
		}
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "璇诲彇鎺ュ彛閰嶇疆澶辫触")
			return
		}
	}
	if req.Enabled && strings.TrimSpace(req.Model) == "" {
		httpx.Error(w, http.StatusBadRequest, "OpenAI 妯″瀷蹇呭～")
		return
	}
	if req.Category == "professional_drawing" {
		req.Provider = "openai"
		if strings.TrimSpace(req.BaseURL) == "" {
			httpx.Error(w, http.StatusBadRequest, "专业绘画接口 Base URL 必填")
			return
		}
		if strings.TrimSpace(req.APIKey) == "" {
			if strings.TrimSpace(req.ID) == "" {
				httpx.Error(w, http.StatusBadRequest, "专业绘画接口 API Key 必填")
				return
			}
			existing, err := h.repo.AdminAPIProviderByID(r.Context(), strings.TrimSpace(req.ID))
			if errors.Is(err, repository.ErrNotFound) || strings.TrimSpace(existing.APIKey) == "" {
				httpx.Error(w, http.StatusBadRequest, "专业绘画接口 API Key 必填")
				return
			}
			if err != nil {
				httpx.Error(w, http.StatusInternalServerError, "读取接口配置失败")
				return
			}
		}
		if req.Enabled && strings.TrimSpace(req.Model) == "" {
			httpx.Error(w, http.StatusBadRequest, "专业绘画接口模型必填")
			return
		}
	}
	if req.Category == "assistant_chat" {
		req.Provider = "openai"
		if strings.TrimSpace(req.BaseURL) == "" {
			httpx.Error(w, http.StatusBadRequest, "智能助手接口 Base URL 必填")
			return
		}
		if strings.TrimSpace(req.APIKey) == "" {
			if strings.TrimSpace(req.ID) == "" {
				httpx.Error(w, http.StatusBadRequest, "智能助手接口 API Key 必填")
				return
			}
			existing, err := h.repo.AdminAPIProviderByID(r.Context(), strings.TrimSpace(req.ID))
			if errors.Is(err, repository.ErrNotFound) || strings.TrimSpace(existing.APIKey) == "" {
				httpx.Error(w, http.StatusBadRequest, "智能助手接口 API Key 必填")
				return
			}
			if err != nil {
				httpx.Error(w, http.StatusInternalServerError, "读取接口配置失败")
				return
			}
		}
		if req.Enabled && strings.TrimSpace(req.Model) == "" {
			httpx.Error(w, http.StatusBadRequest, "智能助手模型必填")
			return
		}
	}
	provider, err := h.repo.AdminUpsertAPIProvider(r.Context(), model.APIProvider{
		ID:        strings.TrimSpace(req.ID),
		Name:      req.Name,
		Category:  req.Category,
		Provider:  req.Provider,
		BaseURL:   strings.TrimSpace(req.BaseURL),
		APIKey:    strings.TrimSpace(req.APIKey),
		Model:     strings.TrimSpace(req.Model),
		Enabled:   req.Enabled,
		SortOrder: req.SortOrder,
	})
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "接口配置不存在")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "保存接口配置失败")
		return
	}
	httpx.JSON(w, http.StatusOK, provider)
}

func (h *Handler) AdminDeleteAPIProvider(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.AdminDeleteAPIProvider(r.Context(), pathTail(r.URL.Path)); errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "接口配置不存在")
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "删除接口配置失败，请确认没有应用正在使用该接口")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) AdminFetchProviderModels(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确："+err.Error())
		return
	}
	models, err := fetchOpenAIModels(r.Context(), req.BaseURL, req.APIKey)
	if err != nil {
		httpx.Error(w, statusForModelFetchError(err), err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, models)
}

func (h *Handler) AdminFetchProviderModelsByID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确："+err.Error())
		return
	}
	provider, err := h.repo.AdminAPIProviderByID(r.Context(), pathTail(strings.TrimSuffix(r.URL.Path, "/models")))
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "接口配置不存在")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "读取接口配置失败")
		return
	}
	if provider.Provider != "openai" {
		httpx.Error(w, http.StatusBadRequest, "当前仅支持 OpenAI 兼容接口获取模型")
		return
	}
	baseURL := provider.BaseURL
	if strings.TrimSpace(req.BaseURL) != "" {
		baseURL = strings.TrimSpace(req.BaseURL)
	}
	apiKey := provider.APIKey
	if strings.TrimSpace(req.APIKey) != "" {
		apiKey = strings.TrimSpace(req.APIKey)
	}
	models, err := fetchOpenAIModels(r.Context(), baseURL, apiKey)
	if err != nil {
		httpx.Error(w, statusForModelFetchError(err), err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, models)
}

func fetchOpenAIModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, modelFetchClientError("Base URL 不能为空")
	}

	endpoints, err := openAIModelEndpoints(baseURL)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, endpoint := range endpoints {
		models, err := fetchOpenAIModelsFromEndpoint(ctx, endpoint, apiKey)
		if err == nil {
			return models, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("获取模型失败")
}

func fetchOpenAIModelsFromEndpoint(ctx context.Context, endpoint, apiKey string) ([]string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, modelFetchClientError("Base URL 不正确")
	}
	if strings.TrimSpace(apiKey) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}
	request.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("获取模型失败：%w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, modelFetchClientError("获取模型失败：API Key 未填写或无效")
		}
		return nil, fmt.Errorf("获取模型失败：上游返回 %d，%s", resp.StatusCode, message)
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, errors.New("模型响应格式不正确，请确认 Base URL 指向 OpenAI 兼容接口地址，例如 https://ai.cangyuansuanli.cn/v1")
	}
	models := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		if strings.TrimSpace(item.ID) != "" {
			models = append(models, item.ID)
		}
	}
	if len(models) == 0 {
		return nil, errors.New("未获取到模型，请确认该渠道 /models 接口返回 data[].id")
	}
	return models, nil
}

func openAIModelEndpoints(baseURL string) ([]string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, modelFetchClientError("Base URL 不正确")
	}

	candidates := []string{}
	add := func(value string) {
		value = strings.TrimRight(value, "/")
		for _, item := range candidates {
			if item == value {
				return
			}
		}
		candidates = append(candidates, value)
	}

	path := strings.TrimRight(parsed.Path, "/")
	if strings.HasSuffix(path, "/models") {
		add(parsed.String())
	}

	if index := strings.Index(path, "/v1"); index >= 0 {
		copy := *parsed
		copy.Path = path[:index] + "/v1/models"
		copy.RawQuery = ""
		add(copy.String())
	}

	copy := *parsed
	copy.RawQuery = ""
	copy.Path = path + "/models"
	add(copy.String())

	if !strings.HasSuffix(path, "/v1") && !strings.Contains(path, "/v1/") {
		copy := *parsed
		copy.Path = path + "/v1/models"
		copy.RawQuery = ""
		add(copy.String())
	}

	return candidates, nil
}

type modelFetchClientError string

func (e modelFetchClientError) Error() string {
	return string(e)
}

func statusForModelFetchError(err error) int {
	var clientErr modelFetchClientError
	if errors.As(err, &clientErr) {
		return http.StatusBadRequest
	}
	return http.StatusBadGateway
}

func (h *Handler) AdminListLoginLogs(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := h.repo.AdminListLoginLogs(r.Context(), limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取登录日志失败")
		return
	}
	httpx.JSON(w, http.StatusOK, logs)
}

func (h *Handler) AdminListTaskLogs(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := h.repo.AdminListTaskLogs(r.Context(), limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取任务日志失败")
		return
	}
	httpx.JSON(w, http.StatusOK, logs)
}

func (h *Handler) AdminListPaymentOrders(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	orders, err := h.repo.AdminListPaymentOrders(r.Context(), limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取订单列表失败")
		return
	}
	httpx.JSON(w, http.StatusOK, orders)
}

func (h *Handler) AdminCancelPaymentOrder(w http.ResponseWriter, r *http.Request) {
	tradeNo := pathTail(r.URL.Path)
	order, err := h.repo.AdminCancelPaymentOrder(r.Context(), tradeNo)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "订单不存在或不可取消")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "取消订单失败")
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}

func (h *Handler) AdminAffiliateOverview(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	overview, err := h.repo.AdminAffiliateOverview(r.Context(), limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取推广返佣数据失败")
		return
	}
	httpx.JSON(w, http.StatusOK, overview)
}

func (h *Handler) AdminListInviteCodes(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	codes, err := h.repo.AdminListInviteCodes(r.Context(), limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "获取兑换码失败")
		return
	}
	httpx.JSON(w, http.StatusOK, codes)
}

func (h *Handler) AdminCreateInviteCodes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Count     int    `json:"count"`
		Amount    string `json:"amount"`
		Note      string `json:"note"`
		ExpiresAt string `json:"expiresAt"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确")
		return
	}
	if req.Count <= 0 || req.Count > 200 {
		httpx.Error(w, http.StatusBadRequest, "一次可生成 1-200 个兑换码")
		return
	}
	req.Amount = strings.TrimSpace(req.Amount)
	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil || amount <= 0 {
		httpx.Error(w, http.StatusBadRequest, "兑换码面值必须大于 0")
		return
	}

	var expiresAt *time.Time
	if strings.TrimSpace(req.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "过期时间格式不正确")
			return
		}
		expiresAt = &parsed
	}

	claims, _ := middleware.ClaimsFromContext(r.Context())
	createdBy := claims.UserID
	codes := make([]model.InviteCode, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		code, err := randomInviteCode()
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "兑换码生成失败")
			return
		}
		codes = append(codes, model.InviteCode{
			Code:      code,
			Amount:    req.Amount,
			MaxUses:   1,
			Note:      strings.TrimSpace(req.Note),
			CreatedBy: &createdBy,
			ExpiresAt: expiresAt,
		})
	}
	created, err := h.repo.AdminCreateInviteCodes(r.Context(), codes)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "保存兑换码失败")
		return
	}
	httpx.JSON(w, http.StatusCreated, created)
}

func randomInviteCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, 17)
	for i := range out {
		if i == 0 {
			out[i] = 'R'
			continue
		}
		if i == 1 {
			out[i] = 'C'
			continue
		}
		if i == 2 || i == 7 || i == 12 {
			out[i] = '-'
			continue
		}
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out), nil
}

func validMembershipLevel(level string) bool {
	return level == "free" || level == "v1" || level == "v2"
}
