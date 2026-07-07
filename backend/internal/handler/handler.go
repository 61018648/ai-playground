package handler

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"image-ai/backend/internal/config"
	"image-ai/backend/internal/httpx"
	"image-ai/backend/internal/middleware"
	"image-ai/backend/internal/model"
	"image-ai/backend/internal/repository"
	"image-ai/backend/internal/security"
)

type Handler struct {
	cfg  config.Config
	repo *repository.Repository
}

var errSMTPNotConfigured = errors.New("smtp not configured")

func New(cfg config.Config, repo *repository.Repository) *Handler {
	return &Handler{cfg: cfg, repo: repo}
}

func (h *Handler) SendCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email   string `json:"email"`
		Purpose string `json:"purpose"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵?")
		return
	}
	req.Email = normalizeEmail(req.Email)
	req.Purpose = strings.TrimSpace(req.Purpose)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		httpx.Error(w, http.StatusBadRequest, "闁喚顔堥弽鐓庣础娑撳秵顒滅涵?")
		return
	}
	if req.Purpose != "register" && req.Purpose != "forgot_password" && req.Purpose != "login" && req.Purpose != "change_email" {
		httpx.Error(w, http.StatusBadRequest, "妤犲矁鐦夐惍浣烘暏闁柧绗夊锝団€?")
		return
	}

	code, err := security.NewNumericCode(6)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "妤犲矁鐦夐惍浣烘晸閹存劕銇戠拹?")
		return
	}
	hash := security.HashCode(req.Email, req.Purpose, code)
	if err := h.repo.CreateVerificationCode(r.Context(), req.Email, req.Purpose, hash, time.Now().Add(10*time.Minute)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "妤犲矁鐦夐惍浣风箽鐎涙ê銇戠拹?")
		return
	}
	if err := h.sendVerificationCode(r.Context(), req.Email, req.Purpose, code); err != nil {
		log.Printf("send verification email failed email=%s purpose=%s: %v", req.Email, req.Purpose, err)
		if h.cfg.Env == "production" || !errors.Is(err, errSMTPNotConfigured) {
			httpx.Error(w, http.StatusInternalServerError, "妤犲矁鐦夐惍渚€鍋栨禒璺哄絺闁礁銇戠拹?")
			return
		}
	}

	resp := map[string]any{"expiresIn": 600}
	if h.cfg.Env != "production" {
		resp["devCode"] = code
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type smtpSetting struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FromName  string `json:"fromName"`
	FromEmail string `json:"fromEmail"`
	Secure    bool   `json:"secure"`
}

func (h *Handler) sendVerificationCode(ctx context.Context, email, purpose, code string) error {
	setting, err := h.repo.SiteSetting(ctx, "smtp")
	if errors.Is(err, repository.ErrNotFound) {
		return errSMTPNotConfigured
	}
	if err != nil {
		return err
	}
	var cfg smtpSetting
	if err := json.Unmarshal(setting.Value, &cfg); err != nil {
		return err
	}
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.Password = strings.TrimSpace(cfg.Password)
	cfg.FromName = strings.TrimSpace(cfg.FromName)
	cfg.FromEmail = strings.TrimSpace(cfg.FromEmail)
	if cfg.Host == "" || cfg.FromEmail == "" {
		return errSMTPNotConfigured
	}
	if cfg.Port == 0 {
		if cfg.Secure {
			cfg.Port = 465
		} else {
			cfg.Port = 587
		}
	}
	if cfg.FromName == "" {
		cfg.FromName = "閹芥ɑ妲I"
	}

	subject := "鎽樻槦AI閭楠岃瘉鐮?"
	body := fmt.Sprintf("您的验证码是：%s\n\n验证码 10 分钟内有效，请勿泄露给他人。\n用途：%s", code, verificationPurposeLabel(purpose))
	return sendSMTPMail(cfg, email, subject, body)
}

func verificationPurposeLabel(purpose string) string {
	switch purpose {
	case "register":
		return "濞夈劌鍞界拹锕€褰?"
	case "forgot_password":
		return "閹垫儳娲栫€靛棛鐖?"
	case "change_email":
		return "閹广垻绮﹂柇顔绢唸"
	case "login":
		return "閻ц缍嶆宀冪槈"
	default:
		return "闁喚顔堟宀冪槈"
	}
}

func sendSMTPMail(cfg smtpSetting, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	fromHeader := mime.QEncoding.Encode("UTF-8", cfg.FromName) + " <" + cfg.FromEmail + ">"
	subjectHeader := mime.QEncoding.Encode("UTF-8", subject)
	msg := strings.Join([]string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + subjectHeader,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if cfg.Username != "" || cfg.Password != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	if cfg.Secure {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return err
		}
		defer client.Quit()
		return sendSMTPClientMail(client, auth, cfg.FromEmail, to, []byte(msg))
	}
	return smtp.SendMail(addr, auth, cfg.FromEmail, []string{to}, []byte(msg))
}

func sendSMTPClientMail(client *smtp.Client, auth smtp.Auth, from, to string, msg []byte) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(msg); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		Code       string `json:"code"`
		InviteCode string `json:"inviteCode"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵?")
		return
	}
	req.Email = normalizeEmail(req.Email)
	if err := validatePassword(req.Password); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ok, err := h.repo.ConsumeVerificationCode(r.Context(), req.Email, "register", security.HashCode(req.Email, "register", req.Code))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "妤犲矁鐦夐惍浣圭墡妤犲苯銇戠拹?")
		return
	}
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "妤犲矁鐦夐惍浣规￥閺佸牊鍨ㄥ鑼剁箖閺?")
		return
	}
	hash, err := security.HashPassword(req.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "鐎靛棛鐖滄径鍕倞婢惰精瑙?")
		return
	}
	nickname := strings.Split(req.Email, "@")[0]
	user, err := h.repo.CreateUser(r.Context(), req.Email, hash, nickname)
	if err != nil {
		log.Printf("register create user failed email=%s: %v", req.Email, err)
		httpx.Error(w, http.StatusConflict, "闁喚顔堝鍙夋暈閸?")
		return
	}
	if strings.TrimSpace(req.InviteCode) != "" {
		_ = h.repo.RecordAffiliateReferral(r.Context(), strings.TrimSpace(req.InviteCode), user.ID)
	}
	token, err := security.SignToken(h.cfg.JWTSecret, security.Claims{UserID: user.ID, Email: user.Email, Role: user.Role}, h.cfg.TokenTTL)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閻ц缍嶆禒銈囧閻㈢喐鍨氭径杈Е")
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"user": user, "accessToken": token})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵?")
		return
	}
	user, passwordHash, err := h.repo.UserByEmail(r.Context(), normalizeEmail(req.Email))
	if errors.Is(err, repository.ErrNotFound) || !security.CheckPassword(passwordHash, req.Password) {
		_ = h.repo.RecordLoginLog(r.Context(), nil, normalizeEmail(req.Email), false, clientIP(r), r.UserAgent(), "闁喚顔堥幋鏍х槕閻椒绗夊锝団€?")
		httpx.Error(w, http.StatusUnauthorized, "闁喚顔堥幋鏍х槕閻椒绗夊锝団€?")
		return
	}
	if err != nil {
		_ = h.repo.RecordLoginLog(r.Context(), nil, normalizeEmail(req.Email), false, clientIP(r), r.UserAgent(), "閻ц缍嶆径杈Е")
		httpx.Error(w, http.StatusInternalServerError, "閻ц缍嶆径杈Е")
		return
	}
	ttl := h.cfg.TokenTTL
	if req.Remember {
		ttl = 7 * 24 * time.Hour
	}
	token, err := security.SignToken(h.cfg.JWTSecret, security.Claims{UserID: user.ID, Email: user.Email, Role: user.Role}, ttl)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閻ц缍嶆禒銈囧閻㈢喐鍨氭径杈Е")
		return
	}
	_ = h.repo.RecordLoginLog(r.Context(), &user.ID, user.Email, true, clientIP(r), r.UserAgent(), "閻ц缍嶉幋鎰")
	httpx.JSON(w, http.StatusOK, map[string]any{"user": user, "accessToken": token})
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"newPassword"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵?")
		return
	}
	req.Email = normalizeEmail(req.Email)
	if err := validatePassword(req.NewPassword); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ok, err := h.repo.ConsumeVerificationCode(r.Context(), req.Email, "forgot_password", security.HashCode(req.Email, "forgot_password", req.Code))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "妤犲矁鐦夐惍浣圭墡妤犲苯銇戠拹?")
		return
	}
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "妤犲矁鐦夐惍浣规￥閺佸牊鍨ㄥ鑼剁箖閺?")
		return
	}
	hash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "鐎靛棛鐖滄径鍕倞婢惰精瑙?")
		return
	}
	if err := h.repo.UpdatePassword(r.Context(), req.Email, hash); err != nil {
		httpx.Error(w, http.StatusNotFound, "閻劍鍩涙稉宥呯摠閸?")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	user, err := h.repo.UserByID(r.Context(), claims.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusUnauthorized, "閻劍鍩涙稉宥呯摠閸?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囬悽銊﹀煕婢惰精瑙?")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) ListApps(w http.ResponseWriter, r *http.Request) {
	apps, err := h.repo.ListApps(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囨惔鏃傛暏閸掓銆冩径杈Е")
		return
	}
	httpx.JSON(w, http.StatusOK, apps)
}

