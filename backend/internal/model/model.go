package model

import (
	"encoding/json"
	"time"
)

type User struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	Nickname        string    `json:"nickname"`
	AvatarURL       string    `json:"avatarUrl"`
	Signature       string    `json:"signature"`
	Role            string    `json:"role"`
	Status          string    `json:"status"`
	MembershipLevel string    `json:"membershipLevel"`
	Balance         string    `json:"balance"`
	Credits         int       `json:"credits"`
	CreatedAt       time.Time `json:"createdAt"`
}

type App struct {
	ID             string          `json:"id"`
	ProviderID     *string         `json:"providerId,omitempty"`
	ProviderName   string          `json:"providerName,omitempty"`
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
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type GenerationJob struct {
	ID             string          `json:"id"`
	UserID         string          `json:"userId"`
	AppID          *string         `json:"appId,omitempty"`
	AppName        string          `json:"appName,omitempty"`
	Prompt         string          `json:"prompt"`
	NegativePrompt string          `json:"negativePrompt"`
	Params         json.RawMessage `json:"params"`
	Model          string          `json:"model"`
	Status         string          `json:"status"`
	Progress       int             `json:"progress"`
	ErrorMessage   string          `json:"errorMessage"`
	Seed           *int64          `json:"seed,omitempty"`
	Assets         []Asset         `json:"assets"`
	CreatedAt      time.Time       `json:"createdAt"`
	StartedAt      *time.Time      `json:"startedAt,omitempty"`
	FinishedAt     *time.Time      `json:"finishedAt,omitempty"`
}

type Asset struct {
	ID           string          `json:"id"`
	JobID        string          `json:"jobId"`
	Kind         string          `json:"kind"`
	URL          string          `json:"url"`
	ThumbnailURL string          `json:"thumbnailUrl"`
	Width        int             `json:"width"`
	Height       int             `json:"height"`
	MimeType     string          `json:"mimeType"`
	SortOrder    int             `json:"sortOrder"`
	Meta         json.RawMessage `json:"meta"`
	CreatedAt    time.Time       `json:"createdAt"`
}

type MediaAsset struct {
	Asset
	Prompt      string    `json:"prompt"`
	AppName     string    `json:"appName,omitempty"`
	Model       string    `json:"model"`
	JobStatus   string    `json:"jobStatus"`
	IsFavorite  bool      `json:"isFavorite"`
	GeneratedAt time.Time `json:"generatedAt"`
}

type AdminStats struct {
	UsersTotal       int64 `json:"usersTotal"`
	AppsTotal        int64 `json:"appsTotal"`
	GenerationsTotal int64 `json:"generationsTotal"`
	AssetsTotal      int64 `json:"assetsTotal"`
	TodayGenerations int64 `json:"todayGenerations"`
}

type AdminGenerationJob struct {
	GenerationJob
	UserEmail string `json:"userEmail"`
}

type SiteSetting struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type APIProvider struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Provider  string    `json:"provider"`
	BaseURL   string    `json:"baseUrl"`
	APIKey    string    `json:"apiKey,omitempty"`
	Model     string    `json:"model"`
	Enabled   bool      `json:"enabled"`
	SortOrder int       `json:"sortOrder"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type LoginLog struct {
	ID        string    `json:"id"`
	UserID    *string   `json:"userId,omitempty"`
	Email     string    `json:"email"`
	Success   bool      `json:"success"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type TaskLog struct {
	ID        string          `json:"id"`
	JobID     *string         `json:"jobId,omitempty"`
	UserID    *string         `json:"userId,omitempty"`
	Action    string          `json:"action"`
	Status    string          `json:"status"`
	Message   string          `json:"message"`
	Meta      json.RawMessage `json:"meta"`
	CreatedAt time.Time       `json:"createdAt"`
}

type BalanceLog struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	OperatorID    *string   `json:"operatorId,omitempty"`
	ChangeType    string    `json:"changeType"`
	Amount        string    `json:"amount"`
	BalanceBefore string    `json:"balanceBefore"`
	BalanceAfter  string    `json:"balanceAfter"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"createdAt"`
}

