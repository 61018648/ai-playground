export interface ApiEnvelope<T> {
  data?: T
  error?: string
}

export interface ApiApp {
  id: string
  providerId?: string
  providerName?: string
  code: string
  name: string
  appType: 'image' | 'text'
  category: string
  description: string
  icon: string
  iconColor: string
  coverUrl: string
  promptTemplate: string
  inputSchema: Record<string, unknown>
  outputSchema: Record<string, unknown>
  priceFree: string
  priceV1: string
  priceV2: string
  visibility: string
  status: string
  sortOrder: number
  createdAt: string
  updatedAt: string
}

export interface ApiUser {
  id: string
  email: string
  nickname: string
  avatarUrl: string
  signature: string
  role: string
  status: string
  membershipLevel: string
  balance: string
  credits: number
  createdAt: string
}

export interface ApiAsset {
  id: string
  jobId: string
  kind: string
  url: string
  thumbnailUrl: string
  width: number
  height: number
  mimeType: string
  sortOrder: number
  meta: Record<string, unknown>
  createdAt: string
}

export interface ApiMediaAsset extends ApiAsset {
  prompt: string
  appName?: string
  model: string
  jobStatus: string
  isFavorite: boolean
  generatedAt: string
}

export interface ApiGeneration {
  id: string
  userId: string
  appId?: string
  appName?: string
  prompt: string
  negativePrompt: string
  params: Record<string, unknown>
  model: string
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  progress: number
  errorMessage: string
  assets: ApiAsset[]
  createdAt: string
  startedAt?: string
  finishedAt?: string
}

export interface ApiAdminStats {
  usersTotal: number
  appsTotal: number
  generationsTotal: number
  assetsTotal: number
  todayGenerations: number
}

export interface ApiAdminGeneration extends ApiGeneration {
  userEmail: string
}

export interface ApiSiteSetting {
  key: 'seo' | 'auth' | 'smtp' | 'payment'
  value: Record<string, unknown>
  updatedAt: string
}

export interface ApiPaymentSetting {
  enabled: boolean
  provider: string
  gatewayUrl: string
  pid: string
  key: string
  notifyUrl: string
  returnUrl: string
  signType: string
  channels?: string[]
  creditPlans?: ApiPaymentPlan[]
  membershipPlans?: ApiPaymentPlan[]
}

export interface ApiProviderConfig {
  id: string
  name: string
  category: string
  provider: string
  baseUrl: string
  apiKey?: string
  model: string
  enabled: boolean
  sortOrder: number
  createdAt: string
  updatedAt: string
}

export interface ApiLoginLog {
  id: string
  userId?: string
  email: string
  success: boolean
  ip: string
  userAgent: string
  message: string
  createdAt: string
}

export interface ApiTaskLog {
  id: string
  jobId?: string
  userId?: string
  action: string
  status: string
  message: string
  meta: Record<string, unknown>
  createdAt: string
}

export interface ApiBalanceLog {
  id: string
  userId: string
  operatorId?: string
  changeType: string
  amount: string
  balanceBefore: string
  balanceAfter: string
  note: string
  createdAt: string
}

export interface ApiBalanceAdjustResult {
  log: ApiBalanceLog
  user: ApiUser
}

export interface ApiInviteCode {
  id: string
  code: string
  amount: string
  maxUses: number
  usedCount: number
  note: string
  createdBy?: string
  usedBy?: string
  usedByEmail?: string
  usedAt?: string
  expiresAt?: string
  createdAt: string
}

export interface ApiRedeemCodeResult {
  code: ApiInviteCode
  user: ApiUser
}

export interface ApiConversationMessage {
  id: string
  conversationId: string
  role: 'user' | 'assistant'
  content: string
  meta: Record<string, unknown>
  createdAt: string
}

export interface ApiConversation {
  id: string
  userId: string
  appId?: string
  appName?: string
  kind: string
  title: string
  createdAt: string
  updatedAt: string
}

export interface ApiAssistantChatResult {
  conversation: ApiConversation
  messages: ApiConversationMessage[]
}