func (h *Handler) GetApp(w http.ResponseWriter, r *http.Request) {
	app, err := h.repo.AppByID(r.Context(), pathTail(r.URL.Path))
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "鎼存梻鏁ゆ稉宥呯摠閸?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囨惔鏃傛暏婢惰精瑙?")
		return
	}
	httpx.JSON(w, http.StatusOK, app)
}

func (h *Handler) CreateGeneration(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var req struct {
		AppID            *string         `json:"appId"`
		Prompt           string          `json:"prompt"`
		NegativePrompt   string          `json:"negativePrompt"`
		Params           json.RawMessage `json:"params"`
		Model            string          `json:"model"`
		ProviderCategory string          `json:"providerCategory"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵?")
		return
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Prompt == "" {
		httpx.Error(w, http.StatusBadRequest, "鐠囩柉绶崗銉﹀絹缁€楦跨槤")
		return
	}
	if req.AppID != nil {
		value := strings.TrimSpace(*req.AppID)
		if value == "" {
			req.AppID = nil
		} else {
			req.AppID = &value
		}
	}

	effectivePrompt := req.Prompt
	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		modelName = "placeholder-v1"
	}
	if req.AppID != nil {
		app, provider, err := h.resolveGenerationAppConfig(r.Context(), *req.AppID)
		if errors.Is(err, repository.ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "搴旂敤涓嶅瓨鍦ㄦ垨鎺ュ彛閰嶇疆涓嶅彲鐢?")
			return
		}
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		effectivePrompt = renderAppPromptTemplate(app.PromptTemplate, req.Prompt, req.Params)
		if strings.TrimSpace(provider.Model) != "" {
			modelName = provider.Model
		}
	} else if provider, err := h.repo.EnabledAPIProviderByCategory(r.Context(), "general"); err == nil {
		if strings.TrimSpace(provider.Model) != "" {
			modelName = provider.Model
		}
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusInternalServerError, "璇诲彇鎺ュ彛閰嶇疆澶辫触")
		return
	}

	job, err := h.repo.CreateGeneration(r.Context(), claims.UserID, req.AppID, effectivePrompt, req.NegativePrompt, req.Params, modelName)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閸掓稑缂撻悽鐔稿灇娴犺濮熸径杈Е")
		return
	}
	_ = h.repo.RecordTaskLog(r.Context(), &job.ID, &claims.UserID, "generation.create", job.Status, "閸掓稑缂撻悽鐔稿灇娴犺濮?", job.Params)
	// V1 uses a synchronous placeholder result so frontend history can be wired immediately.
	if err := h.repo.CompleteGenerationPlaceholder(r.Context(), job.ID); err != nil {
		log.Printf("complete placeholder generation %s: %v", job.ID, err)
		_ = h.repo.RecordTaskLog(r.Context(), &job.ID, &claims.UserID, "generation.complete", "failed", err.Error(), nil)
		httpx.Error(w, http.StatusInternalServerError, "閻㈢喐鍨氭禒璇插閸楃姳缍呯紒鎾寸亯閸愭瑥鍙嗘径杈Е")
		return
	}
	_ = h.repo.RecordTaskLog(r.Context(), &job.ID, &claims.UserID, "generation.complete", "succeeded", "閸楃姳缍呴悽鐔稿灇鐎瑰本鍨?", nil)
	job, err = h.repo.GenerationByID(r.Context(), claims.UserID, job.ID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囬悽鐔稿灇缂佹挻鐏夋径杈Е")
		return
	}

	httpx.JSON(w, http.StatusCreated, job)
}

func (h *Handler) resolveGenerationAppConfig(ctx context.Context, appID string) (model.App, model.APIProvider, error) {
	app, err := h.repo.AppByID(ctx, appID)
	if err != nil {
		return model.App{}, model.APIProvider{}, err
	}
	if app.ProviderID == nil || strings.TrimSpace(*app.ProviderID) == "" {
		return model.App{}, model.APIProvider{}, repository.ErrNotFound
	}
	provider, err := h.repo.AdminAPIProviderByID(ctx, strings.TrimSpace(*app.ProviderID))
	if err != nil {
		return app, provider, err
	}
	expectedCategory := "general"
	if app.AppType == "text" {
		expectedCategory = "general_text"
	}
	if provider.Category != expectedCategory || !provider.Enabled {
		return app, model.APIProvider{}, repository.ErrNotFound
	}
	return app, provider, nil
}

func renderAppPromptTemplate(template string, prompt string, params json.RawMessage) string {
	template = strings.TrimSpace(template)
	if template == "" {
		template = "{{prompt}}"
	}
	values := map[string]string{
		"prompt": strings.TrimSpace(prompt),
	}
	var data map[string]any
	if len(params) > 0 && json.Unmarshal(params, &data) == nil {
		for key, value := range data {
			values[key] = fmt.Sprint(value)
		}
	}
	for key, value := range values {
		template = strings.ReplaceAll(template, "{{"+key+"}}", value)
	}
	return template
}

func (h *Handler) CreateProfessionalDraw(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var req struct {
		ConversationID string          `json:"conversationId"`
		Prompt         string          `json:"prompt"`
		Params         json.RawMessage `json:"params"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵?")
		return
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Prompt == "" {
		httpx.Error(w, http.StatusBadRequest, "鐠囩柉绶崗銉﹀絹缁€楦跨槤")
		return
	}
	user, err := h.repo.UserByID(r.Context(), claims.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusUnauthorized, "閻劍鍩涙稉宥呯摠閸?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囬悽銊﹀煕婢惰精瑙?")
		return
	}
	app, err := h.repo.AppByCode(r.Context(), "ai-drawing")
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "娑撴挷绗熺紒妯兼暰娴溠冩惂閺堫亜鎯庨悽?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "鐠囪褰囨禍褍鎼ч柊宥囩枂婢惰精瑙?")
		return
	}
	provider, providerErr := h.professionalDrawProvider(r.Context(), "drawProviderId")
	if errors.Is(providerErr, repository.ErrNotFound) {
		httpx.Error(w, http.StatusBadRequest, "鐠囧嘲鍘涢崷銊ユ倵閸欓绗撴稉姘辩帛閸ュ彞鑵戦柅澶嬪楠炶泛鎯庨悽銊ф晸閸ョ偓甯撮崣?")
		return
	}
	if providerErr != nil {
		httpx.Error(w, http.StatusInternalServerError, providerErr.Error())
		return
	}
	if provider.Provider != "openai" {
		httpx.Error(w, http.StatusBadRequest, "娑撴挷绗熺紒妯兼暰閻╊喖澧犳禒鍛暜閹?OpenAI 閸忕厧顔愰幒銉ュ經")
		return
	}
	if strings.TrimSpace(provider.BaseURL) == "" || strings.TrimSpace(provider.APIKey) == "" || strings.TrimSpace(provider.Model) == "" {
		httpx.Error(w, http.StatusBadRequest, "娑撴挷绗熺紒妯兼暰閹恒儱褰?Base URL閵嗕竸PI Key 閸滃本膩閸ㄥ鎮曠粔鏉跨箑婵?")
		return
	}
	modelName := provider.Model
	price := priceForMembership(app, user.MembershipLevel)
	conversationID := strings.TrimSpace(req.ConversationID)
	effectivePrompt := req.Prompt
	sourceImageURL := ""
	if conversationID != "" {
		detail, err := h.repo.ConversationByID(r.Context(), claims.UserID, conversationID)
		if errors.Is(err, repository.ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "缂佹鏁炬导姘崇樈娑撳秴鐡ㄩ崷?")
			return
		}
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "鐠囪褰囩紒妯兼暰娑撳﹣绗呴弬鍥с亼鐠?")
			return
		}
		effectivePrompt = buildDrawContextPrompt(detail.Messages, req.Prompt)
		sourceImageURL = latestDrawAssetURL(detail.Messages)
	}
	var result model.DrawConversationResult
	if conversationID != "" {
		result, err = h.repo.CreateChargedDrawMessage(r.Context(), claims.UserID, conversationID, app, req.Prompt, effectivePrompt, req.Params, modelName, price)
	} else {
		result, err = h.repo.CreateChargedDrawConversation(r.Context(), claims.UserID, app, req.Prompt, req.Params, modelName, price)
	}
	if errors.Is(err, repository.ErrInsufficientBalance) {
		httpx.Error(w, http.StatusPaymentRequired, "娴ｆ瑩顤傛稉宥堝喕")
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "缂佹鏁炬导姘崇樈娑撳秴鐡ㄩ崷?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閹绘劒姘︽稉鎾茬瑹缂佹鏁炬禒璇插婢惰精瑙?")
		return
	}
	_ = h.repo.RecordTaskLog(r.Context(), &result.Job.ID, &claims.UserID, "generation.professional_draw", result.Job.Status, "涓撲笟缁樺浘浠诲姟宸叉彁浜?", req.Params)
	httpx.JSON(w, http.StatusCreated, result)
	go h.runProfessionalDrawJob(context.Background(), provider, result.ConversationID, result.Job.ID, effectivePrompt, req.Params, sourceImageURL)
}

