<script setup lang="ts">
import type { ApiAffiliateDashboard, ApiAffiliateWithdrawal, ApiBalanceLog, ApiGeneration, ApiRedeemCodeResult, ApiUser } from '~/composables/useApi'

useHead({ title: '个人中心 - 摘星AI' })

const api = useApi()
const auth = useAuth()
const message = useMessage()
const route = useRoute()
const router = useRouter()
const activeTab = ref('profile')
const recordTab = ref('draw')
const rebateRecordTab = ref<'commissions' | 'withdrawals' | 'invites'>('commissions')
const cardCode = ref('')
const redeemingCode = ref(false)
const paymentOpen = ref(false)
const paymentMode = ref<'membership' | 'credits'>('credits')
const withdrawalOpen = ref(false)
const withdrawalForm = reactive({
  amount: '',
  note: ''
})
const accountForm = reactive({
  avatarUrl: '',
  nickname: '',
  signature: ''
})
const passwordForm = reactive({
  currentPassword: '',
  newPassword: '',
  confirmPassword: ''
})
const emailForm = reactive({
  email: '',
  code: '',
  currentPassword: ''
})
const passwordSaving = ref(false)
const emailSaving = ref(false)
const emailCodeSending = ref(false)
const emailCodeCountdown = ref(0)
let emailCodeTimer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  await auth.loadMe()
  await refreshAffiliate()
})

const { data: jobs, pending: jobsPending } = await useAsyncData(
  'profile-generations',
  () => auth.token.value ? api.get<ApiGeneration[]>('/generations?limit=20') : Promise.resolve([]),
  { default: () => [], watch: [auth.token] }
)

const { data: balanceLogs, pending: balancePending } = await useAsyncData(
  'profile-balance-logs',
  () => auth.token.value ? api.get<ApiBalanceLog[]>('/balance-logs?limit=20') : Promise.resolve([]),
  { default: () => [], watch: [auth.token] }
)

const { data: affiliate, pending: affiliatePending, refresh: refreshAffiliate } = await useAsyncData(
  'profile-affiliate',
  () => auth.token.value ? api.get<ApiAffiliateDashboard>('/affiliate/dashboard') : Promise.resolve(null),
  { default: () => null, watch: [auth.token] }
)

const displayName = computed(() => auth.user.value?.nickname || auth.user.value?.email?.split('@')[0] || '用户')
const avatarText = computed(() => displayName.value.slice(0, 1).toUpperCase())
const membershipName = computed(() => ({
  free: '未开通会员',
  v1: 'V1会员',
  v2: 'V2会员'
}[auth.user.value?.membershipLevel || 'free']))
const membershipBadge = computed(() => ({
  free: 'FREE',
  v1: 'V1',
  v2: 'V2'
}[auth.user.value?.membershipLevel || 'free']))

const tabs = [
  { label: '个人中心', value: 'profile', query: 'profile' },
  { label: '我的收藏', value: 'favorites', query: 'favorites' },
  { label: '账户信息', value: 'account', query: 'account' },
  { label: '账户安全', value: 'security', query: 'security' },
  { label: '推广领返佣', value: 'rebate', query: 'invite' }
]

const tabByQuery = computed(() => Object.fromEntries(tabs.map(tab => [tab.query, tab.value])))
const queryByTab = computed(() => Object.fromEntries(tabs.map(tab => [tab.value, tab.query])))
const normalizeProfileTab = (value: unknown) => {
  const raw = Array.isArray(value) ? value[0] : value
  if (typeof raw !== 'string') return 'profile'
  return tabByQuery.value[raw] || tabs.find(tab => tab.value === raw)?.value || 'profile'
}
const setActiveTab = (value: string) => {
  activeTab.value = value
  router.replace({
    path: '/profile',
    query: {
      ...route.query,
      tab: queryByTab.value[value] || value
    }
  })
}

watch(
  () => route.query.tab,
  (tab) => {
    activeTab.value = normalizeProfileTab(tab)
  },
  { immediate: true }
)

watch(
  () => auth.user.value,
  (user) => {
    accountForm.avatarUrl = user?.avatarUrl || ''
    accountForm.nickname = user?.nickname || user?.email?.split('@')[0] || ''
    accountForm.signature = user?.signature || '我是一个基于深度学习和自然语言处理技术的 AI 助手，旨在为用户提供高效、精准、个性化的智能服务。'
  },
  { immediate: true }
)

const recordTabs = [
  { label: '绘画消费', value: 'draw' },
  { label: '工作流扣费', value: 'workflow' },
  { label: '充值记录', value: 'recharge' },
  { label: '签到奖励', value: 'checkin' },
  { label: '邀请赠送', value: 'invite' }
]

const drawRows = computed(() => jobs.value.slice(0, 6))
const balanceRows = computed(() => balanceLogs.value.slice(0, 6))
const requestURL = useRequestURL()
const affiliateOrigin = computed(() => {
  if (import.meta.client) return window.location.origin
  return requestURL.origin.replace('://localhost:', '://localhost:')
})
const affiliateLink = computed(() => {
  const code = affiliate.value?.profile?.code || ''
  return code ? affiliateOrigin.value + '/?invite=' + code : '推广码加载中'
})
const canSendChangeEmailCode = computed(() => emailCodeCountdown.value === 0 && /\S+@\S+\.\S+/.test(emailForm.email))
const nicknameCount = computed(() => Array.from(accountForm.nickname || '').length)
const signatureCount = computed(() => Array.from(accountForm.signature || '').length)
const moneyText = (value?: string) => Number(value || 0).toFixed(2)
const productTypeLabel = (value: string) => ({
  credits: '积分充值',
  membership: 'VIP会员'
}[value] || value || '-')
const commissionStatusLabel = (value: string) => ({
  settled: '已结算',
  pending: '待结算',
  cancelled: '已取消'
}[value] || value || '-')
const withdrawalStatusLabel = (value: string) => ({
  pending: '提现中',
  paid: '已打款',
  rejected: '已驳回'
}[value] || value || '-')

