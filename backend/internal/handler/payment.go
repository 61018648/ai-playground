package handler

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"image-ai/backend/internal/httpx"
	"image-ai/backend/internal/middleware"
	"image-ai/backend/internal/model"
	"image-ai/backend/internal/repository"
)

type paymentSetting struct {
	Enabled         bool          `json:"enabled"`
	Provider        string        `json:"provider"`
	GatewayURL      string        `json:"gatewayUrl"`
	PID             string        `json:"pid"`
	Key             string        `json:"key"`
	NotifyURL       string        `json:"notifyUrl"`
	ReturnURL       string        `json:"returnUrl"`
	SignType        string        `json:"signType"`
	Channels        []string      `json:"channels"`
	CreditPlans     []paymentPlan `json:"creditPlans"`
	MembershipPlans []paymentPlan `json:"membershipPlans"`
}

type paymentPlan struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	OrderType       string `json:"orderType"`
	Amount          string `json:"amount"`
	Credits         int    `json:"credits"`
	MembershipLevel string `json:"membershipLevel"`
	Desc            string `json:"desc"`
	Period          string `json:"period"`
}

type seoSetting struct {
	SiteName string `json:"siteName"`
	Title    string `json:"title"`
}

var defaultCreditPlans = []paymentPlan{
	{Code: "credits_basic", Name: "基础积分", OrderType: "credits", Amount: "30.00", Credits: 3000, Desc: "3000 积分"},
	{Code: "credits_plus", Name: "高级积分", OrderType: "credits", Amount: "50.00", Credits: 5000, Desc: "5000 积分"},
	{Code: "credits_super", Name: "超级积分", OrderType: "credits", Amount: "100.00", Credits: 12000, Desc: "12000 积分"},
}

var defaultMembershipPlans = []paymentPlan{
	{Code: "vip_v1", Name: "V1会员", OrderType: "membership", Amount: "39.00", MembershipLevel: "v1", Desc: "开通 V1 会员", Period: "30天"},
	{Code: "vip_v2", Name: "V2会员", OrderType: "membership", Amount: "69.00", MembershipLevel: "v2", Desc: "开通 V2 会员", Period: "30天"},
	{Code: "vip_v2_year", Name: "V2年费会员", OrderType: "membership", Amount: "199.00", MembershipLevel: "v2", Desc: "开通 V2 年费会员", Period: "365天"},
}

func (h *Handler) PaymentPlans(w http.ResponseWriter, r *http.Request) {
	credits, membership := h.paymentPlans(r)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"credits":    credits,
		"membership": membership,
	})
}