type professionalDrawSetting struct {
	DrawProviderID    string `json:"drawProviderId"`
	RewriteProviderID string `json:"rewriteProviderId"`
}

func (h *Handler) professionalDrawProvider(ctx context.Context, field string) (model.APIProvider, error) {
	var cfg professionalDrawSetting
	setting, err := h.repo.SiteSetting(ctx, "professional_draw")
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return model.APIProvider{}, errors.New("鐠囪褰囨稉鎾茬瑹缂佹ê娴橀柊宥囩枂婢惰精瑙?")
	}
	if err == nil {
		_ = json.Unmarshal(setting.Value, &cfg)
	}

	providerID := strings.TrimSpace(cfg.DrawProviderID)
	expectedCategory := "general"
	fallbackCategory := "professional_drawing"
	if field == "rewriteProviderId" {
		providerID = strings.TrimSpace(cfg.RewriteProviderID)
		expectedCategory = "general_text"
		fallbackCategory = "assistant_chat"
	}
	if providerID != "" {
		provider, err := h.repo.AdminAPIProviderByID(ctx, providerID)
		if err != nil {
			return provider, err
		}
		if provider.Category != expectedCategory {
			return model.APIProvider{}, errors.New("娑撴挷绗熺紒妯烘禈閸欘亣鍏橀柅澶嬪闁氨鏁ら幒銉ュ經闁板秶鐤?")
		}
		if !provider.Enabled {
			return model.APIProvider{}, errors.New("娑撴挷绗熺紒妯烘禈闁瀚ㄩ惃鍕复閸欙絾婀崥顖滄暏")
		}
		return provider, nil
	}

	provider, err := h.repo.EnabledAPIProviderByCategory(ctx, expectedCategory)
	if errors.Is(err, repository.ErrNotFound) {
		return h.repo.EnabledAPIProviderByCategory(ctx, fallbackCategory)
	}
	return provider, err
}