const formatDate = (value?: string) => value ? new Date(value).toLocaleString('zh-CN') : '-'
const shortID = (value: string) => value.length > 18 ? value.slice(0, 18) + '...' : value
const logout = () => auth.logout()
const restoreDefaultAvatar = () => {
  accountForm.avatarUrl = ''
}
const saveAccountProfile = async () => {
  try {
    const user = await api.put<ApiUser>('/auth/profile', {
      avatarUrl: accountForm.avatarUrl,
      nickname: accountForm.nickname,
      signature: accountForm.signature
    })
    auth.user.value = user
    message.success('保存成功', '账户信息已更新')
  } catch (error) {
    message.error('保存失败', error instanceof Error ? error.message : '账户信息保存失败')
  }
}
const changePassword = async () => {
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    message.error('修改失败', '两次输入的新密码不一致')
    return
  }
  passwordSaving.value = true
  try {
    await api.put<{ ok: boolean }>('/auth/password', {
      currentPassword: passwordForm.currentPassword,
      newPassword: passwordForm.newPassword
    })
    Object.assign(passwordForm, {
      currentPassword: '',
      newPassword: '',
      confirmPassword: ''
    })
    message.success('修改成功', '下次登录请使用新密码')
  } catch (error) {
    message.error('修改失败', error instanceof Error ? error.message : '密码修改失败')
  } finally {
    passwordSaving.value = false
  }
}
const sendChangeEmailCode = async () => {
  if (!canSendChangeEmailCode.value) return
  emailCodeSending.value = true
  try {
    const res = await auth.sendCode(emailForm.email, 'change_email')
    emailCodeCountdown.value = Math.min(res.expiresIn, 60)
    if (emailCodeTimer) clearInterval(emailCodeTimer)
    emailCodeTimer = setInterval(() => {
      emailCodeCountdown.value -= 1
      if (emailCodeCountdown.value <= 0 && emailCodeTimer) {
        clearInterval(emailCodeTimer)
        emailCodeTimer = null
      }
    }, 1000)
    message.success('发送成功', '验证码已发送到新邮箱')
  } catch (error) {
    message.error('发送失败', error instanceof Error ? error.message : '验证码发送失败')
  } finally {
    emailCodeSending.value = false
  }
}
const changeEmail = async () => {
  emailSaving.value = true
  try {
    const user = await api.put<ApiUser>('/auth/email', {
      email: emailForm.email,
      code: emailForm.code,
      currentPassword: emailForm.currentPassword
    })
    auth.user.value = user
    Object.assign(emailForm, {
      email: '',
      code: '',
      currentPassword: ''
    })
    message.success('换绑成功', '登录邮箱已更新')
  } catch (error) {
    message.error('换绑失败', error instanceof Error ? error.message : '邮箱换绑失败')
  } finally {
    emailSaving.value = false
  }
}
const openPayment = (mode: 'membership' | 'credits') => {
  paymentMode.value = mode
  paymentOpen.value = true
}
const redeemCode = async () => {
  if (!cardCode.value.trim()) {
    message.error('兑换失败', '请输入兑换码')
    return
  }
  redeemingCode.value = true
  try {
    const result = await api.post<ApiRedeemCodeResult>('/redeem-codes/redeem', {
      code: cardCode.value.trim()
    })
    auth.user.value = result.user
    cardCode.value = ''
    await refreshNuxtData('profile-balance-logs')
    message.success('兑换成功', '已到账 ' + result.code.amount + ' 积分')
  } catch (error) {
    message.error('兑换失败', error instanceof Error ? error.message : '兑换码不可用')
  } finally {
    redeemingCode.value = false
  }
}
const copyAffiliateLink = async () => {
  if (!affiliateLink.value.startsWith('http')) return
  await navigator.clipboard.writeText(affiliateLink.value)
  message.success('复制成功', '专属邀请链接已复制')
}
const createPoster = () => {
  // 目前先保留为轻量入口，后续可接二维码海报生成服务。
  message.success('已生成', '二维码海报功能入口已准备')
}
const openWithdrawal = () => {
  withdrawalForm.amount = moneyText(affiliate.value?.availableAmount)
  withdrawalForm.note = ''
  withdrawalOpen.value = true
}
const submitWithdrawal = async () => {
  try {
    await api.post<ApiAffiliateWithdrawal>('/affiliate/withdrawals', {
      amount: String(withdrawalForm.amount),
      note: withdrawalForm.note
    })
    withdrawalOpen.value = false
    await refreshAffiliate()
    message.success('提交成功', '提现申请已进入审核')
  } catch (error) {
    message.error('提交失败', error instanceof Error ? error.message : '提现申请提交失败')
  }
}