export interface ApiConversationDetail {
  conversation: ApiConversation
  messages: ApiConversationMessage[]
  job?: ApiGeneration
}

export interface ApiDrawConversationResult {
  conversationId: string
  user: ApiUser
  job: ApiGeneration
  messages: ApiConversationMessage[]
  charged: string
}

export interface ApiPaymentPlan {
  code: string
  name: string
  orderType: 'credits' | 'membership'
  amount: string
  credits: number
  membershipLevel: string
  desc: string
  period?: string
}

export interface ApiPaymentPlans {
  credits: ApiPaymentPlan[]
  membership: ApiPaymentPlan[]
}

export interface ApiPaymentOrder {
  id: string
  tradeNo: string
  userId: string
  provider: string
  orderType: string
  planCode: string
  planName: string
  amount: string
  credits: number
  membershipLevel: string
  status: string
  payUrl: string
  paidAt?: string
  cancelledAt?: string
  expiresAt?: string
  createdAt: string
  updatedAt: string
}

export interface ApiPaymentOrderResult {
  order: ApiPaymentOrder
  payUrl: string
}

export interface ApiAdminPaymentOrder extends ApiPaymentOrder {
  email: string
  nickname: string
}

export interface ApiAffiliateProfile {
  userId: string
  code: string
  level: string
  commissionRate: string
  visits: number
  createdAt: string
  updatedAt: string
}

export interface ApiAffiliateCommission {
  id: string
  referrerId: string
  referredUserId: string
  referredEmail?: string
  paymentOrderId: string
  orderAmount: string
  productType: string
  status: string
  commissionRate: string
  commissionAmount: string
  createdAt: string
}

export interface ApiAffiliateWithdrawal {
  id: string
  userId: string
  amount: string
  status: string
  note: string
  createdAt: string
  updatedAt: string
}

export interface ApiAffiliateInviteUser {
  id: string
  email: string
  nickname: string
  createdAt: string
}

export interface ApiAffiliateDashboard {
  profile: ApiAffiliateProfile
  totalCommission: string
  availableAmount: string
  withdrawingAmount: string
  paidOrderCount: number
  invitedUserCount: number
  commissions: ApiAffiliateCommission[]
  withdrawals: ApiAffiliateWithdrawal[]
  inviteUsers: ApiAffiliateInviteUser[]
}

export interface ApiAdminAffiliateProfile extends ApiAffiliateProfile {
  email: string
  nickname: string
  totalCommission: string
  availableAmount: string
  withdrawingAmount: string
  paidOrderCount: number
  invitedUserCount: number
}

export interface ApiAdminAffiliateOverview {
  profiles: ApiAdminAffiliateProfile[]
  commissions: ApiAffiliateCommission[]
  withdrawals: ApiAffiliateWithdrawal[]
}

export const useApi = () => {
  const config = useRuntimeConfig()
  const token = useCookie<string | null>('image_ai_token')

  const request = async <T>(path: string, options: Parameters<typeof $fetch>[1] = {}) => {
    const headers = new Headers(options.headers as HeadersInit | undefined)
    if (token.value) {
      headers.set('Authorization', `Bearer ${token.value}`)
    }

    let response: ApiEnvelope<T>
    try {
      response = await $fetch<ApiEnvelope<T>>(`${config.public.apiBase}${path}`, {
        ...options,
        headers
      })
    } catch (error: unknown) {
      const fetchError = error as { data?: ApiEnvelope<T>, message?: string }
      throw new Error(fetchError.data?.error || fetchError.message || '请求失败', { cause: error })
    }

    if (response.error) {
      throw new Error(response.error)
    }
    return response.data as T
  }

  return {
    get: <T>(path: string) => request<T>(path),
    post: <T>(path: string, body?: Record<string, unknown>) => request<T>(path, { method: 'POST', body }),
    put: <T>(path: string, body?: Record<string, unknown>) => request<T>(path, { method: 'PUT', body }),
    patch: <T>(path: string, body?: Record<string, unknown>) => request<T>(path, { method: 'PATCH', body }),
    delete: <T>(path: string) => request<T>(path, { method: 'DELETE' })
  }
}