func (h *Handler) CreateProfessionalDrawRewrite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵?")
		return
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Prompt == "" {
		httpx.Error(w, http.StatusBadRequest, "鐠囩柉绶崗銉╂付鐟曚焦榧庨懝鑼畱閹绘劗銇氱拠?")
		return
	}

	provider, err := h.professionalDrawProvider(r.Context(), "rewriteProviderId")
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusBadRequest, "鐠囧嘲鍘涢崷銊ユ倵閸欓绗撴稉姘辩帛閸ュ彞鑵戦柅澶嬪楠炶泛鎯庨悽銊﹂紟閼瑰弶甯撮崣?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if provider.Provider != "openai" {
		httpx.Error(w, http.StatusBadRequest, "閹绘劗銇氱拠宥嗛紟閼硅尙娲伴崜宥勭矌閺€顖涘瘮 OpenAI 閸忕厧顔愰幒銉ュ經")
		return
	}
	if strings.TrimSpace(provider.BaseURL) == "" || strings.TrimSpace(provider.APIKey) == "" || strings.TrimSpace(provider.Model) == "" {
		httpx.Error(w, http.StatusBadRequest, "濞戯箒澹婇幒銉ュ經 Base URL閵嗕竸PI Key 閸滃本膩閸ㄥ鎮曠粔鏉跨箑婵?")
		return
	}

	rewritten, err := requestOpenAIChat(r.Context(), provider, professionalDrawRewritePrompt(req.Prompt), nil)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"prompt": strings.TrimSpace(rewritten)})
}

func professionalDrawRewritePrompt(prompt string) string {
	return "你是专业绘图提示词优化师。请将下面的用户绘图需求润色改写成更适合 AI 生图模型理解的中文提示词。保留用户原意和关键约束，补充主体、风格、构图、光影、材质、画面质量等绘图要素。只输出改写后的提示词，不要解释。\n\n原始需求：\n" + prompt
}

func buildDrawContextPrompt(messages []model.ConversationMessage, prompt string) string {
	var previous []string
	for _, message := range messages {
		if message.Role != "user" {
			continue
		}
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		previous = append(previous, content)
	}
	if len(previous) > 6 {
		previous = previous[len(previous)-6:]
	}
	if len(previous) == 0 {
		return prompt
	}
	var builder strings.Builder
	builder.WriteString("这是同一个生图对话的连续上下文。请理解此前需求并保持主体、风格、设定和约束的一致性；如果最新需求提出修改，请以最新需求为准。\n\n")
	builder.WriteString("历史用户需求：\n")
	for i, content := range previous {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, content))
	}
	builder.WriteString("\n最新用户需求：\n")
	builder.WriteString(prompt)
	return builder.String()
}

func (h *Handler) ListGenerations(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	jobs, err := h.repo.ListGenerations(r.Context(), claims.UserID, limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囬悽鐔稿灇閸樺棗褰舵径杈Е")
		return
	}
	httpx.JSON(w, http.StatusOK, jobs)
}

func (h *Handler) ListBalanceLogs(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := h.repo.ListBalanceLogs(r.Context(), claims.UserID, limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囨担娆擃杺閺冦儱绻旀径杈Е")
		return
	}
	httpx.JSON(w, http.StatusOK, logs)
}

func (h *Handler) ListMediaAssets(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	favoriteOnly := r.URL.Query().Get("favorite") == "1" || r.URL.Query().Get("favorite") == "true"
	assets, err := h.repo.ListMediaAssets(r.Context(), claims.UserID, favoriteOnly, limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囨刊鎺嶇秼鎼存挸銇戠拹?")
		return
	}
	httpx.JSON(w, http.StatusOK, assets)
}

func (h *Handler) FavoriteMediaAsset(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	if err := h.repo.SetFavorite(r.Context(), claims.UserID, pathTail(strings.TrimSuffix(r.URL.Path, "/favorite")), true); errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "娴ｆ粌鎼ф稉宥呯摠閸?")
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閺€鎯版娴ｆ粌鎼ф径杈Е")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) UnfavoriteMediaAsset(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	if err := h.repo.SetFavorite(r.Context(), claims.UserID, pathTail(strings.TrimSuffix(r.URL.Path, "/favorite")), false); errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "娴ｆ粌鎼ф稉宥呯摠閸?")
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閸欐牗绉烽弨鎯版婢惰精瑙?")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	conversations, err := h.repo.ListConversations(r.Context(), claims.UserID, limit)
	if err != nil {
		log.Printf("list conversations for user %s: %v", claims.UserID, err)
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囬張鈧潻鎴滅窗鐠囨繂銇戠拹?")
		return
	}
	httpx.JSON(w, http.StatusOK, conversations)
}

func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	detail, err := h.repo.ConversationByID(r.Context(), claims.UserID, pathTail(r.URL.Path))
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "娴兼俺鐦芥稉宥呯摠閸?")
		return
	}
	if err != nil {
		log.Printf("get conversation %s for user %s: %v", pathTail(r.URL.Path), claims.UserID, err)
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囨导姘崇樈婢惰精瑙?")
		return
	}
	httpx.JSON(w, http.StatusOK, detail)
}

func (h *Handler) ClearDrawConversations(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	if err := h.repo.ClearDrawConversations(r.Context(), claims.UserID); errors.Is(err, repository.ErrNotFound) {
		httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "濞撳懐鈹栫紒妯兼暰閸掓銆冩径杈Е")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) ClearConversations(w http.ResponseWriter, r *http.Request) {
	h.ClearDrawConversations(w, r)
}

