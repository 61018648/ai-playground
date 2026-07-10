<script setup lang="ts">
import type {
  ApiAdminAffiliateOverview,
  ApiAdminGeneration,
  ApiAdminStats,
  ApiAdminPaymentOrder,
  ApiApp,
  ApiBalanceAdjustResult,
  ApiInviteCode,
  ApiLoginLog,
  ApiPaymentPlan,
  ApiProviderConfig,
  ApiSiteSetting,
  ApiTaskLog,
  ApiUser
} from '~/composables/useApi'

const { pageTitle } = useSiteConfig()

useHead(() => ({ title: pageTitle('控制面板') }))

const api = useApi()
const auth = useAuth()
const message = useMessage()
const route = useRoute()
const router = useRouter()
const section = ref('site')
const siteTab = ref<'seo' | 'auth' | 'smtp'>('seo')
const userTab = ref<'list' | 'invites'>('list')
const affiliateTab = ref<'profiles' | 'commissions' | 'withdrawals'>('profiles')
const logTab = ref<'login' | 'task'>('login')
const userEditorOpen = ref(false)
const balanceEditorOpen = ref(false)
const providerEditorOpen = ref(false)
const appEditorOpen = ref(false)
const providerDeleteOpen = ref(false)
const providerToDelete = ref<ApiProviderConfig | null>(null)
const providerModels = ref<string[]>([])
const providerModelsLoading = ref(false)

onMounted(() => {
  auth.loadMe()
})

const isAdmin = computed(() => auth.user.value?.role === 'admin')

const { data: stats, pending: statsPending, refresh: refreshStats } = await useAsyncData(
  'admin-overview',
  () => auth.token.value ? api.get<ApiAdminStats>('/admin/overview') : Promise.resolve(null),
  { default: () => null, watch: [auth.token] }
)

const { data: settings, refresh: refreshSettings } = await useAsyncData(
  'admin-settings',
  () => auth.token.value ? api.get<ApiSiteSetting[]>('/admin/settings') : Promise.resolve([]),
  { default: () => [], watch: [auth.token] }
)

const { data: users, refresh: refreshUsers } = await useAsyncData(
  'admin-users',
  () => auth.token.value ? api.get<ApiUser[]>('/admin/users?limit=100') : Promise.resolve([]),
  { default: () => [], watch: [auth.token] }
)

const { data: inviteCodes, refresh: refreshInviteCodes } = await useAsyncData(
  'admin-invite-codes',
  () => auth.token.value ? api.get<ApiInviteCode[]>('/admin/invite-codes?limit=200') : Promise.resolve([]),
  { default: () => [], watch: [auth.token] }
)

const { data: apps, refresh: refreshApps } = await useAsyncData(
  'admin-apps',
  () => auth.token.value ? api.get<ApiApp[]>('/admin/apps') : Promise.resolve([]),
  { default: () => [], watch: [auth.token] }
)

const { data: generations, refresh: refreshGenerations } = await useAsyncData(
  'admin-generations',
  () => auth.token.value ? api.get<ApiAdminGeneration[]>('/admin/generations?limit=100') : Promise.resolve([]),
  { default: () => [], watch: [auth.token] }
)

const { data: paymentOrders, refresh: refreshPaymentOrders } = await useAsyncData(
  'admin-payment-orders',
  () => auth.token.value ? api.get<ApiAdminPaymentOrder[]>('/admin/payment-orders?limit=100') : Promise.resolve([]),
  { default: () => [], watch: [auth.token] }
)

const { data: affiliates, refresh: refreshAffiliates } = await useAsyncData(
  'admin-affiliates',
  () => auth.token.value ? api.get<ApiAdminAffiliateOverview>('/admin/affiliates?limit=100') : Promise.resolve(null),
  { default: () => null, watch: [auth.token] }
)

const { data: providers, refresh: refreshProviders } = await useAsyncData(
  'admin-api-providers',
  () => auth.token.value ? api.get<ApiProviderConfig[]>('/admin/api-providers') : Promise.resolve([]),
  { default: () => [], watch: [auth.token] }
)

const { data: loginLogs, refresh: refreshLoginLogs } = await useAsyncData(
  'admin-login-logs',
  () => auth.token.value ? api.get<ApiLoginLog[]>('/admin/logs/login?limit=100') : Promise.resolve([]),
  { default: () => [], watch: [auth.token] }
)

const { data: taskLogs, refresh: refreshTaskLogs } = await useAsyncData(
  'admin-task-logs',
  () => auth.token.value ? api.get<ApiTaskLog[]>('/admin/logs/tasks?limit=100') : Promise.resolve([]),
  { default: () => [], watch: [auth.token] }
)

const navItems = [
  { key: 'site', label: '站点配置', icon: 'i-lucide-settings' },
  { key: 'users', label: '用户管理', icon: 'i-lucide-users' },
  { key: 'api', label: '接口配置', icon: 'i-lucide-plug' },
  { key: 'professional', label: '专业绘图', icon: 'i-lucide-palette' },
  { key: 'assistant', label: '智能助手', icon: 'i-lucide-bot' },
  { key: 'payment', label: '支付配置', icon: 'i-lucide-credit-card' },
  { key: 'apps', label: '应用中心', icon: 'i-lucide-store' },
  { key: 'orders', label: '订单管理', icon: 'i-lucide-receipt' },
  { key: 'affiliates', label: '推广返佣', icon: 'i-lucide-hand-coins' },
  { key: 'logs', label: '操作日志', icon: 'i-lucide-scroll-text' },
  { key: 'tasks', label: '任务审计', icon: 'i-lucide-images' }
]

const providerCategoryOptions = [
  { label: '通用生图', value: 'general' },
  { label: '通用文本', value: 'general_text' }
]
const appTypeOptions = [
  { label: '生图', value: 'image' },
  { label: '文本', value: 'text' }
]

const providerCategoryLabel = (value: string) => providerCategoryOptions.find(item => item.value === value)?.label || '通用生图'
const appTypeLabel = (value: string) => appTypeOptions.find(item => item.value === value)?.label || '生图'
const taskLogChannel = (log: ApiTaskLog) => {
  const meta = log.meta || {}
  const providerName = typeof meta.providerName === 'string' ? meta.providerName.trim() : ''
  const provider = typeof meta.provider === 'string' ? meta.provider.trim() : ''
  const model = typeof meta.model === 'string' ? meta.model.trim() : ''
  if (providerName && model) return providerName + ' / ' + model
  if (providerName) return providerName
  if (provider && model) return provider + ' / ' + model
  return model || provider || '-'
}
const activeProviderCategory = computed(() => {
  if (section.value === 'assistant') return 'general_text'
  return 'general'
})
const appProviderOptionsByType = computed(() => {
  const expectedProviderCategory = appForm.appType === 'text' ? 'general_text' : 'general'
  return providers.value
    .filter(provider => provider.category === expectedProviderCategory)
    .map(provider => ({
      label: provider.name + ' / ' + providerCategoryLabel(provider.category) + ' / ' + (provider.model || '未设置模型') + (provider.enabled ? '' : ' / 已停用'),
      value: provider.id
    }))
})
const drawProviderOptions = computed(() => [
  { label: '未选择', value: '' },
  ...providers.value
    .filter(item => item.category === 'general')
    .map(item => ({
      label: item.name + ' / ' + (item.model || '未设置模型') + (item.enabled ? '' : ' / 已停用'),
      value: item.id
    }))
])
const rewriteProviderOptions = computed(() => [
  { label: '未选择', value: '' },
  ...providers.value
    .filter(item => item.category === 'general_text')
    .map(item => ({
      label: item.name + ' / ' + (item.model || '未设置模型') + (item.enabled ? '' : ' / 已停用'),
      value: item.id
    }))
])
const currentTabValue = computed(() => {
  if (section.value === 'site') return siteTab.value
  if (section.value === 'users') return userTab.value
  if (section.value === 'affiliates') return affiliateTab.value
  if (section.value === 'logs') return logTab.value
  return ''
})

const setAdminSection = (key: string) => {
  section.value = key
}

const validNavKeys = computed(() => navItems.map(item => item.key))
const readStringQuery = (value: unknown) => Array.isArray(value) ? String(value[0] || '') : String(value || '')

watch(
  () => route.query,
  (query) => {
    const nextSection = readStringQuery(query.section)
    const nextTab = readStringQuery(query.tab)
    if (nextSection && validNavKeys.value.includes(nextSection)) {
      section.value = nextSection
    }
    if (section.value === 'site' && ['seo', 'auth', 'smtp'].includes(nextTab)) {
      siteTab.value = nextTab as typeof siteTab.value
    }
    if (section.value === 'users' && ['list', 'invites'].includes(nextTab)) {
      userTab.value = nextTab as typeof userTab.value
    }
    if (section.value === 'affiliates' && ['profiles', 'commissions', 'withdrawals'].includes(nextTab)) {
      affiliateTab.value = nextTab as typeof affiliateTab.value
    }
    if (section.value === 'logs' && ['login', 'task'].includes(nextTab)) {
      logTab.value = nextTab as typeof logTab.value
    }
  },
  { immediate: true }
)