func (h *Handler) CreatePaymentOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanCode string `json:"planCode"`
		PayType  string `json:"payType"`
	}
	if err := httpx.DecodeJSONLoose(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求格式不正确："+err.Error())
		return
	}
	plan, ok := h.findPaymentPlan(r, strings.TrimSpace(req.PlanCode))
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "套餐不存在")
		return
	}
	setting, err := h.paymentSetting(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	claims, _ := middleware.ClaimsFromContext(r.Context())
	tradeNo := "PAY" + time.Now().Format("20060102150405") + strings.ToUpper(randomString(6))
	name := h.paymentOrderName(r, plan.Name)
	notifyURL := setting.NotifyURL
	if notifyURL == "" {
		notifyURL = requestBaseURL(r) + "/api/v1/pay/epay/notify"
	}
	returnURL := setting.ReturnURL
	if returnURL == "" {
		returnURL = requestBaseURL(r) + "/profile"
	}
	params := map[string]string{
		"pid":          setting.PID,
		"type":         normalizePayType(req.PayType),
		"out_trade_no": tradeNo,
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"name":         name,
		"money":        plan.Amount,
		"param":        claims.UserID,
		"clientip":     clientIP(r),
		"device":       "pc",
	}
	if !allowedPayType(params["type"], setting.Channels) {
		httpx.Error(w, http.StatusBadRequest, "当前支付渠道未开启")
		return
	}
	params["sign"] = epaySign(params, setting.Key)
	params["sign_type"] = "MD5"
	payURL := epaySubmitURL(setting.GatewayURL) + "?" + encodeQuery(params)

	order, err := h.repo.CreatePaymentOrder(r.Context(), model.PaymentOrder{
		TradeNo:         tradeNo,
		UserID:          claims.UserID,
		Provider:        "epay",
		OrderType:       plan.OrderType,
		PlanCode:        plan.Code,
		PlanName:        plan.Name,
		Amount:          plan.Amount,
		Credits:         plan.Credits,
		MembershipLevel: plan.MembershipLevel,
		Status:          "pending",
		PayURL:          payURL,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "创建支付订单失败")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": order, "payUrl": payURL})
}

func (h *Handler) paymentOrderName(r *http.Request, planName string) string {
	siteName := h.siteName(r)
	if siteName == "" {
		return planName
	}
	return siteName + "-" + planName
}

func (h *Handler) siteName(r *http.Request) string {
	setting, err := h.repo.SiteSettingByKey(r.Context(), "seo")
	if err != nil {
		return ""
	}
	var value seoSetting
	if err := json.Unmarshal(setting.Value, &value); err != nil {
		return ""
	}
	if name := strings.TrimSpace(value.SiteName); name != "" {
		return name
	}
	title := strings.TrimSpace(value.Title)
	if title == "" {
		return ""
	}
	if before, _, ok := strings.Cut(title, " - "); ok {
		return strings.TrimSpace(before)
	}
	return title
}

func (h *Handler) EPayNotify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "fail", http.StatusBadRequest)
		return
	}
	setting, err := h.paymentSetting(r)
	if err != nil {
		http.Error(w, "fail", http.StatusBadRequest)
		return
	}
	params := map[string]string{}
	for key, values := range r.Form {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	if !epayVerify(params, setting.Key) {
		http.Error(w, "fail", http.StatusBadRequest)
		return
	}
	if params["trade_status"] != "TRADE_SUCCESS" {
		_, _ = w.Write([]byte("success"))
		return
	}
	if _, _, err := h.repo.CompletePaymentOrder(r.Context(), params["out_trade_no"]); err != nil {
		http.Error(w, "fail", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte("success"))
}

func (h *Handler) EPayReturn(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/profile", http.StatusFound)
}

func (h *Handler) paymentSetting(r *http.Request) (paymentSetting, error) {
	setting, err := h.repo.SiteSettingByKey(r.Context(), "payment")
	if errors.Is(err, repository.ErrNotFound) {
		return paymentSetting{}, errors.New("支付配置不存在")
	}
	if err != nil {
		return paymentSetting{}, errors.New("读取支付配置失败")
	}
	var value paymentSetting
	if err := json.Unmarshal(setting.Value, &value); err != nil {
		return paymentSetting{}, errors.New("支付配置格式不正确")
	}
	if !value.Enabled {
		return paymentSetting{}, errors.New("支付功能未启用")
	}
	if value.Provider != "" && value.Provider != "epay" {
		return paymentSetting{}, errors.New("当前仅支持易支付")
	}
	if strings.TrimSpace(value.GatewayURL) == "" || strings.TrimSpace(value.PID) == "" || strings.TrimSpace(value.Key) == "" {
		return paymentSetting{}, errors.New("易支付网关、商户 PID 和密钥必填")
	}
	if len(value.Channels) == 0 {
		value.Channels = []string{"alipay"}
	}
	return value, nil
}

func (h *Handler) paymentPlans(r *http.Request) ([]paymentPlan, []paymentPlan) {
	setting, err := h.repo.SiteSettingByKey(r.Context(), "payment")
	if err != nil {
		return defaultCreditPlans, defaultMembershipPlans
	}
	var value paymentSetting
	if err := json.Unmarshal(setting.Value, &value); err != nil {
		return defaultCreditPlans, defaultMembershipPlans
	}
	credits := normalizePaymentPlans(value.CreditPlans, "credits")
	membership := normalizePaymentPlans(value.MembershipPlans, "membership")
	if len(credits) == 0 {
		credits = defaultCreditPlans
	}
	if len(membership) == 0 {
		membership = defaultMembershipPlans
	}
	return credits, membership
}

func normalizePaymentPlans(plans []paymentPlan, orderType string) []paymentPlan {
	out := make([]paymentPlan, 0, len(plans))
	for _, plan := range plans {
		plan.Code = strings.TrimSpace(plan.Code)
		plan.Name = strings.TrimSpace(plan.Name)
		plan.Amount = strings.TrimSpace(plan.Amount)
		plan.Desc = strings.TrimSpace(plan.Desc)
		plan.Period = strings.TrimSpace(plan.Period)
		plan.MembershipLevel = strings.TrimSpace(plan.MembershipLevel)
		if plan.Code == "" || plan.Name == "" || plan.Amount == "" {
			continue
		}
		plan.OrderType = orderType
		if orderType == "credits" {
			plan.MembershipLevel = ""
			if plan.Credits <= 0 {
				continue
			}
		}
		if orderType == "membership" {
			plan.Credits = 0
			if plan.MembershipLevel == "" {
				plan.MembershipLevel = "v1"
			}
		}
		out = append(out, plan)
	}
	return out
}

func (h *Handler) findPaymentPlan(r *http.Request, code string) (paymentPlan, bool) {
	credits, membership := h.paymentPlans(r)
	for _, plan := range append(credits, membership...) {
		if plan.Code == code {
			return plan, true
		}
	}
	return paymentPlan{}, false
}

func normalizePayType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "alipay"
	}
	if value != "alipay" && value != "wxpay" && value != "qqpay" {
		return "alipay"
	}
	return value
}

func allowedPayType(value string, channels []string) bool {
	value = normalizePayType(value)
	for _, channel := range channels {
		if channel == value {
			return true
		}
	}
	return false
}

func epaySubmitURL(gateway string) string {
	gateway = strings.TrimRight(strings.TrimSpace(gateway), "/")
	if strings.HasSuffix(gateway, ".php") {
		return gateway
	}
	return gateway + "/submit.php"
}

func epaySign(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	sum := md5.Sum([]byte(strings.Join(parts, "&") + key))
	return hex.EncodeToString(sum[:])
}

func epayVerify(params map[string]string, key string) bool {
	return strings.EqualFold(params["sign"], epaySign(params, key))
}

func encodeQuery(params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return values.Encode()
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func randomString(length int) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var builder strings.Builder
	for builder.Len() < length {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			builder.WriteByte(alphabet[time.Now().UnixNano()%int64(len(alphabet))])
			continue
		}
		builder.WriteByte(alphabet[n.Int64()])
	}
	return builder.String()
}