func (h *Handler) AssistantChat(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var req struct {
		ConversationID string                `json:"conversationId"`
		Message        string                `json:"message"`
		Stream         bool                  `json:"stream"`
		Attachments    []assistantAttachment `json:"attachments"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵?")
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		httpx.Error(w, http.StatusBadRequest, "鐠囩柉绶崗銉︾Х閹垰鍞寸€?")
		return
	}
	provider, err := h.repo.EnabledAPIProviderByCategory(r.Context(), "general_text")
	if errors.Is(err, repository.ErrNotFound) {
		provider, err = h.repo.EnabledAPIProviderByCategory(r.Context(), "assistant_chat")
	}
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusBadRequest, "鐠囧嘲鍘涢崷銊ユ倵閸欎即鍘ょ純顔艰嫙閸氼垳鏁ら弲楦垮厴閸斺晜澧滈幒銉ュ經")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "鐠囪褰囬弲楦垮厴閸斺晜澧滈幒銉ュ經婢惰精瑙?")
		return
	}
	if provider.Provider != "openai" {
		httpx.Error(w, http.StatusBadRequest, "閺呴缚鍏橀崝鈺傚閻╊喖澧犳禒鍛暜閹?OpenAI 閸忕厧顔愰幒銉ュ經")
		return
	}
	if strings.TrimSpace(provider.BaseURL) == "" || strings.TrimSpace(provider.APIKey) == "" || strings.TrimSpace(provider.Model) == "" {
		httpx.Error(w, http.StatusBadRequest, "閺呴缚鍏橀崝鈺傚閹恒儱褰?Base URL閵嗕竸PI Key 閸滃本膩閸ㄥ鎮曠粔鏉跨箑婵?")
		return
	}
	if req.Stream {
		h.streamAssistantChat(w, r, claims.UserID, provider, strings.TrimSpace(req.ConversationID), req.Message, req.Attachments)
		return
	}
	answer, err := requestOpenAIChat(r.Context(), provider, req.Message, req.Attachments)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	attachmentMeta, _ := json.Marshal(assistantAttachmentMeta(req.Attachments))
	result, err := h.repo.AddAssistantChatMessage(r.Context(), claims.UserID, strings.TrimSpace(req.ConversationID), req.Message, answer, provider.Model, attachmentMeta)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "閸斺晜澧滄导姘崇樈娑撳秴鐡ㄩ崷?")
		return
	}
	if err != nil {
		log.Printf("save assistant conversation for user %s: %v", claims.UserID, err)
		httpx.Error(w, http.StatusInternalServerError, "娣囨繂鐡ㄩ崝鈺傚娴兼俺鐦芥径杈Е")
		return
	}
	_ = h.repo.RecordTaskLog(r.Context(), nil, &claims.UserID, "assistant.chat", "succeeded", "閺呴缚鍏橀崝鈺傚閸ョ偛顦茬€瑰本鍨?", nil)
	httpx.JSON(w, http.StatusCreated, result)
}

func (h *Handler) streamAssistantChat(w http.ResponseWriter, r *http.Request, userID string, provider model.APIProvider, conversationID, prompt string, attachments []assistantAttachment) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, "瑜版挸澧犻張宥呭娑撳秵鏁幐浣圭ウ瀵繗绶崙?")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	var full strings.Builder
	writeSSE := func(event string, payload any) {
		data, _ := json.Marshal(payload)
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}

	err := requestOpenAIChatStream(r.Context(), provider, prompt, attachments, func(delta string) {
		full.WriteString(delta)
		writeSSE("delta", map[string]string{"content": delta})
	})
	if err != nil {
		writeSSE("error", map[string]string{"error": err.Error()})
		return
	}
	attachmentMeta, _ := json.Marshal(assistantAttachmentMeta(attachments))
	result, err := h.repo.AddAssistantChatMessage(r.Context(), userID, conversationID, prompt, full.String(), provider.Model, attachmentMeta)
	if errors.Is(err, repository.ErrNotFound) {
		writeSSE("error", map[string]string{"error": "鍔╂墜浼氳瘽涓嶅瓨鍦?"})
		return
	}
	if err != nil {
		log.Printf("save assistant conversation for user %s: %v", userID, err)
		writeSSE("error", map[string]string{"error": "娣囨繂鐡ㄩ崝鈺傚娴兼俺鐦芥径杈Е"})
		return
	}
	_ = h.repo.RecordTaskLog(r.Context(), nil, &userID, "assistant.chat", "succeeded", "閺呴缚鍏橀崝鈺傚濞翠礁绱￠崶鐐差槻鐎瑰本鍨?", nil)
	writeSSE("done", result)
}

func (h *Handler) runProfessionalDrawJob(ctx context.Context, provider model.APIProvider, conversationID, jobID, prompt string, params json.RawMessage, sourceImageURL string) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := h.repo.MarkProfessionalDrawRunning(ctx, jobID, conversationID); err != nil {
		log.Printf("mark professional draw running %s: %v", jobID, err)
	}
	imageURL, err := requestOpenAIImageWithSource(ctx, provider, prompt, params, sourceImageURL)
	if err != nil {
		h.failProfessionalDrawJob(ctx, conversationID, jobID, err.Error())
		return
	}
	if err := h.repo.CompleteProfessionalDrawJob(ctx, jobID, conversationID, imageURL, provider.Model); err != nil {
		log.Printf("complete professional draw job %s: %v", jobID, err)
	}
}

func (h *Handler) failProfessionalDrawJob(ctx context.Context, conversationID, jobID, message string) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := h.repo.FailProfessionalDrawJob(ctx, jobID, conversationID, message); err != nil {
		log.Printf("fail professional draw job %s: %v", jobID, err)
	}
}

func latestDrawAssetURL(messages []model.ConversationMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message.Role != "assistant" || len(message.Meta) == 0 {
			continue
		}
		var meta struct {
			AssetURL string `json:"assetUrl"`
			Status   string `json:"status"`
		}
		if err := json.Unmarshal(message.Meta, &meta); err != nil {
			continue
		}
		if strings.TrimSpace(meta.Status) != "" && meta.Status != "succeeded" {
			continue
		}
		if assetURL := strings.TrimSpace(meta.AssetURL); assetURL != "" {
			return assetURL
		}
	}
	return ""
}

func requestOpenAIImageWithSource(ctx context.Context, provider model.APIProvider, prompt string, params json.RawMessage, sourceImageURL string) (string, error) {
	if strings.TrimSpace(sourceImageURL) == "" {
		return requestOpenAIImage(ctx, provider, prompt, params)
	}
	return requestOpenAIImageEdit(ctx, provider, prompt, params, sourceImageURL)
}

func requestOpenAIImage(ctx context.Context, provider model.APIProvider, prompt string, params json.RawMessage) (string, error) {
	if strings.TrimSpace(provider.BaseURL) == "" {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛 Base URL 鏈厤缃?")
	}
	if strings.TrimSpace(provider.APIKey) == "" {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛 API Key 鏈厤缃?")
	}
	endpoint, err := openAIImagesEndpoint(provider.BaseURL)
	if err != nil {
		return "", err
	}
	size := "1024x1024"
	var drawParams struct {
		Ratio string `json:"ratio"`
	}
	_ = json.Unmarshal(params, &drawParams)
	switch drawParams.Ratio {
	case "16:9":
		size = "1536x1024"
	case "9:16":
		size = "1024x1536"
	case "4:3":
		size = "1536x1024"
	case "3:4":
		size = "1024x1536"
	}
	body, _ := json.Marshal(map[string]any{
		"model":           provider.Model,
		"prompt":          prompt,
		"n":               1,
		"size":            size,
		"quality":         "medium",
		"response_format": "b64_json",
	})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛鍦板潃涓嶆纭?")
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(provider.APIKey))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("专业绘图接口请求失败：%w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(respBody))
		if message == "" {
			message = resp.Status
		}
		return "", fmt.Errorf("专业绘图接口返回 %d：%s", resp.StatusCode, message)
	}
	var payload struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛鍝嶅簲鏍煎紡涓嶆纭?")
	}
	if len(payload.Data) == 0 {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛鏈繑鍥炲浘鐗?")
	}
	if strings.TrimSpace(payload.Data[0].B64JSON) != "" {
		data := strings.TrimSpace(payload.Data[0].B64JSON)
		if _, err := base64.StdEncoding.DecodeString(data); err != nil {
			return "", errors.New("娑撴挷绗熺紒妯兼暰閹恒儱褰涙潻鏂挎礀閻?base64 閸ュ墽澧栭弮鐘虫櫏")
		}
		return "data:image/png;base64," + data, nil
	}
	if strings.TrimSpace(payload.Data[0].URL) != "" {
		return strings.TrimSpace(payload.Data[0].URL), nil
	}
	return "", errors.New("娑撴挷绗熺紒妯兼暰閹恒儱褰涢張顏囩箲閸ョ偛娴橀悧鍥ф勾閸р偓")
}

func requestOpenAIImageEdit(ctx context.Context, provider model.APIProvider, prompt string, params json.RawMessage, sourceImageURL string) (string, error) {
	if strings.TrimSpace(provider.BaseURL) == "" {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛 Base URL 鏈厤缃?")
	}
	if strings.TrimSpace(provider.APIKey) == "" {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛 API Key 鏈厤缃?")
	}
	endpoint, err := openAIImageEditsEndpoint(provider.BaseURL)
	if err != nil {
		return "", err
	}
	size := "1024x1024"
	var drawParams struct {
		Ratio string `json:"ratio"`
	}
	_ = json.Unmarshal(params, &drawParams)
	switch drawParams.Ratio {
	case "16:9":
		size = "1536x1024"
	case "9:16":
		size = "1024x1536"
	case "4:3":
		size = "1536x1024"
	case "3:4":
		size = "1024x1536"
	}
	body, _ := json.Marshal(map[string]any{
		"model":   provider.Model,
		"prompt":  prompt,
		"n":       1,
		"size":    size,
		"quality": "medium",
		"images": []map[string]string{
			{"image_url": strings.TrimSpace(sourceImageURL)},
		},
		"response_format": "b64_json",
	})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛鍦板潃涓嶆纭?")
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(provider.APIKey))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("专业绘图接口请求失败：%w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(respBody))
		if message == "" {
			message = resp.Status
		}
		return "", fmt.Errorf("专业绘图接口返回 %d：%s", resp.StatusCode, message)
	}
	var payload struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛鍝嶅簲鏍煎紡涓嶆纭?")
	}
	if len(payload.Data) == 0 {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛鏈繑鍥炲浘鐗?")
	}
	if strings.TrimSpace(payload.Data[0].B64JSON) != "" {
		data := strings.TrimSpace(payload.Data[0].B64JSON)
		if _, err := base64.StdEncoding.DecodeString(data); err != nil {
			return "", errors.New("娑撴挷绗熺紒妯兼暰閹恒儱褰涙潻鏂挎礀閻?base64 閸ュ墽澧栭弮鐘虫櫏")
		}
		return "data:image/png;base64," + data, nil
	}
	if strings.TrimSpace(payload.Data[0].URL) != "" {
		return strings.TrimSpace(payload.Data[0].URL), nil
	}
	return "", errors.New("娑撴挷绗熺紒妯兼暰閹恒儱褰涢張顏囩箲閸ョ偛娴橀悧鍥ф勾閸р偓")
}

type assistantAttachment struct {
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	DataURL  string `json:"dataUrl"`
	Text     string `json:"text"`
	Size     int    `json:"size"`
}

func assistantAttachmentMeta(attachments []assistantAttachment) []map[string]any {
	meta := make([]map[string]any, 0, len(attachments))
	for _, item := range attachments {
		meta = append(meta, map[string]any{
			"name":     item.Name,
			"mimeType": item.MimeType,
			"size":     item.Size,
			"hasText":  strings.TrimSpace(item.Text) != "",
		})
	}
	return meta
}

func openAIChatMessages(prompt string, attachments []assistantAttachment) []map[string]any {
	content := []map[string]any{{"type": "text", "text": prompt}}
	for _, item := range attachments {
		if strings.HasPrefix(item.MimeType, "image/") && strings.TrimSpace(item.DataURL) != "" {
			content = append(content, map[string]any{
				"type": "image_url",
				"image_url": map[string]string{
					"url": item.DataURL,
				},
			})
			continue
		}
		if strings.TrimSpace(item.Text) != "" {
			content = append(content, map[string]any{
				"type": "text",
				"text": "闂勫嫪娆?" + item.Name + "閿涙瓡n" + item.Text,
			})
		}
	}
	return []map[string]any{
		{"role": "system", "content": "浣犳槸鎽樻槦AI鐨勬櫤鑳藉姪鎵嬶紝鍥炵瓟瑕佹竻鏅般€佸弸濂姐€佸彲鎵ц銆傛敮鎸佽В鏋愮敤鎴蜂笂浼犵殑鍥剧墖鎴栨枃鏈檮浠躲€傝浣跨敤 Markdown 缁勭粐澶嶆潅鍥炵瓟銆?"},
		{"role": "user", "content": content},
	}
}

func requestOpenAIChat(ctx context.Context, provider model.APIProvider, prompt string, attachments []assistantAttachment) (string, error) {
	endpoint, err := openAIChatEndpoint(provider.BaseURL)
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]any{
		"model":    provider.Model,
		"messages": openAIChatMessages(prompt, attachments),
	})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", errors.New("鏅鸿兘鍔╂墜鎺ュ彛鍦板潃涓嶆纭?")
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(provider.APIKey))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("智能助手接口请求失败：%w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(respBody))
		if message == "" {
			message = resp.Status
		}
		return "", fmt.Errorf("智能助手接口返回 %d：%s", resp.StatusCode, message)
	}
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return "", errors.New("鏅鸿兘鍔╂墜鎺ュ彛鍝嶅簲鏍煎紡涓嶆纭?")
	}
	if len(payload.Choices) == 0 || strings.TrimSpace(payload.Choices[0].Message.Content) == "" {
		return "", errors.New("鏅鸿兘鍔╂墜鎺ュ彛鏈繑鍥炲唴瀹?")
	}
	return strings.TrimSpace(payload.Choices[0].Message.Content), nil
}

func requestOpenAIChatStream(ctx context.Context, provider model.APIProvider, prompt string, attachments []assistantAttachment, onDelta func(string)) error {
	endpoint, err := openAIChatEndpoint(provider.BaseURL)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{
		"model":    provider.Model,
		"messages": openAIChatMessages(prompt, attachments),
		"stream":   true,
	})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return errors.New("鏅鸿兘鍔╂墜鎺ュ彛鍦板潃涓嶆纭?")
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(provider.APIKey))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("智能助手接口请求失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		message := strings.TrimSpace(string(respBody))
		if message == "" {
			message = resp.Status
		}
		return fmt.Errorf("智能助手接口返回 %d：%s", resp.StatusCode, message)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				onDelta(choice.Delta.Content)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取智能助手流失败：%w", err)
	}
	return nil
}

func openAIImagesEndpoint(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛 Base URL 涓嶆纭?")
	}
	path := strings.TrimRight(parsed.Path, "/")
	if strings.HasSuffix(path, "/images/generations") {
		return parsed.String(), nil
	}
	if index := strings.Index(path, "/v1"); index >= 0 {
		parsed.Path = path[:index] + "/v1/images/generations"
		parsed.RawQuery = ""
		return parsed.String(), nil
	}
	if strings.HasSuffix(path, "/v1") {
		parsed.Path = path + "/images/generations"
	} else {
		parsed.Path = path + "/v1/images/generations"
	}
	parsed.RawQuery = ""
	return parsed.String(), nil
}

func openAIImageEditsEndpoint(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("涓撲笟缁樺浘鎺ュ彛 Base URL 涓嶆纭?")
	}
	path := strings.TrimRight(parsed.Path, "/")
	if strings.HasSuffix(path, "/images/edits") {
		return parsed.String(), nil
	}
	if index := strings.Index(path, "/v1"); index >= 0 {
		parsed.Path = path[:index] + "/v1/images/edits"
		parsed.RawQuery = ""
		return parsed.String(), nil
	}
	if strings.HasSuffix(path, "/v1") {
		parsed.Path = path + "/images/edits"
	} else {
		parsed.Path = path + "/v1/images/edits"
	}
	parsed.RawQuery = ""
	return parsed.String(), nil
}

func openAIChatEndpoint(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("鏅鸿兘鍔╂墜鎺ュ彛 Base URL 涓嶆纭?")
	}
	path := strings.TrimRight(parsed.Path, "/")
	if strings.HasSuffix(path, "/chat/completions") {
		return parsed.String(), nil
	}
	if index := strings.Index(path, "/v1"); index >= 0 {
		parsed.Path = path[:index] + "/v1/chat/completions"
		parsed.RawQuery = ""
		return parsed.String(), nil
	}
	parsed.Path = path + "/v1/chat/completions"
	parsed.RawQuery = ""
	return parsed.String(), nil
}

func (h *Handler) AffiliateDashboard(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	dashboard, err := h.repo.AffiliateDashboard(r.Context(), claims.UserID)
	if err != nil {
		log.Printf("affiliate dashboard failed user_id=%s: %v", claims.UserID, err)
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囬幒銊ョ畭閺佺増宓佹径杈Е")
		return
	}
	httpx.JSON(w, http.StatusOK, dashboard)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nickname  string `json:"nickname"`
		AvatarURL string `json:"avatarUrl"`
		Signature string `json:"signature"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵顕嗙窗"+err.Error())
		return
	}
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.AvatarURL = strings.TrimSpace(req.AvatarURL)
	req.Signature = strings.TrimSpace(req.Signature)
	if req.Nickname == "" {
		httpx.Error(w, http.StatusBadRequest, "閻劍鍩涢崥宥囆炴稉宥堝厴娑撹櫣鈹?")
		return
	}
	if len([]rune(req.Nickname)) > 12 {
		httpx.Error(w, http.StatusBadRequest, "閻劍鍩涢崥宥囆為張鈧径?12 娑擃亜鐡?")
		return
	}
	if len([]rune(req.Signature)) > 128 {
		httpx.Error(w, http.StatusBadRequest, "娑擃亝鈧咁劮閸氬秵娓舵径?128 娑擃亜鐡?")
		return
	}
	claims, _ := middleware.ClaimsFromContext(r.Context())
	user, err := h.repo.UpdateUserProfile(r.Context(), claims.UserID, req.Nickname, req.AvatarURL, req.Signature)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "閻劍鍩涙稉宥呯摠閸?")
		return
	}
	if err != nil {
		log.Printf("update profile failed user_id=%s: %v", claims.UserID, err)
		httpx.Error(w, http.StatusInternalServerError, "娣囨繂鐡ㄧ拹锔藉煕娣団剝浼呮径杈Е")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵顕嗙窗"+err.Error())
		return
	}
	if strings.TrimSpace(req.CurrentPassword) == "" {
		httpx.Error(w, http.StatusBadRequest, "鐠囩柉绶崗銉ョ秼閸撳秴鐦戦惍?")
		return
	}
	if err := validatePassword(req.NewPassword); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.CurrentPassword == req.NewPassword {
		httpx.Error(w, http.StatusBadRequest, "閺傛澘鐦戦惍浣风瑝閼宠棄鎷拌ぐ鎾冲鐎靛棛鐖滈惄绋挎倱")
		return
	}
	claims, _ := middleware.ClaimsFromContext(r.Context())
	_, passwordHash, err := h.repo.UserPasswordHashByID(r.Context(), claims.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "閻劍鍩涙稉宥呯摠閸?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "鐠囪褰囩拹锔藉煕娣団剝浼呮径杈Е")
		return
	}
	if !security.CheckPassword(passwordHash, req.CurrentPassword) {
		httpx.Error(w, http.StatusBadRequest, "瑜版挸澧犵€靛棛鐖滄稉宥嗩劀绾?")
		return
	}
	hash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "鐎靛棛鐖滄径鍕倞婢惰精瑙?")
		return
	}
	if err := h.repo.UpdatePasswordByID(r.Context(), claims.UserID, hash); errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "閻劍鍩涙稉宥呯摠閸?")
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "娣囶喗鏁肩€靛棛鐖滄径杈Е")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email           string `json:"email"`
		Code            string `json:"code"`
		CurrentPassword string `json:"currentPassword"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵顕嗙窗"+err.Error())
		return
	}
	req.Email = normalizeEmail(req.Email)
	req.Code = strings.TrimSpace(req.Code)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		httpx.Error(w, http.StatusBadRequest, "闁喚顔堥弽鐓庣础娑撳秵顒滅涵?")
		return
	}
	if req.Code == "" {
		httpx.Error(w, http.StatusBadRequest, "鐠囩柉绶崗銉╁仏缁犻亶鐛欑拠浣虹垳")
		return
	}
	if strings.TrimSpace(req.CurrentPassword) == "" {
		httpx.Error(w, http.StatusBadRequest, "鐠囩柉绶崗銉ョ秼閸撳秴鐦戦惍?")
		return
	}
	claims, _ := middleware.ClaimsFromContext(r.Context())
	user, passwordHash, err := h.repo.UserPasswordHashByID(r.Context(), claims.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "閻劍鍩涙稉宥呯摠閸?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "鐠囪褰囩拹锔藉煕娣団剝浼呮径杈Е")
		return
	}
	if user.Email == req.Email {
		httpx.Error(w, http.StatusBadRequest, "閺備即鍋栫粻鍙樼瑝閼宠棄鎷拌ぐ鎾冲闁喚顔堥惄绋挎倱")
		return
	}
	if !security.CheckPassword(passwordHash, req.CurrentPassword) {
		httpx.Error(w, http.StatusBadRequest, "瑜版挸澧犵€靛棛鐖滄稉宥嗩劀绾?")
		return
	}
	if existing, _, err := h.repo.UserByEmail(r.Context(), req.Email); err == nil && existing.ID != user.ID {
		httpx.Error(w, http.StatusConflict, "闁喚顔堝鑼额潶閸楃姷鏁?")
		return
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusInternalServerError, "閺嶏繝鐛欓柇顔绢唸婢惰精瑙?")
		return
	}
	ok, err := h.repo.ConsumeVerificationCode(r.Context(), req.Email, "change_email", security.HashCode(req.Email, "change_email", req.Code))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "妤犲矁鐦夐惍浣圭墡妤犲苯銇戠拹?")
		return
	}
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "妤犲矁鐦夐惍浣规￥閺佸牊鍨ㄥ鑼剁箖閺?")
		return
	}
	updated, err := h.repo.UpdateUserEmail(r.Context(), claims.UserID, req.Email)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "閻劍鍩涙稉宥呯摠閸?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusConflict, "闁喚顔堝鑼额潶閸楃姷鏁?")
		return
	}
	httpx.JSON(w, http.StatusOK, updated)
}

func (h *Handler) RedeemCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵顕嗙窗"+err.Error())
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		httpx.Error(w, http.StatusBadRequest, "鐠囩柉绶崗銉ュ幀閹广垻鐖?")
		return
	}
	claims, _ := middleware.ClaimsFromContext(r.Context())
	code, user, err := h.repo.RedeemInviteCode(r.Context(), claims.UserID, req.Code)
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "閸忔垶宕查惍浣风瑝鐎涙ê婀?")
		return
	}
	if errors.Is(err, repository.ErrAlreadyUsed) {
		httpx.Error(w, http.StatusBadRequest, "閸忔垶宕查惍浣稿嚒鐞氼偂濞囬悽?")
		return
	}
	if errors.Is(err, repository.ErrExpired) {
		httpx.Error(w, http.StatusBadRequest, "閸忔垶宕查惍浣稿嚒鏉╁洦婀?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閸忔垶宕叉径杈Е")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"code": code, "user": user})
}

func (h *Handler) CreateAffiliateWithdrawal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount string `json:"amount"`
		Note   string `json:"note"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "鐠囬攱鐪伴弽鐓庣础娑撳秵顒滅涵顕嗙窗"+err.Error())
		return
	}
	req.Amount = strings.TrimSpace(req.Amount)
	if req.Amount == "" {
		httpx.Error(w, http.StatusBadRequest, "鐠囩柉绶崗銉﹀絹閻滀即鍣炬０?")
		return
	}
	claims, _ := middleware.ClaimsFromContext(r.Context())
	withdrawal, err := h.repo.CreateAffiliateWithdrawal(r.Context(), claims.UserID, req.Amount, strings.TrimSpace(req.Note))
	if errors.Is(err, repository.ErrInsufficientBalance) {
		httpx.Error(w, http.StatusBadRequest, "閸欘垱褰侀悳棰佺稇妫版繀绗夌搾?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閹绘劒姘﹂幓鎰箛閻㈠疇顕径杈Е")
		return
	}
	httpx.JSON(w, http.StatusCreated, withdrawal)
}

func (h *Handler) RecordAffiliateVisit(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.RecordAffiliateVisit(r.Context(), pathTail(r.URL.Path)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "鐠佹澘缍嶉幒銊ョ畭鐠佸潡妫舵径杈Е")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func priceForMembership(app model.App, level string) string {
	switch level {
	case "v2":
		return app.PriceV2
	case "v1":
		return app.PriceV1
	default:
		return app.PriceFree
	}
}

func (h *Handler) GetGeneration(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	job, err := h.repo.GenerationByID(r.Context(), claims.UserID, pathTail(r.URL.Path))
	if errors.Is(err, repository.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "閻㈢喐鍨氭禒璇插娑撳秴鐡ㄩ崷?")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "閼惧嘲褰囬悽鐔稿灇娴犺濮熸径杈Е")
		return
	}
	httpx.JSON(w, http.StatusOK, job)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("瀵嗙爜鑷冲皯闇€瑕?8 浣?")
	}
	return nil
}

func pathTail(path string) string {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func clientIP(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); value != "" {
		return strings.TrimSpace(strings.Split(value, ",")[0])
	}
	if value := strings.TrimSpace(r.Header.Get("X-Real-IP")); value != "" {
		return value
	}
	return r.RemoteAddr
}