watch(
  [section, currentTabValue],
  ([nextSection, nextTab]) => {
    const query: Record<string, string> = { ...route.query as Record<string, string>, section: nextSection }
    if (nextTab) {
      query.tab = nextTab
    } else {
      delete query.tab
    }
    if (readStringQuery(route.query.section) !== query.section || readStringQuery(route.query.tab) !== (query.tab || '')) {
      router.replace({ query })
    }
  }
)

const membershipOptions = [
  { label: 'Free', value: 'free' },
  { label: 'V1会员', value: 'v1' },
  { label: 'V2会员', value: 'v2' }
]

const statCards = computed(() => [
  { label: '用户', value: stats.value?.usersTotal ?? 0, icon: 'i-lucide-users' },
  { label: '应用', value: stats.value?.appsTotal ?? 0, icon: 'i-lucide-layout-grid' },
  { label: '任务', value: stats.value?.generationsTotal ?? 0, icon: 'i-lucide-sparkles' },
  { label: '今日', value: stats.value?.todayGenerations ?? 0, icon: 'i-lucide-calendar-days' }
])

const settingMap = computed(() => Object.fromEntries(settings.value.map(item => [item.key, item.value])))

const seoForm = reactive({
  siteName: '',
  title: '',
  description: '',
  keywords: ''
})

const authForm = reactive({
  allowRegister: true,
  allowPasswordLogin: true,
  requireEmailCode: true,
  inviteOnly: false
})

const smtpForm = reactive({
  host: '',
  port: 587,
  username: '',
  password: '',
  fromName: '',
  fromEmail: '',
  secure: false
})

const paymentForm = reactive({
  enabled: false,
  provider: 'epay',
  gatewayUrl: '',
  pid: '',
  key: '',
  notifyUrl: '',
  returnUrl: '',
  signType: 'MD5',
  channels: ['alipay'],
  creditPlans: [] as ApiPaymentPlan[],
  membershipPlans: [] as ApiPaymentPlan[]
})

const professionalDrawForm = reactive({
  drawProviderId: '',
  rewriteProviderId: ''
})

const defaultCreditPlans = () => [
  { code: 'credits_basic', name: '基础积分', orderType: 'credits' as const, amount: '30.00', credits: 3000, membershipLevel: '', desc: '3000 积分', period: '' },
  { code: 'credits_plus', name: '高级积分', orderType: 'credits' as const, amount: '50.00', credits: 5000, membershipLevel: '', desc: '5000 积分', period: '' },
  { code: 'credits_super', name: '超级积分', orderType: 'credits' as const, amount: '100.00', credits: 12000, membershipLevel: '', desc: '12000 积分', period: '' }
]

const defaultMembershipPlans = () => [
  { code: 'vip_v1', name: 'V1会员', orderType: 'membership' as const, amount: '39.00', credits: 0, membershipLevel: 'v1', desc: '开通 V1 会员', period: '30天' },
  { code: 'vip_v2', name: 'V2会员', orderType: 'membership' as const, amount: '69.00', credits: 0, membershipLevel: 'v2', desc: '开通 V2 会员', period: '30天' },
  { code: 'vip_v2_year', name: 'V2年费会员', orderType: 'membership' as const, amount: '199.00', credits: 0, membershipLevel: 'v2', desc: '开通 V2 年费会员', period: '365天' }
]

const paymentChannelOptions = [
  { label: '支付宝', value: 'alipay' },
  { label: '微信', value: 'wxpay' },
  { label: 'QQ 钱包', value: 'qqpay' }
]

const hasPaymentChannel = (value: string) => paymentForm.channels.includes(value)

const togglePaymentChannel = (value: string) => {
  if (hasPaymentChannel(value)) {
    paymentForm.channels = paymentForm.channels.filter(channel => channel !== value)
    return
  }
  paymentForm.channels = [...paymentForm.channels, value]
}

const providerForm = reactive({
  id: '',
  name: '',
  category: 'general',
  provider: 'openai',
  baseUrl: '',
  apiKey: '',
  model: '',
  enabled: false,
  sortOrder: 100
})

const inviteForm = reactive({
  count: 10,
  amount: '10',
  note: '',
  expiresAt: ''
})

const userForm = reactive({
  id: '',
  email: '',
  password: '',
  nickname: '',
  role: 'user',
  status: 'active',
  membershipLevel: 'free',
  credits: 0
})

const balanceForm = reactive({
  userId: '',
  email: '',
  current: '0',
  type: 'increase',
  amount: '',
  note: ''
})

const appForm = reactive({
  id: '',
  providerId: '',
  code: '',
  name: '',
  appType: 'image' as 'image' | 'text',
  category: '',
  description: '',
  icon: 'i-lucide-sparkles',
  iconColor: 'bg-emerald-100 text-emerald-600',
  coverUrl: '',
  promptTemplate: '{{prompt}}',
  inputSchemaText: '{\n  "fields": []\n}',
  outputSchemaText: '{}',
  priceFree: '0',
  priceV1: '0',
  priceV2: '0',
  visibility: 'public',
  status: 'active',
  sortOrder: 100
})

watchEffect(() => {
  Object.assign(seoForm, settingMap.value.seo || {})
  Object.assign(authForm, settingMap.value.auth || {})
  Object.assign(smtpForm, settingMap.value.smtp || {})
  const payment = (settingMap.value.payment || {}) as Partial<typeof paymentForm>
  const creditPlans = Array.isArray(payment.creditPlans) && payment.creditPlans.length ? payment.creditPlans : defaultCreditPlans()
  const membershipPlans = Array.isArray(payment.membershipPlans) && payment.membershipPlans.length ? payment.membershipPlans : defaultMembershipPlans()
  Object.assign(paymentForm, {
    channels: ['alipay'],
    ...payment,
    creditPlans,
    membershipPlans
  })
  Object.assign(professionalDrawForm, settingMap.value.professional_draw || {})
})

const errorText = (error: unknown) => error instanceof Error ? error.message : '操作失败'

const saveSetting = async (key: 'seo' | 'auth' | 'smtp' | 'payment' | 'professional_draw', value: Record<string, unknown>) => {
  try {
    await api.put<ApiSiteSetting>('/admin/settings/' + key, { value })
    await refreshSettings()
    message.success('保存成功', '站点配置已更新')
  } catch (error) {
    message.error('保存失败', errorText(error))
  }
}

const resetProviderForm = () => {
  const category = activeProviderCategory.value
  Object.assign(providerForm, {
    id: '',
    name: '',
    category,
    provider: 'openai',
    baseUrl: '',
    apiKey: '',
    model: '',
    enabled: false,
    sortOrder: 100
  })
  providerModels.value = []
}

const openCreateProvider = () => {
  resetProviderForm()
  providerEditorOpen.value = true
}

const openProviderEditor = (provider: ApiProviderConfig) => {
  Object.assign(providerForm, provider)
  providerModels.value = []
  providerEditorOpen.value = true
}

const addCreditPlan = () => {
  paymentForm.creditPlans.push({
    code: 'credits_' + Date.now(),
    name: '积分套餐',
    orderType: 'credits',
    amount: '10.00',
    credits: 1000,
    membershipLevel: '',
    desc: '1000 积分',
    period: ''
  })
}

const addMembershipPlan = () => {
  paymentForm.membershipPlans.push({
    code: 'vip_' + Date.now(),
    name: '会员套餐',
    orderType: 'membership',
    amount: '39.00',
    credits: 0,
    membershipLevel: 'v1',
    desc: '会员权益套餐',
    period: '30天'
  })
}

const removeCreditPlan = (index: number) => {
  paymentForm.creditPlans.splice(index, 1)
}

const removeMembershipPlan = (index: number) => {
  paymentForm.membershipPlans.splice(index, 1)
}

const savePaymentSetting = async () => {
  const value = {
    ...paymentForm,
    creditPlans: paymentForm.creditPlans.map(plan => ({ ...plan, orderType: 'credits', membershipLevel: '', credits: Number(plan.credits || 0) })),
    membershipPlans: paymentForm.membershipPlans.map(plan => ({ ...plan, orderType: 'membership', credits: 0 }))
  }
  await saveSetting('payment', value)
}

const saveProfessionalDrawSetting = async () => {
  await saveSetting('professional_draw', { ...professionalDrawForm })
}

watch(() => providerForm.category, () => {
  providerForm.provider = 'openai'
})

const filteredProviders = computed(() => providers.value.filter((item) => {
  if (section.value === 'api') {
    return item.category === 'general' || item.category === 'general_text'
  }
  return item.category === activeProviderCategory.value
}))