onUnmounted(() => {
  if (emailCodeTimer) clearInterval(emailCodeTimer)
})
</script>

<template>
  <div class="max-w-[100rem] mx-auto px-4 sm:px-6 py-5">
    <UAlert
      v-if="!auth.token.value"
      color="warning"
      variant="soft"
      icon="i-lucide-lock"
      title="请先登录"
      description="登录后可以查看个人资产、资料和消费记录。"
    />

    <div
      v-else
      :class="activeTab === 'rebate' || activeTab === 'account' || activeTab === 'security' ? 'space-y-5' : 'grid grid-cols-1 2xl:grid-cols-[minmax(0,1fr)_22.5rem] gap-6'"
    >
      <main class="min-w-0 space-y-5">
        <div>
          <h1 class="text-xl font-bold text-highlighted">
            个人中心
          </h1>
          <p class="text-sm text-dimmed mt-1">
            账户资产、资料安全和邀请返佣
          </p>
        </div>

        <div class="border-b border-default">
          <div class="flex items-center gap-6 overflow-x-auto">
            <button
              v-for="tab in tabs"
              :key="tab.value"
              type="button"
              class="h-12 shrink-0 border-b-2 text-sm font-medium transition-colors"
              :class="activeTab === tab.value ? 'border-primary text-primary' : 'border-transparent text-toned hover:text-highlighted'"
              @click="setActiveTab(tab.value)"
            >
              {{ tab.label }}
            </button>
          </div>
        </div>

        <template v-if="activeTab === 'account'">
          <section class="rounded-lg border border-default bg-default p-6">
            <h2 class="text-lg font-bold text-highlighted">
              用户账户信息设置
            </h2>

            <div class="mt-8 rounded-lg border border-default p-5 sm:p-6 space-y-4">
              <div class="grid grid-cols-1 md:grid-cols-[6rem_minmax(0,1fr)] gap-3 md:items-center">
                <label class="text-sm font-semibold text-highlighted">我的头像</label>
                <div class="flex flex-col sm:flex-row sm:items-center gap-4">
                  <div class="h-14 w-14 shrink-0 rounded-lg overflow-hidden bg-elevated ring-1 ring-default">
                    <img
                      v-if="accountForm.avatarUrl"
                      :src="accountForm.avatarUrl"
                      :alt="displayName"
                      class="h-full w-full object-cover"
                    >
                    <div
                      v-else
                      class="h-full w-full grid place-items-center bg-gradient-to-br from-sky-100 via-indigo-100 to-amber-100 text-xl font-bold text-primary"
                    >
                      {{ avatarText }}
                    </div>
                  </div>
                  <UInput
                    v-model="accountForm.avatarUrl"
                    class="min-w-0 flex-1"
                    placeholder="/uploads/media/2026-06-27/avatar.jpg"
                  />
                  <button
                    type="button"
                    class="shrink-0 text-sm text-primary hover:text-primary/80"
                    @click="restoreDefaultAvatar"
                  >
                    恢复默认
                  </button>
                </div>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-[6rem_minmax(0,1fr)] gap-3 md:items-center">
                <label class="text-sm font-semibold text-highlighted">用户名称</label>
                <div class="flex flex-col sm:flex-row sm:items-center gap-3">
                  <div class="relative min-w-0 flex-1">
                    <UInput
                      v-model="accountForm.nickname"
                      maxlength="12"
                      class="w-full"
                    />
                    <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-dimmed">
                      {{ nicknameCount }} / 12
                    </span>
                  </div>
                  <button
                    type="button"
                    class="shrink-0 text-sm text-primary hover:text-primary/80"
                    @click="saveAccountProfile"
                  >
                    修改
                  </button>
                </div>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-[6rem_minmax(0,1fr)] gap-3">
                <label class="pt-2 text-sm font-semibold text-highlighted">个性签名</label>
                <div class="flex flex-col sm:flex-row gap-3">
                  <div class="relative min-w-0 flex-1">
                    <UTextarea
                      v-model="accountForm.signature"
                      maxlength="128"
                      :rows="4"
                      class="w-full"
                    />
                    <span class="absolute bottom-3 right-3 text-xs text-dimmed">
                      {{ signatureCount }} / 128
                    </span>
                  </div>
                  <button
                    type="button"
                    class="shrink-0 self-start pt-2 text-sm text-primary hover:text-primary/80"
                    @click="saveAccountProfile"
                  >
                    修改
                  </button>
                </div>
              </div>
            </div>

            <div class="mt-4 grid grid-cols-1 lg:grid-cols-2 gap-4">
              <div class="rounded-lg border border-default p-5">
                <div class="grid grid-cols-[2.25rem_1fr] gap-3 items-center">
                  <span class="h-8 w-8 rounded-full bg-orange-500 text-white grid place-items-center">
                    <UIcon
                      name="i-lucide-mail"
                      class="h-4 w-4"
                    />
                  </span>
                  <div class="flex flex-col sm:flex-row sm:items-center gap-3">
                    <span class="w-16 shrink-0 text-sm font-semibold text-highlighted">邮箱</span>
                    <UInput
                      :model-value="auth.user.value?.email || ''"
                      disabled
                      class="min-w-0 flex-1"
                    />
                    <button
                      type="button"
                      class="shrink-0 text-sm text-primary/60"
                    >
                      修改
                    </button>
                  </div>
                </div>
              </div>

              <div class="rounded-lg border border-default p-5">
                <div class="grid grid-cols-[2.25rem_1fr] gap-3 items-center">
                  <span class="h-8 w-8 rounded-full bg-blue-500 text-white grid place-items-center">
                    <UIcon
                      name="i-lucide-bell"
                      class="h-4 w-4"
                    />
                  </span>
                  <div class="flex flex-col sm:flex-row sm:items-center gap-3">
                    <span class="w-16 shrink-0 text-sm font-semibold text-highlighted">QQ</span>
                    <span class="min-w-0 flex-1 text-sm font-semibold text-highlighted sm:text-right">未绑定 QQ</span>
                    <button
                      type="button"
                      class="shrink-0 text-sm text-primary/60"
                    >
                      绑定
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </template>

        <template v-else-if="activeTab === 'security'">
          <section class="rounded-lg border border-default bg-default p-6">
            <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <h2 class="text-lg font-bold text-highlighted">
                  账户安全设置
                </h2>
                <p class="mt-1 text-sm text-dimmed">
                  管理登录密码和绑定邮箱，保护账户资产与生成记录。
                </p>
              </div>
              <UBadge
                color="success"
                variant="soft"
                class="w-fit"
              >
                当前账户正常
              </UBadge>
            </div>

            <div class="mt-6 grid grid-cols-1 xl:grid-cols-2 gap-5">
              <div class="rounded-lg border border-default p-5">
                <div class="flex items-center gap-3">
                  <span class="h-10 w-10 rounded-full bg-primary/10 text-primary grid place-items-center">
                    <UIcon
                      name="i-lucide-lock-keyhole"
                      class="h-5 w-5"
                    />
                  </span>
                  <div>
                    <h3 class="font-semibold text-highlighted">
                      修改登录密码
                    </h3>
                    <p class="mt-1 text-xs text-dimmed">
                      修改后请使用新密码重新登录其他设备。
                    </p>
                  </div>
                </div>

                <div class="mt-5 space-y-4">
                  <UFormField label="当前密码">
                    <UInput
                      v-model="passwordForm.currentPassword"
                      type="password"
                      placeholder="请输入当前登录密码"
                    />
                  </UFormField>
                  <UFormField label="新密码">
                    <UInput
                      v-model="passwordForm.newPassword"
                      type="password"
                      placeholder="至少 8 位字符"
                    />
                  </UFormField>
                  <UFormField label="确认新密码">
                    <UInput
                      v-model="passwordForm.confirmPassword"
                      type="password"
                      placeholder="再次输入新密码"
                    />
                  </UFormField>
                  <div class="flex justify-end pt-2">
                    <UButton
                      icon="i-lucide-shield-check"
                      :loading="passwordSaving"
                      @click="changePassword"
                    >
                      保存新密码
                    </UButton>
                  </div>
                </div>
              </div>

              <div class="rounded-lg border border-default p-5">
                <div class="flex items-center gap-3">
                  <span class="h-10 w-10 rounded-full bg-orange-100 text-orange-500 grid place-items-center">
                    <UIcon
                      name="i-lucide-mail-check"
                      class="h-5 w-5"
                    />
                  </span>
                  <div>
                    <h3 class="font-semibold text-highlighted">
                      换绑邮箱号
                    </h3>
                    <p class="mt-1 text-xs text-dimmed">
                      当前邮箱：{{ auth.user.value?.email || '-' }}
                    </p>
                  </div>
                </div>

                <div class="mt-5 space-y-4">
                  <UFormField label="新邮箱">
                    <UInput
                      v-model="emailForm.email"
                      type="email"
                      placeholder="请输入新的邮箱地址"
                    />
                  </UFormField>
                  <UFormField label="邮箱验证码">
                    <div class="flex gap-2">
                      <UInput
                        v-model="emailForm.code"
                        class="min-w-0 flex-1"
                        placeholder="输入验证码"
                      />
                      <UButton
                        color="neutral"
                        variant="soft"
                        :loading="emailCodeSending"
                        :disabled="!canSendChangeEmailCode || emailCodeSending"
                        @click="sendChangeEmailCode"
                      >
                        {{ emailCodeCountdown > 0 ? `${emailCodeCountdown}s 后重发` : '发送验证码' }}
                      </UButton>
                    </div>
                  </UFormField>
                  <UFormField label="当前密码">
                    <UInput
                      v-model="emailForm.currentPassword"
                      type="password"
                      placeholder="请输入当前登录密码"
                    />
                  </UFormField>
                  <div class="flex justify-end pt-2">
                    <UButton
                      icon="i-lucide-refresh-cw"
                      :loading="emailSaving"
                      @click="changeEmail"
                    >
                      确认换绑
                    </UButton>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </template>

        <template v-else-if="activeTab !== 'rebate'">
          <section class="rounded-lg border border-default bg-default p-6">
            <h2 class="font-semibold text-highlighted">
              用户积分余额
            </h2>
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mt-6">
              <div class="rounded-lg border border-default p-5 min-h-32">
                <div class="flex items-center gap-2 text-primary text-sm font-medium">
                  <UIcon
                    name="i-lucide-zap"
                    class="h-5 w-5 text-warning"
                  />
                  积分余额
                </div>
                <div class="mt-5 flex items-end gap-3">
                  <span class="text-3xl font-bold text-highlighted">{{ auth.user.value?.balance || '0' }}</span>
                  <button
                    type="button"
                    class="text-sm text-muted hover:text-primary mb-1"
                    @click="openPayment('credits')"
                  >
                    购买更多 ->
                  </button>
                </div>
              </div>

              <div class="rounded-lg border border-default p-5 min-h-32">
                <div class="flex items-center gap-2 text-primary text-sm font-medium">
                  <UIcon
                    name="i-lucide-crown"
                    class="h-5 w-5 text-warning"
                  />
                  {{ membershipName }}
                </div>
                <div class="mt-5 flex flex-wrap items-center gap-2 text-sm">
                  <span class="font-medium text-highlighted">会员过期时间:</span>
                  <span class="text-error font-semibold">{{ membershipName }}</span>
                  <UButton
                    size="xs"
                    color="primary"
                    @click="openPayment('membership')"
                  >
                    开通会员
                  </UButton>
                </div>
              </div>

              <div class="rounded-lg border border-default p-5 min-h-32">
                <div class="flex items-center gap-2 text-primary text-sm font-medium">
                  <UIcon
                    name="i-lucide-credit-card"
                    class="h-4 w-4"
                  />
                  卡密充值
                </div>
                <div class="mt-5 flex gap-2">
                  <UInput
                    v-model="cardCode"
                    class="flex-1"
                    placeholder="请粘贴或填写您的兑换码"
                  />
                  <UButton
                    :loading="redeemingCode"
                    @click="redeemCode"
                  >
                    兑换
                  </UButton>
                </div>
              </div>
            </div>
          </section>

          <section class="rounded-lg border border-default bg-default p-6">
            <div class="border-b border-default">
              <div class="flex items-center gap-8 overflow-x-auto">
                <button
                  v-for="tab in recordTabs"
                  :key="tab.value"
                  type="button"
                  class="h-11 shrink-0 border-b-2 text-sm transition-colors"
                  :class="recordTab === tab.value ? 'border-primary text-primary' : 'border-transparent text-toned hover:text-highlighted'"
                  @click="recordTab = tab.value"
                >
                  {{ tab.label }}
                </button>
              </div>
            </div>

            <div
              v-if="recordTab === 'draw'"
              class="mt-3 overflow-x-auto"
            >
              <div class="min-w-[46rem]">
                <div class="grid grid-cols-[1fr_1fr_1fr_1fr_1fr_1.1fr] bg-elevated px-3 py-3 text-sm font-medium text-toned">
                  <span>绘画编号</span>
                  <span>模型</span>
                  <span>版本</span>
                  <span>类型</span>
                  <span>绘画额度</span>
                  <span>时间</span>
                </div>
                <div
                  v-if="jobsPending"
                  class="p-4 space-y-3"
                >
                  <USkeleton
                    v-for="item in 3"
                    :key="item"
                    class="h-9"
                  />
                </div>
                <div
                  v-else-if="drawRows.length === 0"
                  class="px-3 py-8 text-center text-sm text-dimmed"
                >
                  暂无绘画消费记录
                </div>
                <div
                  v-for="job in drawRows"
                  v-else
                  :key="job.id"
                  class="grid grid-cols-[1fr_1fr_1fr_1fr_1fr_1.1fr] border-b border-default px-3 py-3 text-sm items-center"
                >
                  <span class="truncate text-highlighted">{{ shortID(job.id) }}</span>
                  <span class="truncate text-toned">{{ job.model || '-' }}</span>
                  <span class="text-toned">{{ String(job.params?.quality || '1K') }}</span>
                  <span class="text-toned">{{ job.appName || '标准画质' }}</span>
                  <span class="text-primary">{{ job.status === 'succeeded' ? '0' : '-' }}</span>
                  <span class="text-dimmed">{{ formatDate(job.createdAt) }}</span>
                </div>
              </div>
            </div>

            <div
              v-else
              class="mt-3 overflow-x-auto"
            >
              <div class="min-w-[42rem]">
                <div class="grid grid-cols-[1fr_1fr_1fr_1fr_1.2fr] bg-elevated px-3 py-3 text-sm font-medium text-toned">
                  <span>类型</span>
                  <span>金额</span>
                  <span>变动前</span>
                  <span>变动后</span>
                  <span>时间</span>
                </div>
                <div
                  v-if="balancePending"
                  class="p-4 space-y-3"
                >
                  <USkeleton
                    v-for="item in 3"
                    :key="item"
                    class="h-9"
                  />
                </div>
                <div
                  v-else-if="balanceRows.length === 0"
                  class="px-3 py-8 text-center text-sm text-dimmed"
                >
                  暂无记录
                </div>
                <div
                  v-for="log in balanceRows"
                  v-else
                  :key="log.id"
                  class="grid grid-cols-[1fr_1fr_1fr_1fr_1.2fr] border-b border-default px-3 py-3 text-sm items-center"
                >
                  <span class="text-highlighted">{{ log.note || log.changeType }}</span>
                  <span class="text-primary">{{ log.amount }}</span>
                  <span class="text-toned">{{ log.balanceBefore }}</span>
                  <span class="text-highlighted">{{ log.balanceAfter }}</span>
                  <span class="text-dimmed">{{ formatDate(log.createdAt) }}</span>
                </div>
              </div>
            </div>

            <div class="flex justify-end gap-2 mt-4">
              <UButton
                icon="i-lucide-chevron-left"
                color="neutral"
                variant="soft"
                size="xs"
                aria-label="上一页"
              />
              <UButton
                size="xs"
                variant="outline"
              >
                1
              </UButton>
              <UButton
                icon="i-lucide-chevron-right"
                color="neutral"
                variant="soft"
                size="xs"
                aria-label="下一页"
              />
            </div>
          </section>
        </template>

        <section
          v-else
          class="grid grid-cols-1 xl:grid-cols-[26.25rem_minmax(0,1fr)] gap-6"
        >
          <div class="space-y-5">
            <div class="relative overflow-hidden rounded-lg bg-gradient-to-br from-orange-400 to-orange-600 p-7 text-white min-h-[20.3rem]">
              <UIcon
                name="i-lucide-award"
                class="absolute right-7 top-7 h-8 w-8 text-white/80"
              />
              <p class="text-2xl font-bold">
                {{ affiliate?.profile.level || '初级代理' }}
              </p>
              <p class="mt-5 text-sm font-semibold">
                享受{{ affiliate?.profile.commissionRate || '20.00' }}%返佣
              </p>
              <p class="mt-12 text-4xl font-bold">
                {{ moneyText(affiliate?.totalCommission) }} 元
              </p>
              <div class="mt-8 grid grid-cols-2 gap-6">
                <div>
                  <p class="text-2xl font-bold">
                    {{ moneyText(affiliate?.availableAmount) }} 元
                  </p>
                  <p class="mt-1 text-sm font-semibold">
                    剩余可提现金额
                  </p>
                  <UButton
                    class="mt-2 bg-white/25 hover:bg-white/35 text-white"
                    size="sm"
                    :disabled="Number(affiliate?.availableAmount || 0) <= 0"
                    @click="openWithdrawal()"
                  >
                    立即提现
                  </UButton>
                </div>
                <div>
                  <p class="text-2xl font-bold">
                    {{ moneyText(affiliate?.withdrawingAmount) }} 元
                  </p>
                  <p class="mt-1 text-sm font-semibold">
                    提现中金额
                  </p>
                </div>
              </div>
            </div>

            <div class="rounded-lg border border-default bg-default p-5">
              <div class="space-y-0">
                <div class="flex items-center justify-between border-b border-default py-4 first:pt-0">
                  <span class="inline-flex items-center gap-3 text-sm font-semibold text-highlighted">
                    <UIcon name="i-lucide-shopping-bag" />购买订单数量
                  </span>
                  <span class="font-semibold text-highlighted">{{ affiliate?.paidOrderCount || 0 }}</span>
                </div>
                <div class="flex items-center justify-between border-b border-default py-4">
                  <span class="inline-flex items-center gap-3 text-sm font-semibold text-highlighted">
                    <UIcon name="i-lucide-link" />推广链接访问次数
                  </span>
                  <span class="font-semibold text-highlighted">{{ affiliate?.profile.visits || 0 }}</span>
                </div>
                <div class="flex items-center justify-between py-4 last:pb-0">
                  <span class="inline-flex items-center gap-3 text-sm font-semibold text-highlighted">
                    <UIcon name="i-lucide-user-plus" />邀请用户
                  </span>
                  <span class="font-semibold text-highlighted">{{ affiliate?.invitedUserCount || 0 }}</span>
                </div>
              </div>
            </div>

            <div class="rounded-lg border border-default bg-default p-5 space-y-8">
              <div>
                <h3 class="font-semibold text-highlighted">
                  欢迎加入代理
                </h3>
                <p class="mt-4 text-sm leading-7 text-toned">
                  初级代理提现需提供真实截图，需满足邀请用户数满 10 人以上才可提现。
                </p>
              </div>
              <div>
                <h3 class="font-semibold text-highlighted">
                  如何升级代理
                </h3>
                <p class="mt-4 text-sm leading-7 text-toned">
                  满足相应条件可以联系管理员升级代理等级，获得更高返佣比例。
                </p>
              </div>
            </div>
          </div>

          <div class="min-w-0 space-y-5">
            <div class="rounded-lg border border-default bg-default p-5">
              <div class="flex flex-col lg:flex-row lg:items-center gap-3">
                <label class="shrink-0 text-sm font-semibold text-highlighted">推荐链接:</label>
                <UInput
                  :model-value="affiliateLink"
                  readonly
                  class="flex-1"
                />
                <div class="flex flex-wrap gap-3">
                  <UButton
                    icon="i-lucide-qr-code"
                    @click="createPoster()"
                  >
                    生成二维码海报
                  </UButton>
                  <UButton
                    icon="i-lucide-copy"
                    @click="copyAffiliateLink()"
                  >
                    复制专属邀请链接
                  </UButton>
                </div>
              </div>
            </div>

            <div class="rounded-lg border border-default bg-default overflow-hidden min-h-[26rem]">
              <div class="grid grid-cols-3 border-b border-default">
                <button
                  type="button"
                  class="h-14 border-b-2 text-sm font-medium transition-colors"
                  :class="rebateRecordTab === 'commissions' ? 'border-primary text-primary' : 'border-transparent text-toned hover:text-highlighted'"
                  @click="rebateRecordTab = 'commissions'"
                >
                  推介记录
                </button>
                <button
                  type="button"
                  class="h-14 border-b-2 text-sm font-medium transition-colors"
                  :class="rebateRecordTab === 'withdrawals' ? 'border-primary text-primary' : 'border-transparent text-toned hover:text-highlighted'"
                  @click="rebateRecordTab = 'withdrawals'"
                >
                  提现记录
                </button>
                <button
                  type="button"
                  class="h-14 border-b-2 text-sm font-medium transition-colors"
                  :class="rebateRecordTab === 'invites' ? 'border-primary text-primary' : 'border-transparent text-toned hover:text-highlighted'"
                  @click="rebateRecordTab = 'invites'"
                >
                  邀请用户
                </button>
              </div>

              <div
                v-if="affiliatePending"
                class="p-5 space-y-3"
              >
                <USkeleton
                  v-for="item in 4"
                  :key="item"
                  class="h-10"
                />
              </div>

              <div
                v-else-if="rebateRecordTab === 'commissions'"
                class="overflow-x-auto"
              >
                <div class="min-w-[54rem]">
                  <div class="grid grid-cols-[1fr_1fr_0.8fr_0.8fr_0.8fr_1.15fr] bg-elevated px-4 py-3 text-sm font-semibold text-highlighted">
                    <span>订单金额</span>
                    <span>商品类型</span>
                    <span>状态</span>
                    <span>佣金比例</span>
                    <span>佣金</span>
                    <span>订购时间</span>
                  </div>
                  <div
                    v-if="!affiliate?.commissions?.length"
                    class="grid min-h-[18rem] place-items-center text-sm text-dimmed"
                  >
                    <div class="text-center">
                      <UIcon
                        name="i-lucide-inbox"
                        class="mx-auto mb-2 h-9 w-9 text-dimmed"
                      />
                      无数据
                    </div>
                  </div>
                  <div
                    v-for="item in affiliate?.commissions || []"
                    v-else
                    :key="item.id"
                    class="grid grid-cols-[1fr_1fr_0.8fr_0.8fr_0.8fr_1.15fr] border-b border-default px-4 py-3 text-sm items-center"
                  >
                    <span class="text-highlighted">{{ moneyText(item.orderAmount) }} 元</span>
                    <span class="text-toned">{{ productTypeLabel(item.productType) }}</span>
                    <span class="text-toned">{{ commissionStatusLabel(item.status) }}</span>
                    <span class="text-toned">{{ item.commissionRate }}%</span>
                    <span class="text-primary">{{ moneyText(item.commissionAmount) }} 元</span>
                    <span class="text-dimmed">{{ formatDate(item.createdAt) }}</span>
                  </div>
                </div>
              </div>

              <div
                v-else-if="rebateRecordTab === 'withdrawals'"
                class="overflow-x-auto"
              >
                <div class="min-w-[42rem]">
                  <div class="grid grid-cols-[1fr_1fr_1.3fr_1.2fr] bg-elevated px-4 py-3 text-sm font-semibold text-highlighted">
                    <span>提现金额</span>
                    <span>状态</span>
                    <span>备注</span>
                    <span>申请时间</span>
                  </div>
                  <div
                    v-if="!affiliate?.withdrawals?.length"
                    class="grid min-h-[18rem] place-items-center text-sm text-dimmed"
                  >
                    无数据
                  </div>
                  <div
                    v-for="item in affiliate?.withdrawals || []"
                    v-else
                    :key="item.id"
                    class="grid grid-cols-[1fr_1fr_1.3fr_1.2fr] border-b border-default px-4 py-3 text-sm items-center"
                  >
                    <span class="text-primary">{{ moneyText(item.amount) }} 元</span>
                    <span class="text-toned">{{ withdrawalStatusLabel(item.status) }}</span>
                    <span class="truncate text-toned">{{ item.note || '-' }}</span>
                    <span class="text-dimmed">{{ formatDate(item.createdAt) }}</span>
                  </div>
                </div>
              </div>

              <div
                v-else
                class="overflow-x-auto"
              >
                <div class="min-w-[38rem]">
                  <div class="grid grid-cols-[1.3fr_1fr_1.1fr] bg-elevated px-4 py-3 text-sm font-semibold text-highlighted">
                    <span>邮箱</span>
                    <span>昵称</span>
                    <span>注册时间</span>
                  </div>
                  <div
                    v-if="!affiliate?.inviteUsers?.length"
                    class="grid min-h-[18rem] place-items-center text-sm text-dimmed"
                  >
                    无数据
                  </div>
                  <div
                    v-for="item in affiliate?.inviteUsers || []"
                    v-else
                    :key="item.id"
                    class="grid grid-cols-[1.3fr_1fr_1.1fr] border-b border-default px-4 py-3 text-sm items-center"
                  >
                    <span class="truncate text-highlighted">{{ item.email }}</span>
                    <span class="truncate text-toned">{{ item.nickname || '-' }}</span>
                    <span class="text-dimmed">{{ formatDate(item.createdAt) }}</span>
                  </div>
                </div>
              </div>

              <div class="flex justify-end gap-2 border-t border-default p-4">
                <UButton
                  icon="i-lucide-chevron-left"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  aria-label="上一页"
                />
                <UButton
                  size="xs"
                  variant="outline"
                >
                  1
                </UButton>
                <UButton
                  icon="i-lucide-chevron-right"
                  color="neutral"
                  variant="soft"
                  size="xs"
                  aria-label="下一页"
                />
              </div>
            </div>
          </div>
        </section>
      </main>

      <aside
        v-if="activeTab !== 'rebate' && activeTab !== 'account' && activeTab !== 'security'"
        class="rounded-lg border border-default bg-default p-5 h-fit"
      >
        <div class="flex items-center justify-between">
          <h2 class="text-2xl font-bold text-primary">
            我的信息
          </h2>
          <div class="flex items-center gap-3 text-sm">
            <UButton
              color="neutral"
              variant="ghost"
              size="sm"
            >
              编辑资料
            </UButton>
            <UButton
              color="error"
              variant="soft"
              size="sm"
              @click="logout"
            >
              退出登录
            </UButton>
          </div>
        </div>

        <div class="mt-14 flex flex-col items-center text-center">
          <div class="h-38 w-38 rounded-lg overflow-hidden bg-elevated ring-1 ring-default">
            <img
              v-if="auth.user.value?.avatarUrl"
              :src="auth.user.value.avatarUrl"
              :alt="displayName"
              class="h-full w-full object-cover"
            >
            <div
              v-else
              class="h-full w-full grid place-items-center bg-gradient-to-br from-sky-100 via-indigo-100 to-amber-100 text-5xl font-bold text-primary"
            >
              {{ avatarText }}
            </div>
          </div>

          <h3 class="mt-6 text-lg font-bold text-highlighted">
            {{ displayName }}
          </h3>
          <p class="mt-3 text-sm text-primary/70">
            {{ auth.user.value?.email }}
          </p>
          <UBadge
            class="mt-3"
            color="neutral"
            variant="soft"
          >
            {{ membershipBadge }}
          </UBadge>

          <div class="mt-4 flex items-center gap-2">
            <span class="h-6 w-6 rounded-full bg-orange-500 text-white grid place-items-center">
              <UIcon
                name="i-lucide-mail"
                class="h-3.5 w-3.5"
              />
            </span>
            <span class="h-6 w-6 rounded-full bg-blue-500 text-white grid place-items-center">
              <UIcon
                name="i-lucide-bell"
                class="h-3.5 w-3.5"
              />
            </span>
          </div>
        </div>

        <p class="mt-9 text-sm leading-7 text-highlighted">
          我是一个基于深度学习和自然语言处理技术的 AI 助手，旨在为用户提供高效、精准、个性化的智能服务。
        </p>

        <div class="mt-6 space-y-3">
          <div class="rounded-lg border border-default bg-elevated/60 p-3 flex items-center gap-3">
            <span class="h-8 w-8 rounded-full bg-orange-100 text-orange-500 grid place-items-center">
              <UIcon name="i-lucide-mail" />
            </span>
            <div>
              <p class="text-xs text-dimmed">
                邮箱状态
              </p>
              <p class="font-semibold text-highlighted">
                已绑定
              </p>
            </div>
          </div>
          <div class="rounded-lg border border-default bg-elevated/60 p-3 flex items-center gap-3">
            <span class="h-8 w-8 rounded-full bg-blue-100 text-blue-500 grid place-items-center">
              <UIcon name="i-lucide-bell" />
            </span>
            <div>
              <p class="text-xs text-dimmed">
                QQ绑定
              </p>
              <p class="font-semibold text-highlighted">
                未绑定
              </p>
            </div>
          </div>
          <div class="rounded-lg border border-default bg-elevated/60 p-3 flex items-center gap-3">
            <span class="h-8 w-8 rounded-full bg-green-100 text-green-500 grid place-items-center">
              <UIcon name="i-lucide-calendar-days" />
            </span>
            <div>
              <p class="text-xs text-dimmed">
                注册时间
              </p>
              <p class="font-semibold text-highlighted">
                {{ formatDate(auth.user.value?.createdAt) }}
              </p>
            </div>
          </div>
          <div class="rounded-lg border border-default bg-elevated/60 p-3 max-w-40">
            <div class="flex items-center gap-2 text-xs text-dimmed">
              <span class="h-6 w-6 rounded-full bg-primary/10 text-primary grid place-items-center">
                <UIcon
                  name="i-lucide-zap"
                  class="h-3.5 w-3.5"
                />
              </span>
              积分余额
            </div>
            <p class="mt-4 text-xl font-bold text-highlighted">
              {{ auth.user.value?.balance || 0 }}
            </p>
          </div>
        </div>
      </aside>
    </div>

    <PaymentModal
      v-model:open="paymentOpen"
      :initial-mode="paymentMode"
    />

    <UModal
      v-model:open="withdrawalOpen"
      title="申请提现"
      :ui="{ content: 'max-w-md' }"
    >
      <template #body>
        <div class="space-y-3">
          <div class="rounded-lg border border-default bg-elevated/50 p-3 text-sm text-toned">
            当前可提现金额：<span class="font-semibold text-highlighted">{{ moneyText(affiliate?.availableAmount) }} 元</span>
          </div>
          <UFormField label="提现金额">
            <UInput
              v-model="withdrawalForm.amount"
              type="number"
              step="0.01"
              min="0"
              placeholder="请输入提现金额"
            />
          </UFormField>
          <UFormField label="备注">
            <UTextarea
              v-model="withdrawalForm.note"
              :rows="3"
              placeholder="请输入收款方式或备注"
            />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton
            color="neutral"
            variant="soft"
            @click="withdrawalOpen = false"
          >
            取消
          </UButton>
          <UButton @click="submitWithdrawal()">
            提交申请
          </UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