type PaymentOrder struct {
	ID              string     `json:"id"`
	TradeNo         string     `json:"tradeNo"`
	UserID          string     `json:"userId"`
	Provider        string     `json:"provider"`
	OrderType       string     `json:"orderType"`
	PlanCode        string     `json:"planCode"`
	PlanName        string     `json:"planName"`
	Amount          string     `json:"amount"`
	Credits         int        `json:"credits"`
	MembershipLevel string     `json:"membershipLevel"`
	Status          string     `json:"status"`
	PayURL          string     `json:"payUrl"`
	PaidAt          *time.Time `json:"paidAt,omitempty"`
	CancelledAt     *time.Time `json:"cancelledAt,omitempty"`
	ExpiresAt       time.Time  `json:"expiresAt"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type AdminPaymentOrder struct {
	PaymentOrder
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}

type AffiliateProfile struct {
	UserID         string    `json:"userId"`
	Code           string    `json:"code"`
	Level          string    `json:"level"`
	CommissionRate string    `json:"commissionRate"`
	Visits         int       `json:"visits"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type AffiliateCommission struct {
	ID               string    `json:"id"`
	ReferrerID       string    `json:"referrerId"`
	ReferredUserID   string    `json:"referredUserId"`
	ReferredEmail    string    `json:"referredEmail,omitempty"`
	PaymentOrderID   string    `json:"paymentOrderId"`
	OrderAmount      string    `json:"orderAmount"`
	ProductType      string    `json:"productType"`
	Status           string    `json:"status"`
	CommissionRate   string    `json:"commissionRate"`
	CommissionAmount string    `json:"commissionAmount"`
	CreatedAt        time.Time `json:"createdAt"`
}

type AffiliateWithdrawal struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Amount    string    `json:"amount"`
	Status    string    `json:"status"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AffiliateInviteUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Nickname  string    `json:"nickname"`
	CreatedAt time.Time `json:"createdAt"`
}

type AffiliateDashboard struct {
	Profile           AffiliateProfile      `json:"profile"`
	TotalCommission   string                `json:"totalCommission"`
	AvailableAmount   string                `json:"availableAmount"`
	WithdrawingAmount string                `json:"withdrawingAmount"`
	PaidOrderCount    int                   `json:"paidOrderCount"`
	InvitedUserCount  int                   `json:"invitedUserCount"`
	Commissions       []AffiliateCommission `json:"commissions"`
	Withdrawals       []AffiliateWithdrawal `json:"withdrawals"`
	InviteUsers       []AffiliateInviteUser `json:"inviteUsers"`
}

type AdminAffiliateProfile struct {
	AffiliateProfile
	Email             string `json:"email"`
	Nickname          string `json:"nickname"`
	TotalCommission   string `json:"totalCommission"`
	AvailableAmount   string `json:"availableAmount"`
	WithdrawingAmount string `json:"withdrawingAmount"`
	PaidOrderCount    int    `json:"paidOrderCount"`
	InvitedUserCount  int    `json:"invitedUserCount"`
}

type AdminAffiliateOverview struct {
	Profiles    []AdminAffiliateProfile `json:"profiles"`
	Commissions []AffiliateCommission   `json:"commissions"`
	Withdrawals []AffiliateWithdrawal   `json:"withdrawals"`
}

type InviteCode struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Amount      string     `json:"amount"`
	MaxUses     int        `json:"maxUses"`
	UsedCount   int        `json:"usedCount"`
	Note        string     `json:"note"`
	CreatedBy   *string    `json:"createdBy,omitempty"`
	UsedBy      *string    `json:"usedBy,omitempty"`
	UsedByEmail string     `json:"usedByEmail,omitempty"`
	UsedAt      *time.Time `json:"usedAt,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type ConversationMessage struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversationId"`
	Role           string          `json:"role"`
	Content        string          `json:"content"`
	Meta           json.RawMessage `json:"meta"`
	CreatedAt      time.Time       `json:"createdAt"`
}

type Conversation struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	AppID     *string   `json:"appId,omitempty"`
	AppName   string    `json:"appName,omitempty"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AssistantChatResult struct {
	Conversation Conversation          `json:"conversation"`
	Messages     []ConversationMessage `json:"messages"`
}

type ConversationDetail struct {
	Conversation Conversation          `json:"conversation"`
	Messages     []ConversationMessage `json:"messages"`
	Job          *GenerationJob        `json:"job,omitempty"`
}

type DrawConversationResult struct {
	ConversationID string                `json:"conversationId"`
	User           User                  `json:"user"`
	Job            GenerationJob         `json:"job"`
	Messages       []ConversationMessage `json:"messages"`
	Charged        string                `json:"charged"`
}