const providerPayload = (enabled = providerForm.enabled) => ({
  id: providerForm.id,
  name: providerForm.name,
  category: providerForm.category,
  provider: 'openai',
  baseUrl: providerForm.baseUrl,
  apiKey: providerForm.apiKey,
  model: providerForm.model,
  enabled,
  sortOrder: providerForm.sortOrder
})

const saveProviderDraftForModels = async () => {
  const saved = await api.post<ApiProviderConfig>('/admin/api-providers', providerPayload(false))
  Object.assign(providerForm, saved)
  await refreshProviders()
  return saved
}

const fetchProviderModels = async () => {
  try {
    providerModelsLoading.value = true
    const inputBaseURL = providerForm.baseUrl
    const inputAPIKey = providerForm.apiKey
    await saveProviderDraftForModels()
    providerModels.value = await api.post<string[]>('/admin/api-providers/' + providerForm.id + '/models', {
      baseUrl: inputBaseURL,
      apiKey: inputAPIKey
    })
    message.success('获取成功', '已获取 ' + providerModels.value.length + ' 个模型')
  } catch (error) {
    message.error('获取失败', errorText(error))
  } finally {
    providerModelsLoading.value = false
  }
}

const saveProvider = async () => {
  try {
    await api.post<ApiProviderConfig>('/admin/api-providers', providerPayload())
    providerEditorOpen.value = false
    resetProviderForm()
    await refreshProviders()
    message.success('保存成功', '接口配置已更新')
  } catch (error) {
    message.error('保存失败', errorText(error))
  }
}

const deleteProvider = (provider: ApiProviderConfig) => {
  providerToDelete.value = provider
  providerDeleteOpen.value = true
}

const confirmDeleteProvider = async () => {
  if (!providerToDelete.value) return
  try {
    await api.delete<{ ok: boolean }>('/admin/api-providers/' + providerToDelete.value.id)
    providerDeleteOpen.value = false
    providerToDelete.value = null
    await Promise.all([refreshProviders(), refreshApps()])
    message.success('删除成功', '接口配置已删除')
  } catch (error) {
    message.error('删除失败', errorText(error))
  }
}

const cancelPaymentOrder = async (tradeNo: string) => {
  try {
    await api.post<ApiAdminPaymentOrder>('/admin/payment-orders/' + tradeNo + '/cancel')
    await refreshPaymentOrders()
    message.success('取消成功', '订单已取消')
  } catch (error) {
    message.error('取消失败', errorText(error))
  }
}

const productTypeLabel = (value: string) => ({
  credits: '积分充值',
  membership: 'VIP会员'
}[value] || value || '-')

const withdrawalStatusLabel = (value: string) => ({
  pending: '提现中',
  paid: '已打款',
  rejected: '已驳回'
}[value] || value || '-')

const formatAdminDate = (value?: string) => value ? new Date(value).toLocaleString('zh-CN') : '-'

const openPayUrl = (url: string) => {
  if (url) {
    window.open(url, '_blank', 'noopener,noreferrer')
  }
}

const resetUserForm = () => {
  Object.assign(userForm, {
    id: '',
    email: '',
    password: '',
    nickname: '',
    role: 'user',
    status: 'active',
    membershipLevel: 'free',
    credits: 0
  })
}

const openCreateUser = () => {
  resetUserForm()
  userEditorOpen.value = true
}

const openUserEditor = (user: ApiUser) => {
  Object.assign(userForm, {
    id: user.id,
    email: user.email,
    password: '',
    nickname: user.nickname,
    role: user.role,
    status: user.status,
    membershipLevel: user.membershipLevel || 'free',
    credits: user.credits
  })
  userEditorOpen.value = true
}

const openBalanceEditor = (user: ApiUser) => {
  Object.assign(balanceForm, {
    userId: user.id,
    email: user.email,
    current: user.balance,
    type: 'increase',
    amount: '',
    note: ''
  })
  balanceEditorOpen.value = true
}

const saveBalanceAdjustment = async () => {
  try {
    const result = await api.post<ApiBalanceAdjustResult>('/admin/users/' + balanceForm.userId + '/balance', {
      type: balanceForm.type,
      amount: String(balanceForm.amount),
      note: balanceForm.note
    })
    balanceEditorOpen.value = false
    await Promise.all([refreshUsers(), refreshTaskLogs()])
    message.success('调整成功', '余额已变更为 ' + result.user.balance)
  } catch (error) {
    message.error('调整失败', errorText(error))
  }
}

const saveUser = async () => {
  try {
    const payload: Record<string, unknown> = {
      email: userForm.email,
      nickname: userForm.nickname,
      role: userForm.role,
      status: userForm.status,
      membershipLevel: userForm.membershipLevel,
      credits: userForm.credits
    }
    if (userForm.password) {
      payload.password = userForm.password
    }
    if (userForm.id) {
      await api.put<ApiUser>('/admin/users/' + userForm.id, payload)
    } else {
      await api.post<ApiUser>('/admin/users', payload)
    }
    userEditorOpen.value = false
    await Promise.all([refreshUsers(), refreshStats()])
    message.success('保存成功', userForm.id ? '用户资料已更新' : '用户已创建')
  } catch (error) {
    message.error('保存失败', errorText(error))
  }
}

const createInviteCodes = async () => {
  try {
    await api.post<ApiInviteCode[]>('/admin/invite-codes', {
      count: inviteForm.count,
      amount: String(inviteForm.amount),
      note: inviteForm.note,
      expiresAt: inviteForm.expiresAt ? new Date(inviteForm.expiresAt).toISOString() : ''
    })
    await refreshInviteCodes()
    message.success('生成成功', '已生成 ' + inviteForm.count + ' 个兑换码')
  } catch (error) {
    message.error('生成失败', errorText(error))
  }
}

const exportInviteCodes = () => {
  const rows = [
    ['code', 'amount', 'used', 'usedByEmail', 'usedAt', 'note', 'expiresAt', 'createdAt'],
    ...inviteCodes.value.map(item => [
      item.code,
      item.amount || '0',
      item.usedCount > 0 ? 'yes' : 'no',
      item.usedByEmail || '',
      item.usedAt || '',
      item.note,
      item.expiresAt || '',
      item.createdAt
    ])
  ]
  const csv = rows.map(row => row.map(cell => '"' + cell.replace(/"/g, '""') + '"').join(',')).join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'redeem-codes-' + Date.now() + '.csv'
  link.click()
  URL.revokeObjectURL(url)
  message.success('导出成功', '兑换码 CSV 已生成')
}

const resetAppForm = () => {
  Object.assign(appForm, {
    id: '',
    appType: 'image',
    providerId: appProviderOptionsByType.value[0]?.value || '',
    code: '',
    name: '',
    category: '',
    description: '',
    icon: 'i-lucide-sparkles',
    iconColor: 'bg-emerald-100 text-emerald-600',
    coverUrl: '',
    promptTemplate: '{{prompt}}',
    inputSchemaText: '{\n  "fields": []\n}',
    outputSchemaText: '{}',
    priceFree: '0',
    priceV1: '0',
    priceV2: '0',
    visibility: 'public',
    status: 'active',
    sortOrder: 100
  })
}

const openCreateApp = () => {
  resetAppForm()
  appEditorOpen.value = true
}

const formatJsonText = (value: unknown) => JSON.stringify(value || {}, null, 2)
const selectValue = (value: unknown) => {
  if (typeof value === 'string') return value
  if (value && typeof value === 'object' && 'value' in value) {
    return String((value as { value?: unknown }).value || '')
  }
  return ''
}

const openAppEditor = (app: ApiApp) => {
  Object.assign(appForm, {
    id: app.id,
    appType: app.appType || 'image',
    providerId: app.providerId || '',
    code: app.code,
    name: app.name,
    category: app.category,
    description: app.description,
    icon: app.icon,
    iconColor: app.iconColor,
    coverUrl: app.coverUrl,
    promptTemplate: app.promptTemplate,
    inputSchemaText: formatJsonText(app.inputSchema),
    outputSchemaText: formatJsonText(app.outputSchema),
    priceFree: app.priceFree,
    priceV1: app.priceV1,
    priceV2: app.priceV2,
    visibility: app.visibility,
    status: app.status,
    sortOrder: app.sortOrder
  })
  appEditorOpen.value = true
}

const saveApp = async () => {
  try {
    const payload = {
      providerId: selectValue(appForm.providerId),
      code: appForm.code,
      name: appForm.name,
      appType: selectValue(appForm.appType) || 'image',
      category: appForm.category,
      description: appForm.description,
      icon: appForm.icon,
      iconColor: appForm.iconColor,
      coverUrl: appForm.coverUrl,
      promptTemplate: appForm.promptTemplate,
      inputSchema: JSON.parse(appForm.inputSchemaText || '{}'),
      outputSchema: JSON.parse(appForm.outputSchemaText || '{}'),
      priceFree: appForm.priceFree,
      priceV1: appForm.priceV1,
      priceV2: appForm.priceV2,
      visibility: appForm.visibility,
      status: appForm.status,
      sortOrder: appForm.sortOrder
    }
    if (appForm.id) {
      await api.put<ApiApp>('/admin/apps/' + appForm.id, payload)
    } else {
      await api.post<ApiApp>('/admin/apps', payload)
    }
    appEditorOpen.value = false
    await Promise.all([refreshApps(), refreshStats()])
    message.success('保存成功', appForm.id ? '应用已更新' : '应用已创建')
  } catch (error) {
    message.error('保存失败', errorText(error))
  }
}

watch(() => appForm.appType, () => {
  const valid = appProviderOptionsByType.value.some(provider => provider.value === appForm.providerId)
  if (!valid) {
    appForm.providerId = appProviderOptionsByType.value[0]?.value || ''
  }
})

const refreshAll = () => Promise.all([
  refreshStats(),
  refreshSettings(),
  refreshUsers(),
  refreshInviteCodes(),
  refreshApps(),
  refreshGenerations(),
  refreshPaymentOrders(),
  refreshAffiliates(),
  refreshProviders(),
  refreshLoginLogs(),
  refreshTaskLogs()
])
</script>

<template>
  <div class="min-h-[calc(100vh-4rem)] bg-elevated/30">
    <div
      v-if="!auth.token.value || !isAdmin"
      class="max-w-2xl mx-auto px-4 py-16"
    >
      <UAlert
        :color="!auth.token.value ? 'warning' : 'error'"
        variant="soft"
        :icon="!auth.token.value ? 'i-lucide-lock' : 'i-lucide-shield-alert'"
        :title="!auth.token.value ? '请先登录管理员账号' : '当前账号没有管理员权限'"
        description="请使用管理员账号登录后进入控制面板"
      />
    </div>

    <div
      v-else
      class="flex min-h-[calc(100vh-4rem)]"
    >
      <aside class="w-64 shrink-0 border-r border-default bg-default px-3 py-4">
        <div class="px-3 pb-4 border-b border-default">
          <p class="text-sm font-semibold text-highlighted">
            管理控制台
          </p>
          <p class="text-xs text-dimmed mt-1 truncate">
            {{ auth.user.value?.email }}
          </p>
        </div>
        <nav class="mt-4 space-y-1">
          <button
            v-for="item in navItems"
            :key="item.key"
            type="button"
            class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors text-left"
            :class="section === item.key ? 'bg-primary/10 text-primary' : 'text-muted hover:bg-elevated hover:text-highlighted'"
            @click="setAdminSection(item.key)"
          >
            <UIcon
              :name="item.icon"
              class="w-5 h-5"
            />
            {{ item.label }}
          </button>
        </nav>
      </aside>

      <main
        :key="section"
        class="flex-1 min-w-0 px-6 py-5 space-y-5"
      >
        <div class="flex items-center justify-between">
          <div>
            <h1 class="text-xl font-bold text-highlighted">
              {{ navItems.find(item => item.key === section)?.label }}
            </h1>
            <p class="text-sm text-dimmed mt-1">
              独立后台控制面板
            </p>
          </div>
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="soft"
            :loading="statsPending"
            @click="refreshAll()"
          >
            刷新
          </UButton>
        </div>

        <div class="grid grid-cols-2 xl:grid-cols-4 gap-3">
          <div
            v-for="card in statCards"
            :key="card.label"
            class="rounded-lg border border-default bg-default p-4 flex items-center gap-3"
          >
            <div class="w-10 h-10 rounded-lg bg-primary/10 text-primary flex items-center justify-center">
              <UIcon
                :name="card.icon"
                class="w-5 h-5"
              />
            </div>
            <div>
              <p class="text-xs text-dimmed">
                {{ card.label }}
              </p>
              <p class="text-2xl font-bold text-highlighted">
                {{ card.value }}
              </p>
            </div>
          </div>
        </div>

        <section
          v-if="section === 'site'"
          class="rounded-lg border border-default bg-default overflow-hidden"
        >
          <div class="flex items-center gap-1 p-2 border-b border-default">
            <UButton
              :color="siteTab === 'seo' ? 'primary' : 'neutral'"
              variant="soft"
              size="sm"
              @click="siteTab = 'seo'"
            >
              SEO 设置
            </UButton>
            <UButton
              :color="siteTab === 'auth' ? 'primary' : 'neutral'"
              variant="soft"
              size="sm"
              @click="siteTab = 'auth'"
            >
              注册登录
            </UButton>
            <UButton
              :color="siteTab === 'smtp' ? 'primary' : 'neutral'"
              variant="soft"
              size="sm"
              @click="siteTab = 'smtp'"
            >
              SMTP 配置
            </UButton>
          </div>

          <div
            v-if="siteTab === 'seo'"
            class="max-w-2xl p-4 space-y-3"
          >
            <UFormField label="站点名称">
              <UInput
                v-model="seoForm.siteName"
                placeholder="请输入站点名称"
              />
            </UFormField>
            <UFormField label="SEO 标题">
              <UInput
                v-model="seoForm.title"
                placeholder="请输入搜索引擎标题"
              />
            </UFormField>
            <UFormField label="SEO 描述">
              <UTextarea
                v-model="seoForm.description"
                placeholder="请输入站点描述"
              />
            </UFormField>
            <UFormField label="关键词">
              <UInput
                v-model="seoForm.keywords"
                placeholder="多个关键词用逗号分隔"
              />
            </UFormField>
            <UButton
              block
              @click="saveSetting('seo', { ...seoForm })"
            >
              保存 SEO
            </UButton>
          </div>

          <div
            v-else-if="siteTab === 'auth'"
            class="max-w-2xl p-4 space-y-3"
          >
            <UCheckbox
              v-model="authForm.allowRegister"
              label="允许注册"
            />
            <UCheckbox
              v-model="authForm.allowPasswordLogin"
              label="允许密码登录"
            />
            <UCheckbox
              v-model="authForm.requireEmailCode"
              label="注册需要邮箱验证码"
            />
            <UCheckbox
              v-model="authForm.inviteOnly"
              label="仅邀请码注册"
            />
            <UButton
              block
              @click="saveSetting('auth', { ...authForm })"
            >
              保存登录配置
            </UButton>
          </div>

          <div
            v-else
            class="max-w-2xl p-4 space-y-3"
          >
            <UFormField label="SMTP Host">
              <UInput v-model="smtpForm.host" />
            </UFormField>
            <UFormField label="端口">
              <UInput
                v-model.number="smtpForm.port"
                type="number"
              />
            </UFormField>
            <UFormField label="用户名">
              <UInput v-model="smtpForm.username" />
            </UFormField>
            <UFormField label="密码">
              <UInput
                v-model="smtpForm.password"
                type="password"
              />
            </UFormField>
            <UFormField label="发件人邮箱">
              <UInput v-model="smtpForm.fromEmail" />
            </UFormField>
            <UFormField label="发件人名称">
              <UInput
                v-model="smtpForm.fromName"
                placeholder="请输入发件人名称"
              />
            </UFormField>
            <UCheckbox
              v-model="smtpForm.secure"
              label="启用 SSL/TLS"
            />
            <UButton
              block
              @click="saveSetting('smtp', { ...smtpForm })"
            >
              保存 SMTP
            </UButton>
          </div>
        </section>

        <section
          v-else-if="section === 'users'"
          class="rounded-lg border border-default bg-default overflow-hidden"
        >
          <div class="flex items-center justify-between gap-3 p-2 border-b border-default">
            <div class="flex items-center gap-1">
              <UButton
                :color="userTab === 'list' ? 'primary' : 'neutral'"
                variant="soft"
                size="sm"
                @click="userTab = 'list'"
              >
                用户列表
              </UButton>
              <UButton
                :color="userTab === 'invites' ? 'primary' : 'neutral'"
                variant="soft"
                size="sm"
                @click="userTab = 'invites'"
              >
                兑换码
              </UButton>
            </div>
            <div class="flex items-center gap-2">
              <UButton
                v-if="userTab === 'list'"
                icon="i-lucide-user-plus"
                size="sm"
                @click="openCreateUser()"
              >
                添加用户
              </UButton>
              <UButton
                v-else
                icon="i-lucide-download"
                color="neutral"
                variant="soft"
                size="sm"
                @click="exportInviteCodes()"
              >
                导出
              </UButton>
            </div>
          </div>

          <template v-if="userTab === 'list'">
            <div class="grid grid-cols-[1.3fr_0.8fr_0.55fr_0.55fr_0.55fr_0.85fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
              <span>邮箱</span>
              <span>昵称</span>
              <span>余额</span>
              <span>积分</span>
              <span>状态</span>
              <span>操作</span>
            </div>
            <div
              v-for="user in users"
              :key="user.id"
              class="grid grid-cols-[1.3fr_0.8fr_0.55fr_0.55fr_0.55fr_0.85fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
            >
              <span class="truncate text-highlighted">{{ user.email }}</span>
              <span class="truncate text-toned">{{ user.nickname || '-' }}</span>
              <span class="text-dimmed">{{ user.balance }}</span>
              <span class="text-dimmed">{{ user.credits }}</span>
              <UBadge
                :color="user.status === 'active' ? 'success' : 'error'"
                variant="soft"
              >
                {{ user.status === 'active' ? '启用' : '停用' }}
              </UBadge>
              <div class="flex items-center gap-2">
                <UButton
                  size="xs"
                  color="primary"
                  variant="soft"
                  @click="openBalanceEditor(user)"
                >
                  调整余额
                </UButton>
                <UButton
                  size="xs"
                  color="neutral"
                  variant="soft"
                  @click="openUserEditor(user)"
                >
                  编辑
                </UButton>
              </div>
            </div>
          </template>

          <div
            v-else
            class="grid grid-cols-1 xl:grid-cols-[22rem_1fr] gap-4 p-4"
          >
            <div class="space-y-3">
              <h2 class="font-semibold text-highlighted">
                生成兑换码
              </h2>
              <UFormField label="生成数量">
                <UInput
                  v-model.number="inviteForm.count"
                  type="number"
                  min="1"
                  max="200"
                />
              </UFormField>
              <UFormField label="兑换面值">
                <UInput
                  v-model="inviteForm.amount"
                  type="number"
                  min="0.01"
                  step="0.01"
                  placeholder="例如：10"
                />
              </UFormField>
              <UFormField label="备注">
                <UInput
                  v-model="inviteForm.note"
                  placeholder="例如：渠道、活动、批次"
                />
              </UFormField>
              <UFormField label="过期时间">
                <UInput
                  v-model="inviteForm.expiresAt"
                  type="datetime-local"
                />
              </UFormField>
              <UButton
                block
                @click="createInviteCodes()"
              >
                批量生成
              </UButton>
            </div>

            <div class="overflow-hidden rounded-lg border border-default">
              <div class="grid grid-cols-[1fr_0.55fr_0.55fr_1.2fr_1fr_1fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
                <span>兑换码</span>
                <span>面值</span>
                <span>状态</span>
                <span>使用人</span>
                <span>备注</span>
                <span>创建时间</span>
              </div>
              <div
                v-for="item in inviteCodes"
                :key="item.id"
                class="grid grid-cols-[1fr_0.55fr_0.55fr_1.2fr_1fr_1fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
              >
                <span class="font-mono text-highlighted">{{ item.code }}</span>
                <span class="text-primary">{{ item.amount || '0' }}</span>
                <span :class="item.usedCount > 0 ? 'text-error' : 'text-success'">{{ item.usedCount > 0 ? '已使用' : '未使用' }}</span>
                <span class="truncate text-toned">{{ item.usedByEmail || '-' }}</span>
                <span class="truncate text-toned">{{ item.note || '-' }}</span>
                <span class="text-dimmed">{{ new Date(item.createdAt).toLocaleString('zh-CN') }}</span>
              </div>
            </div>
          </div>
        </section>

        <section
          v-else-if="section === 'professional'"
          class="rounded-lg border border-default bg-default overflow-hidden"
        >
          <div class="flex items-center justify-between gap-3 p-2 border-b border-default">
            <p class="text-sm font-semibold text-highlighted px-2">
              专业绘图
            </p>
          </div>
          <div class="p-4 space-y-4">
            <div class="grid grid-cols-1 xl:grid-cols-2 gap-4">
              <div class="rounded-lg border border-default p-4 space-y-3">
                <h2 class="font-semibold text-highlighted">
                  生图接口
                </h2>
                <UFormField label="选择通用生图接口">
                  <select
                    v-model="professionalDrawForm.drawProviderId"
                    class="w-full rounded-md border border-default bg-default px-3 py-2 text-sm text-highlighted outline-none transition-colors focus:border-primary"
                  >
                    <option
                      v-for="provider in drawProviderOptions"
                      :key="'draw-' + provider.value"
                      :value="provider.value"
                    >
                      {{ provider.label }}
                    </option>
                  </select>
                </UFormField>
              </div>

              <div class="rounded-lg border border-default p-4 space-y-3">
                <h2 class="font-semibold text-highlighted">
                  润色接口
                </h2>
                <UFormField label="选择通用文本接口">
                  <select
                    v-model="professionalDrawForm.rewriteProviderId"
                    class="w-full rounded-md border border-default bg-default px-3 py-2 text-sm text-highlighted outline-none transition-colors focus:border-primary"
                  >
                    <option
                      v-for="provider in rewriteProviderOptions"
                      :key="'rewrite-' + provider.value"
                      :value="provider.value"
                    >
                      {{ provider.label }}
                    </option>
                  </select>
                </UFormField>
              </div>
            </div>

            <UButton
              block
              @click="saveProfessionalDrawSetting"
            >
              保存专业绘图配置
            </UButton>
          </div>
        </section>

        <section
          v-else-if="section === 'api' || section === 'assistant'"
          class="rounded-lg border border-default bg-default overflow-hidden"
        >
          <div class="flex items-center justify-between gap-3 p-2 border-b border-default">
            <p class="text-sm font-semibold text-highlighted px-2">
              {{ section === 'assistant' ? '通用文本接口' : '接口配置' }}
            </p>
            <UButton
              icon="i-lucide-plus"
              size="sm"
              @click="openCreateProvider()"
            >
              新建接口
            </UButton>
          </div>

          <div class="overflow-hidden">
            <div class="grid grid-cols-[0.9fr_0.75fr_0.75fr_1.1fr_0.75fr_0.55fr_0.55fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
              <span>名称</span>
              <span>分类</span>
              <span>接口类型</span>
              <span>Base URL</span>
              <span>模型</span>
              <span>状态</span>
              <span>操作</span>
            </div>
            <div
              v-for="provider in filteredProviders"
              :key="provider.id"
              class="grid grid-cols-[0.9fr_0.75fr_0.75fr_1.1fr_0.75fr_0.55fr_0.55fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
            >
              <span class="truncate text-highlighted">{{ provider.name }}</span>
              <span class="truncate text-toned">{{ providerCategoryLabel(provider.category) }}</span>
              <span class="truncate text-toned">{{ provider.provider }}</span>
              <span class="truncate text-dimmed">{{ provider.baseUrl || '-' }}</span>
              <span class="truncate text-dimmed">{{ provider.model || '-' }}</span>
              <UBadge
                :color="provider.enabled ? 'success' : 'neutral'"
                variant="soft"
              >
                {{ provider.enabled ? '启用' : '停用' }}
              </UBadge>
              <div class="flex items-center gap-2">
                <UButton
                  size="xs"
                  color="neutral"
                  variant="soft"
                  @click="openProviderEditor(provider)"
                >
                  编辑
                </UButton>
                <UButton
                  size="xs"
                  color="error"
                  variant="soft"
                  @click="deleteProvider(provider)"
                >
                  删除
                </UButton>
              </div>
            </div>
          </div>
        </section>

        <section
          v-else-if="section === 'payment'"
          class="rounded-lg border border-default bg-default overflow-hidden"
        >
          <div class="flex items-center justify-between gap-3 p-2 border-b border-default">
            <p class="text-sm font-semibold text-highlighted px-2">
              易支付配置
            </p>
          </div>
          <div class="p-4 space-y-5">
            <div class="grid grid-cols-1 xl:grid-cols-2 gap-4">
              <div class="rounded-lg border border-default p-4 space-y-3">
                <h2 class="font-semibold text-highlighted">
                  易支付参数
                </h2>
                <UCheckbox
                  v-model="paymentForm.enabled"
                  label="启用支付"
                />
                <UFormField label="支付类型">
                  <USelect
                    v-model="paymentForm.provider"
                    :items="[{ label: '易支付', value: 'epay' }]"
                    value-key="value"
                    label-key="label"
                  />
                </UFormField>
                <UFormField label="网关地址">
                  <UInput
                    v-model="paymentForm.gatewayUrl"
                    placeholder="https://pay.example.com/"
                  />
                </UFormField>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <UFormField label="商户 PID">
                    <UInput v-model="paymentForm.pid" />
                  </UFormField>
                  <UFormField label="签名方式">
                    <USelect
                      v-model="paymentForm.signType"
                      :items="['MD5']"
                    />
                  </UFormField>
                </div>
                <UFormField label="商户密钥">
                  <UInput
                    v-model="paymentForm.key"
                    type="password"
                  />
                </UFormField>
                <UFormField label="异步通知地址">
                  <UInput v-model="paymentForm.notifyUrl" />
                </UFormField>
                <UFormField label="同步返回地址">
                  <UInput v-model="paymentForm.returnUrl" />
                </UFormField>
                <UFormField label="支付渠道">
                  <div class="flex flex-wrap gap-3">
                    <button
                      v-for="item in paymentChannelOptions"
                      :key="item.value"
                      type="button"
                      class="inline-flex items-center gap-2 rounded-full border px-3 py-2 text-sm transition-colors"
                      :class="hasPaymentChannel(item.value) ? 'border-primary bg-primary/10 text-primary' : 'border-default text-muted hover:text-highlighted hover:border-primary/50'"
                      @click="togglePaymentChannel(item.value)"
                    >
                      <UIcon
                        :name="hasPaymentChannel(item.value) ? 'i-lucide-check-circle-2' : 'i-lucide-circle'"
                        class="h-4 w-4"
                      />
                      {{ item.label }}
                    </button>
                  </div>
                </UFormField>
              </div>

              <div class="rounded-lg border border-default p-4 space-y-3">
                <div class="flex items-center justify-between gap-3">
                  <h2 class="font-semibold text-highlighted">
                    会员购买列表
                  </h2>
                  <UButton
                    icon="i-lucide-plus"
                    size="xs"
                    variant="soft"
                    @click="addMembershipPlan"
                  >
                    添加
                  </UButton>
                </div>
                <div class="space-y-3">
                  <div
                    v-for="(plan, index) in paymentForm.membershipPlans"
                    :key="plan.code"
                    class="rounded-lg border border-default bg-elevated/30 p-3 space-y-3"
                  >
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                      <UFormField label="等级名称">
                        <UInput v-model="plan.name" />
                      </UFormField>
                      <UFormField label="等级标识">
                        <USelect
                          v-model="plan.membershipLevel"
                          :items="membershipOptions"
                          value-key="value"
                          label-key="label"
                        />
                      </UFormField>
                      <UFormField label="套餐编码">
                        <UInput v-model="plan.code" />
                      </UFormField>
                      <UFormField label="价格">
                        <UInput
                          v-model="plan.amount"
                          type="number"
                          step="0.01"
                        />
                      </UFormField>
                      <UFormField label="周期">
                        <UInput
                          v-model="plan.period"
                          placeholder="例如：30天、365天"
                        />
                      </UFormField>
                      <UFormField label="描述">
                        <UInput v-model="plan.desc" />
                      </UFormField>
                    </div>
                    <div class="flex justify-end">
                      <UButton
                        color="error"
                        variant="ghost"
                        size="xs"
                        @click="removeMembershipPlan(index)"
                      >
                        删除套餐
                      </UButton>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="rounded-lg border border-default p-4 space-y-3">
              <div class="flex items-center justify-between gap-3">
                <h2 class="font-semibold text-highlighted">
                  积分充值列表
                </h2>
                <UButton
                  icon="i-lucide-plus"
                  size="xs"
                  variant="soft"
                  @click="addCreditPlan"
                >
                  添加
                </UButton>
              </div>
              <div class="grid grid-cols-1 xl:grid-cols-2 gap-3">
                <div
                  v-for="(plan, index) in paymentForm.creditPlans"
                  :key="plan.code"
                  class="rounded-lg border border-default bg-elevated/30 p-3 space-y-3"
                >
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <UFormField label="套餐名称">
                      <UInput v-model="plan.name" />
                    </UFormField>
                    <UFormField label="套餐编码">
                      <UInput v-model="plan.code" />
                    </UFormField>
                    <UFormField label="价格">
                      <UInput
                        v-model="plan.amount"
                        type="number"
                        step="0.01"
                      />
                    </UFormField>
                    <UFormField label="积分数量">
                      <UInput
                        v-model.number="plan.credits"
                        type="number"
                        min="1"
                      />
                    </UFormField>
                    <UFormField
                      label="描述"
                      class="md:col-span-2"
                    >
                      <UInput v-model="plan.desc" />
                    </UFormField>
                  </div>
                  <div class="flex justify-end">
                    <UButton
                      color="error"
                      variant="ghost"
                      size="xs"
                      @click="removeCreditPlan(index)"
                    >
                      删除套餐
                    </UButton>
                  </div>
                </div>
              </div>
            </div>

            <UButton
              block
              @click="savePaymentSetting"
            >
              保存支付配置
            </UButton>
          </div>
        </section>

        <section
          v-else-if="section === 'apps'"
          class="rounded-lg border border-default bg-default overflow-hidden"
        >
          <div class="flex items-center justify-between gap-3 p-2 border-b border-default">
            <p class="text-sm font-semibold text-highlighted px-2">
              应用产品
            </p>
            <UButton
              icon="i-lucide-plus"
              size="sm"
              @click="openCreateApp()"
            >
              新建应用
            </UButton>
          </div>
          <div class="grid grid-cols-[0.75fr_0.9fr_0.45fr_0.75fr_1fr_0.55fr_0.55fr_0.4fr_0.5fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
            <span>标识</span>
            <span>名称</span>
            <span>类型</span>
            <span>分类</span>
            <span>接口配置</span>
            <span>可见性</span>
            <span>状态</span>
            <span>排序</span>
            <span>操作</span>
          </div>
          <div
            v-for="app in apps"
            :key="app.id"
            class="grid grid-cols-[0.75fr_0.9fr_0.45fr_0.75fr_1fr_0.55fr_0.55fr_0.4fr_0.5fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
          >
            <span class="truncate font-mono text-xs text-dimmed">{{ app.code }}</span>
            <span class="truncate text-highlighted">{{ app.name }}</span>
            <span class="truncate text-toned">{{ appTypeLabel(app.appType) }}</span>
            <span class="truncate text-toned">{{ app.category }}</span>
            <span class="truncate text-toned">{{ app.providerName || '未绑定' }}</span>
            <UBadge
              :color="app.visibility === 'public' ? 'primary' : 'neutral'"
              variant="soft"
            >
              {{ app.visibility === 'public' ? '公开' : '私有' }}
            </UBadge>
            <UBadge
              :color="app.status === 'active' ? 'success' : 'error'"
              variant="soft"
            >
              {{ app.status === 'active' ? '启用' : '停用' }}
            </UBadge>
            <span class="text-dimmed">{{ app.sortOrder }}</span>
            <UButton
              size="xs"
              color="neutral"
              variant="soft"
              @click="openAppEditor(app)"
            >
              编辑
            </UButton>
          </div>
        </section>

        <section
          v-else-if="section === 'orders'"
          class="rounded-lg border border-default bg-default overflow-hidden"
        >
          <div class="flex items-center justify-between gap-3 p-2 border-b border-default">
            <p class="text-sm font-semibold text-highlighted px-2">
              订单管理
            </p>
            <UButton
              icon="i-lucide-refresh-cw"
              size="sm"
              color="neutral"
              variant="soft"
              @click="refreshPaymentOrders()"
            >
              刷新
            </UButton>
          </div>
          <div class="grid grid-cols-[1fr_0.75fr_0.8fr_0.8fr_0.7fr_0.7fr_0.6fr_1fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
            <span>订单号</span>
            <span>用户</span>
            <span>套餐</span>
            <span>金额</span>
            <span>状态</span>
            <span>超时</span>
            <span>创建</span>
            <span>操作</span>
          </div>
          <div
            v-for="order in paymentOrders"
            :key="order.id"
            class="grid grid-cols-[1fr_0.75fr_0.8fr_0.8fr_0.7fr_0.7fr_0.6fr_1fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
          >
            <span class="font-mono text-xs text-highlighted truncate">{{ order.tradeNo }}</span>
            <span class="truncate text-toned">{{ order.nickname || order.email }}</span>
            <span class="truncate text-toned">{{ order.planName }}</span>
            <span class="text-dimmed">¥{{ order.amount }}</span>
            <UBadge
              :color="order.status === 'paid' ? 'success' : order.status === 'cancelled' ? 'neutral' : order.status === 'expired' ? 'warning' : 'primary'"
              variant="soft"
            >
              {{ order.status }}
            </UBadge>
            <span class="text-dimmed">{{ order.expiresAt ? new Date(order.expiresAt).toLocaleTimeString('zh-CN', { hour12: false }) : '-' }}</span>
            <span class="text-dimmed">{{ new Date(order.createdAt).toLocaleString('zh-CN') }}</span>
            <div class="flex items-center gap-2">
              <UButton
                size="xs"
                color="neutral"
                variant="soft"
                @click="openPayUrl(order.payUrl)"
              >
                查看
              </UButton>
              <UButton
                v-if="order.status === 'pending'"
                size="xs"
                color="error"
                variant="soft"
                @click="cancelPaymentOrder(order.tradeNo)"
              >
                取消
              </UButton>
            </div>
          </div>
        </section>

        <section
          v-else-if="section === 'affiliates'"
          class="rounded-lg border border-default bg-default overflow-hidden"
        >
          <div class="flex items-center justify-between gap-3 p-2 border-b border-default">
            <div class="flex items-center gap-1">
              <UButton
                :color="affiliateTab === 'profiles' ? 'primary' : 'neutral'"
                variant="soft"
                size="sm"
                @click="affiliateTab = 'profiles'"
              >
                代理用户
              </UButton>
              <UButton
                :color="affiliateTab === 'commissions' ? 'primary' : 'neutral'"
                variant="soft"
                size="sm"
                @click="affiliateTab = 'commissions'"
              >
                返佣记录
              </UButton>
              <UButton
                :color="affiliateTab === 'withdrawals' ? 'primary' : 'neutral'"
                variant="soft"
                size="sm"
                @click="affiliateTab = 'withdrawals'"
              >
                提现记录
              </UButton>
            </div>
            <UButton
              icon="i-lucide-refresh-cw"
              size="sm"
              color="neutral"
              variant="soft"
              @click="refreshAffiliates()"
            >
              刷新
            </UButton>
          </div>

          <template v-if="affiliateTab === 'profiles'">
            <div class="grid grid-cols-[1.2fr_0.75fr_0.7fr_0.65fr_0.7fr_0.7fr_0.7fr_0.8fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
              <span>用户</span>
              <span>推广码</span>
              <span>等级</span>
              <span>比例</span>
              <span>总佣金</span>
              <span>可提现</span>
              <span>邀请</span>
              <span>访问</span>
            </div>
            <div
              v-for="profile in affiliates?.profiles || []"
              :key="profile.userId"
              class="grid grid-cols-[1.2fr_0.75fr_0.7fr_0.65fr_0.7fr_0.7fr_0.7fr_0.8fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
            >
              <span class="truncate text-highlighted">{{ profile.nickname || profile.email }}</span>
              <span class="font-mono text-xs text-toned">{{ profile.code }}</span>
              <span class="truncate text-toned">{{ profile.level }}</span>
              <span class="text-dimmed">{{ profile.commissionRate }}%</span>
              <span class="text-primary">{{ profile.totalCommission }}</span>
              <span class="text-primary">{{ profile.availableAmount }}</span>
              <span class="text-dimmed">{{ profile.invitedUserCount }}</span>
              <span class="text-dimmed">{{ profile.visits }}</span>
            </div>
          </template>

          <template v-else-if="affiliateTab === 'commissions'">
            <div class="grid grid-cols-[1.2fr_0.9fr_0.75fr_0.75fr_0.75fr_0.9fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
              <span>邀请用户</span>
              <span>商品类型</span>
              <span>订单金额</span>
              <span>佣金比例</span>
              <span>佣金</span>
              <span>时间</span>
            </div>
            <div
              v-for="item in affiliates?.commissions || []"
              :key="item.id"
              class="grid grid-cols-[1.2fr_0.9fr_0.75fr_0.75fr_0.75fr_0.9fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
            >
              <span class="truncate text-highlighted">{{ item.referredEmail || item.referredUserId }}</span>
              <span class="text-toned">{{ productTypeLabel(item.productType) }}</span>
              <span class="text-dimmed">{{ item.orderAmount }}</span>
              <span class="text-dimmed">{{ item.commissionRate }}%</span>
              <span class="text-primary">{{ item.commissionAmount }}</span>
              <span class="text-dimmed">{{ formatAdminDate(item.createdAt) }}</span>
            </div>
          </template>

          <template v-else>
            <div class="grid grid-cols-[1fr_0.8fr_0.75fr_1.2fr_0.9fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
              <span>用户 ID</span>
              <span>金额</span>
              <span>状态</span>
              <span>备注</span>
              <span>申请时间</span>
            </div>
            <div
              v-for="item in affiliates?.withdrawals || []"
              :key="item.id"
              class="grid grid-cols-[1fr_0.8fr_0.75fr_1.2fr_0.9fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
            >
              <span class="font-mono text-xs text-highlighted truncate">{{ item.userId }}</span>
              <span class="text-primary">{{ item.amount }}</span>
              <span class="text-toned">{{ withdrawalStatusLabel(item.status) }}</span>
              <span class="truncate text-toned">{{ item.note || '-' }}</span>
              <span class="text-dimmed">{{ formatAdminDate(item.createdAt) }}</span>
            </div>
          </template>
        </section>

        <section
          v-else-if="section === 'logs'"
          class="rounded-lg border border-default bg-default overflow-hidden"
        >
          <div class="flex items-center gap-1 p-2 border-b border-default">
            <UButton
              :color="logTab === 'login' ? 'primary' : 'neutral'"
              variant="soft"
              size="sm"
              @click="logTab = 'login'"
            >
              登录日志
            </UButton>
            <UButton
              :color="logTab === 'task' ? 'primary' : 'neutral'"
              variant="soft"
              size="sm"
              @click="logTab = 'task'"
            >
              任务日志
            </UButton>
          </div>

          <template v-if="logTab === 'login'">
            <div class="grid grid-cols-[1.2fr_0.5fr_0.8fr_1fr_1fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
              <span>邮箱</span>
              <span>结果</span>
              <span>IP</span>
              <span>消息</span>
              <span>时间</span>
            </div>
            <div
              v-for="log in loginLogs"
              :key="log.id"
              class="grid grid-cols-[1.2fr_0.5fr_0.8fr_1fr_1fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
            >
              <span class="truncate text-highlighted">{{ log.email }}</span>
              <UBadge
                :color="log.success ? 'success' : 'error'"
                variant="soft"
              >
                {{ log.success ? '成功' : '失败' }}
              </UBadge>
              <span class="truncate text-dimmed">{{ log.ip }}</span>
              <span class="truncate text-toned">{{ log.message }}</span>
              <span class="text-dimmed">{{ new Date(log.createdAt).toLocaleString('zh-CN') }}</span>
            </div>
          </template>

          <template v-else>
            <div class="grid grid-cols-[1fr_0.7fr_1fr_1.2fr_1fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
              <span>动作</span>
              <span>状态</span>
              <span>渠道</span>
              <span>消息</span>
              <span>时间</span>
            </div>
            <div
              v-for="log in taskLogs"
              :key="log.id"
              class="grid grid-cols-[1fr_0.7fr_1fr_1.2fr_1fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
            >
              <span class="truncate text-highlighted">{{ log.action }}</span>
              <UBadge
                :color="log.status === 'succeeded' ? 'success' : log.status === 'failed' ? 'error' : 'warning'"
                variant="soft"
              >
                {{ log.status || '-' }}
              </UBadge>
              <span class="truncate text-toned">{{ taskLogChannel(log) }}</span>
              <span class="truncate text-toned">{{ log.message }}</span>
              <span class="text-dimmed">{{ new Date(log.createdAt).toLocaleString('zh-CN') }}</span>
            </div>
          </template>
        </section>

        <section
          v-else
          class="rounded-lg border border-default bg-default overflow-hidden"
        >
          <div class="grid grid-cols-[1fr_0.9fr_1.6fr_0.7fr_0.7fr_1fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
            <span>用户</span>
            <span>应用</span>
            <span>提示词</span>
            <span>状态</span>
            <span>产物</span>
            <span>时间</span>
          </div>
          <div
            v-for="job in generations"
            :key="job.id"
            class="grid grid-cols-[1fr_0.9fr_1.6fr_0.7fr_0.7fr_1fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
          >
            <span class="truncate text-highlighted">{{ job.userEmail }}</span>
            <span class="truncate text-toned">{{ job.appName || '-' }}</span>
            <span class="truncate text-dimmed">{{ job.prompt }}</span>
            <UBadge
              :color="job.status === 'succeeded' ? 'success' : job.status === 'failed' ? 'error' : 'warning'"
              variant="soft"
            >
              {{ job.status }}
            </UBadge>
            <span class="text-dimmed">{{ job.assets?.length ?? 0 }}</span>
            <span class="text-dimmed">{{ new Date(job.createdAt).toLocaleString('zh-CN') }}</span>
          </div>
        </section>
      </main>
    </div>

    <UModal
      v-model:open="userEditorOpen"
      :title="userForm.id ? '编辑用户' : '添加用户'"
      :ui="{ content: 'max-w-lg' }"
    >
      <template #body>
        <div class="space-y-3">
          <UFormField label="邮箱">
            <UInput
              v-model="userForm.email"
              placeholder="user@example.com"
            />
          </UFormField>
          <UFormField :label="userForm.id ? '新密码（留空不修改）' : '登录密码'">
            <UInput
              v-model="userForm.password"
              type="password"
              placeholder="至少 8 位"
            />
          </UFormField>
          <UFormField label="昵称">
            <UInput v-model="userForm.nickname" />
          </UFormField>
          <UFormField label="角色">
            <USelect
              v-model="userForm.role"
              :items="['user', 'admin']"
            />
          </UFormField>
          <UFormField label="状态">
            <USelect
              v-model="userForm.status"
              :items="['active', 'disabled']"
            />
          </UFormField>
          <UFormField label="会员等级">
            <USelect
              v-model="userForm.membershipLevel"
              :items="membershipOptions"
              value-key="value"
              label-key="label"
            />
          </UFormField>
          <UFormField label="积分">
            <UInput
              v-model.number="userForm.credits"
              type="number"
            />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton
            color="neutral"
            variant="soft"
            @click="userEditorOpen = false"
          >
            取消
          </UButton>
          <UButton @click="saveUser()">
            保存
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="balanceEditorOpen"
      title="调整用户余额"
      :ui="{ content: 'max-w-lg' }"
    >
      <template #body>
        <div class="space-y-3">
          <div class="rounded-lg border border-default bg-elevated/40 p-3 text-sm">
            <p class="text-dimmed">
              用户
            </p>
            <p class="font-medium text-highlighted truncate">
              {{ balanceForm.email }}
            </p>
            <p class="mt-2 text-dimmed">
              当前余额：<span class="font-semibold text-highlighted">{{ balanceForm.current }}</span>
            </p>
          </div>
          <UFormField label="调整方式">
            <USelect
              v-model="balanceForm.type"
              :items="[
                { label: '增加', value: 'increase' },
                { label: '减少', value: 'decrease' },
                { label: '覆盖', value: 'set' }
              ]"
              value-key="value"
              label-key="label"
            />
          </UFormField>
          <UFormField label="金额">
            <UInput
              v-model="balanceForm.amount"
              type="number"
              step="0.01"
              min="0"
              placeholder="0.00"
            />
          </UFormField>
          <UFormField label="备注">
            <UTextarea
              v-model="balanceForm.note"
              :rows="3"
              placeholder="例如：后台充值、退款、活动赠送"
            />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton
            color="neutral"
            variant="soft"
            @click="balanceEditorOpen = false"
          >
            取消
          </UButton>
          <UButton @click="saveBalanceAdjustment()">
            确认调整
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="providerEditorOpen"
      :title="providerForm.id ? '编辑接口' : '新建接口'"
      :ui="{ content: 'max-w-lg' }"
    >
      <template #body>
        <div class="space-y-3">
          <UFormField label="渠道名称">
            <UInput v-model="providerForm.name" />
          </UFormField>
          <UFormField label="接口分类">
            <select
              v-model="providerForm.category"
              class="w-full rounded-md border border-default bg-default px-3 py-2 text-sm text-highlighted outline-none transition-colors focus:border-primary"
            >
              <option
                v-for="option in providerCategoryOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </option>
            </select>
          </UFormField>
          <UFormField label="接口类型">
            <UInput
              model-value="OpenAI 兼容"
              disabled
            />
          </UFormField>
          <UFormField label="Base URL">
            <UInput v-model="providerForm.baseUrl" />
          </UFormField>
          <UFormField label="API Key">
            <UInput
              v-model="providerForm.apiKey"
              type="password"
              :placeholder="providerForm.id ? '留空则不修改已保存密钥' : '请输入 API Key'"
            />
          </UFormField>
          <UFormField label="模型">
            <div class="flex gap-2">
              <USelect
                v-if="providerModels.length"
                v-model="providerForm.model"
                :items="providerModels"
                class="flex-1"
              />
              <UInput
                v-else
                v-model="providerForm.model"
                class="flex-1"
                placeholder="可手动填写模型名称"
              />
              <UButton
                color="neutral"
                variant="soft"
                :loading="providerModelsLoading"
                @click="fetchProviderModels()"
              >
                获取模型
              </UButton>
            </div>
            <p
              class="mt-1 text-xs text-dimmed"
            >
              首次获取会先保存一条未启用接口配置，获取成功后可从下拉框选择模型再保存启用。
            </p>
          </UFormField>
          <UFormField label="排序">
            <UInput
              v-model.number="providerForm.sortOrder"
              type="number"
            />
          </UFormField>
          <UCheckbox
            v-model="providerForm.enabled"
            label="启用接口"
          />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton
            color="neutral"
            variant="soft"
            @click="providerEditorOpen = false"
          >
            取消
          </UButton>
          <UButton @click="saveProvider()">
            保存
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="providerDeleteOpen"
      title="删除接口配置"
      :ui="{ content: 'max-w-md' }"
    >
      <template #body>
        <div class="space-y-3">
          <div class="rounded-lg border border-error/30 bg-error/5 p-3 text-sm text-toned">
            确认删除接口配置
            <span class="font-semibold text-highlighted">「{{ providerToDelete?.name }}」</span>
            吗？删除后，已绑定该接口的应用可能无法继续调用。
          </div>
          <div class="text-xs text-dimmed">
            分类：{{ providerCategoryLabel(providerToDelete?.category || '') }}，模型：{{ providerToDelete?.model || '-' }}
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton
            color="neutral"
            variant="soft"
            @click="providerDeleteOpen = false"
          >
            取消
          </UButton>
          <UButton
            color="error"
            @click="confirmDeleteProvider"
          >
            确认删除
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="appEditorOpen"
      :title="appForm.id ? '编辑应用' : '新建应用'"
      :ui="{ content: 'max-w-3xl' }"
    >
      <template #body>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <UFormField label="应用类型">
            <select
              v-model="appForm.appType"
              class="w-full rounded-md border border-default bg-default px-3 py-2 text-sm text-highlighted outline-none transition-colors focus:border-primary"
            >
              <option
                v-for="option in appTypeOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </option>
            </select>
          </UFormField>
          <UFormField
            label="接口配置"
          >
            <select
              v-model="appForm.providerId"
              class="w-full rounded-md border border-default bg-default px-3 py-2 text-sm text-highlighted outline-none transition-colors focus:border-primary"
            >
              <option
                v-for="provider in appProviderOptionsByType"
                :key="provider.value"
                :value="provider.value"
              >
                {{ provider.label }}
              </option>
            </select>
          </UFormField>
          <UFormField label="应用标识">
            <UInput
              v-model="appForm.code"
              placeholder="creative-image"
            />
          </UFormField>
          <UFormField label="应用名称">
            <UInput v-model="appForm.name" />
          </UFormField>
          <UFormField label="分类">
            <UInput v-model="appForm.category" />
          </UFormField>
          <UFormField label="排序">
            <UInput
              v-model.number="appForm.sortOrder"
              type="number"
            />
          </UFormField>
          <UFormField label="图标">
            <UInput
              v-model="appForm.icon"
              placeholder="i-lucide-sparkles"
            />
          </UFormField>
          <UFormField label="图标样式">
            <UInput
              v-model="appForm.iconColor"
              placeholder="bg-emerald-100 text-emerald-600"
            />
          </UFormField>
          <UFormField label="可见性">
            <USelect
              v-model="appForm.visibility"
              :items="['public', 'private']"
            />
          </UFormField>
          <UFormField label="状态">
            <USelect
              v-model="appForm.status"
              :items="['active', 'disabled']"
            />
          </UFormField>
          <UFormField label="Free 价格">
            <UInput
              v-model="appForm.priceFree"
              type="number"
              step="0.01"
            />
          </UFormField>
          <UFormField label="V1会员价格">
            <UInput
              v-model="appForm.priceV1"
              type="number"
              step="0.01"
            />
          </UFormField>
          <UFormField label="V2会员价格">
            <UInput
              v-model="appForm.priceV2"
              type="number"
              step="0.01"
            />
          </UFormField>
          <UFormField
            label="封面 URL"
            class="md:col-span-2"
          >
            <UInput v-model="appForm.coverUrl" />
          </UFormField>
          <UFormField
            label="描述"
            class="md:col-span-2"
          >
            <UTextarea
              v-model="appForm.description"
              :rows="3"
            />
          </UFormField>
          <UFormField
            label="提示词模板"
            class="md:col-span-2"
          >
            <UTextarea
              v-model="appForm.promptTemplate"
              :rows="3"
            />
          </UFormField>
          <UFormField label="输入 Schema JSON">
            <UTextarea
              v-model="appForm.inputSchemaText"
              :rows="8"
              class="font-mono text-xs"
            />
          </UFormField>
          <UFormField label="输出 Schema JSON">
            <UTextarea
              v-model="appForm.outputSchemaText"
              :rows="8"
              class="font-mono text-xs"
            />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton
            color="neutral"
            variant="soft"
            @click="appEditorOpen = false"
          >
            取消
          </UButton>
          <UButton @click="saveApp()">
            保存
          </UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
