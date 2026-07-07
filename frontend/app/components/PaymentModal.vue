<script setup lang="ts">
import type { ApiPaymentOrderResult, ApiPaymentPlan, ApiPaymentPlans } from '~/composables/useApi'

const open = defineModel<boolean>('open', { default: false })
const props = withDefaults(defineProps<{ initialMode?: 'membership' | 'credits' }>(), {
  initialMode: 'credits'
})

const api = useApi()
const message = useMessage()
const mode = ref<'membership' | 'credits'>(props.initialMode)
const loadingPlan = ref('')
const reloadingOrder = ref(false)
const selectedPayType = ref('alipay')
const currentStep = ref<'plans' | 'pay'>('plans')
const currentTab = ref<'membership' | 'credits'>(props.initialMode)
const currentOrder = ref<ApiPaymentOrderResult | null>(null)
const now = ref(Date.now())
let countdownTimer: ReturnType<typeof setInterval> | null = null

watch(() => props.initialMode, (value) => {
  mode.value = value
  currentTab.value = value
  currentStep.value = 'plans'
  currentOrder.value = null
})

watch(open, (value) => {
  if (!value) {
    currentStep.value = 'plans'
    currentOrder.value = null
  }
})

const startCountdown = () => {
  now.value = Date.now()
  if (countdownTimer) return
  countdownTimer = setInterval(() => {
    now.value = Date.now()
  }, 1000)
}

const stopCountdown = () => {
  if (!countdownTimer) return
  clearInterval(countdownTimer)
  countdownTimer = null
}

onBeforeUnmount(stopCountdown)

watch(selectedPayType, async (value, previous) => {
  if (currentStep.value !== 'pay' || !currentOrder.value || value === previous || reloadingOrder.value) {
    return
  }
  await recreateOrder()
})

const { data: plans } = await useAsyncData(
  'payment-plans',
  () => api.get<ApiPaymentPlans>('/pay/plans'),
  {
    default: () => ({ credits: [], membership: [] }),
    lazy: true
  }
)

const { data: paymentSetting } = await useAsyncData(
  'public-payment-setting',
  async () => {
    const settings = await api.get<{ key: string, value: Record<string, unknown> }[]>('/public/settings')
    return settings.find(item => item.key === 'payment')?.value as { channels?: string[] } || { channels: ['alipay'] }
  },
  { default: () => ({ channels: ['alipay'] }) }
)

const allowedChannels = computed(() => paymentSetting.value.channels?.length ? paymentSetting.value.channels : ['alipay'])
const availablePayTypes = computed(() => [
  { code: 'alipay', name: '支付宝支付', desc: '打开支付宝收银台完成付款', icon: 'i-simple-icons-alipay' },
  { code: 'wxpay', name: '微信支付', desc: '扫码完成真实付款', icon: 'i-simple-icons-wechat' },
  { code: 'qqpay', name: 'QQ 钱包', desc: '使用 QQ 钱包完成付款', icon: 'i-lucide-message-circle' }
].filter(item => allowedChannels.value.includes(item.code)))

const visiblePlans = computed(() => mode.value === 'credits' ? plans.value.credits : plans.value.membership)
const title = computed(() => mode.value === 'credits' ? '积分充值' : 'VIP会员开通')
const subtitle = computed(() => mode.value === 'credits' ? '充值积分后可继续使用绘画、对话和应用工具' : '开通会员后可享受更优惠的产品价格')
const currentPlan = computed(() => currentOrder.value?.order ? visiblePlans.value.find(item => item.code === currentOrder.value?.order.planCode) || null : null)
const tabOffset = computed(() => currentTab.value === 'membership' ? 'translateX(0%)' : 'translateX(100%)')
const orderExpiresAt = computed(() => {
  const order = currentOrder.value?.order
  if (!order) return 0
  if (order.expiresAt) return new Date(order.expiresAt).getTime()
  return new Date(order.createdAt).getTime() + 15 * 60 * 1000
})
const remainingSeconds = computed(() => {
  if (!orderExpiresAt.value) return 15 * 60
  return Math.max(0, Math.ceil((orderExpiresAt.value - now.value) / 1000))
})
const remainingText = computed(() => {
  const minutes = Math.floor(remainingSeconds.value / 60)
  const seconds = remainingSeconds.value % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
})
const orderExpired = computed(() => currentStep.value === 'pay' && currentOrder.value?.order.status === 'pending' && remainingSeconds.value <= 0)

const unitText = (plan: ApiPaymentPlan) => {
  if (plan.orderType === 'membership') return plan.period || plan.desc
  if (!plan.credits) return ''
  return '约 ¥' + (Number(plan.amount) / plan.credits).toFixed(6) + ' / 积分'
}

const createOrder = async (plan: ApiPaymentPlan) => {
  if (loadingPlan.value) return
  loadingPlan.value = plan.code
  try {
    const result = await api.post<ApiPaymentOrderResult>('/pay/orders', {
      planCode: plan.code,
      payType: selectedPayType.value
    })
    currentOrder.value = result
    currentStep.value = 'pay'
    startCountdown()
  } catch (error) {
    message.error('创建订单失败', error instanceof Error ? error.message : '请检查支付配置')
  } finally {
    loadingPlan.value = ''
  }
}

const recreateOrder = async () => {
  const planCode = currentOrder.value?.order.planCode
  if (!planCode || reloadingOrder.value) return
  reloadingOrder.value = true
  try {
    const result = await api.post<ApiPaymentOrderResult>('/pay/orders', {
      planCode,
      payType: selectedPayType.value
    })
    currentOrder.value = result
    startCountdown()
  } catch (error) {
    message.error('切换支付方式失败', error instanceof Error ? error.message : '请稍后重试')
  } finally {
    reloadingOrder.value = false
  }
}

const openPayUrl = () => {
  if (orderExpired.value) {
    message.error('订单已超时', '请返回套餐重新生成订单')
    return
  }
  if (currentOrder.value?.payUrl) {
    window.open(currentOrder.value.payUrl, '_blank', 'noopener,noreferrer')
  }
}

const closeModal = () => {
  open.value = false
  currentStep.value = 'plans'
  currentOrder.value = null
  stopCountdown()
}
</script>

<template>
  <UModal
    v-model:open="open"
    :ui="{ content: 'max-w-[56rem] rounded-2xl' }"
  >
    <template #content>
      <div class="relative bg-default rounded-2xl px-6 sm:px-8 py-6 sm:py-8 overflow-hidden">
        <UButton
          icon="i-lucide-x"
          color="neutral"
          variant="ghost"
          size="sm"
          class="absolute right-4 top-4 rounded-full"
          aria-label="关闭"
          @click="closeModal"
        />

        <Transition
          mode="out-in"
          enter-active-class="transition duration-200 ease-out"
          enter-from-class="opacity-0 translate-y-2 scale-[0.98]"
          enter-to-class="opacity-100 translate-y-0 scale-100"
          leave-active-class="transition duration-150 ease-in"
          leave-from-class="opacity-100 translate-y-0 scale-100"
          leave-to-class="opacity-0 -translate-y-2 scale-[0.98]"
        >
          <div
            :key="currentStep"
            class="space-y-6"
          >
            <div class="text-center">
              <h2 class="text-2xl font-bold text-highlighted">
                {{ currentStep === 'plans' ? title : '订单支付' }}
              </h2>
              <p class="mt-3 text-sm font-medium text-dimmed flex items-center justify-center gap-2">
                <UIcon
                  name="i-lucide-zap"
                  class="h-4 w-4"
                />
                {{ currentStep === 'plans' ? subtitle : '完成付款后，系统收到平台回调会自动入账。' }}
              </p>
            </div>

            <template v-if="currentStep === 'plans'">
              <div class="mx-auto max-w-md rounded-full border border-default bg-default p-1">
                <div class="relative grid grid-cols-2">
                  <span
                    class="pointer-events-none absolute inset-y-1 left-1 w-[calc(50%-0.25rem)] rounded-full bg-primary/10 transition-transform duration-300 ease-out"
                    :style="{ transform: tabOffset }"
                  />
                  <button
                    type="button"
                    class="relative z-10 h-10 rounded-full text-sm font-semibold transition-colors flex items-center justify-center gap-2"
                    :class="currentTab === 'membership' ? 'text-primary' : 'text-muted hover:text-highlighted'"
                    @click="currentTab = 'membership'; mode = 'membership'"
                  >
                    <span class="flex items-center gap-2">
                      <UIcon
                        name="i-lucide-crown"
                        class="h-4 w-4"
                      />
                      会员购买
                    </span>
                  </button>
                  <button
                    type="button"
                    class="relative z-10 h-10 rounded-full text-sm font-semibold transition-colors flex items-center justify-center gap-2"
                    :class="currentTab === 'credits' ? 'text-primary' : 'text-muted hover:text-highlighted'"
                    @click="currentTab = 'credits'; mode = 'credits'"
                  >
                    <span class="flex items-center gap-2">
                      <UIcon
                        name="i-lucide-zap"
                        class="h-4 w-4"
                      />
                      积分充值
                    </span>
                  </button>
                </div>
              </div>

              <div class="overflow-hidden">
                <div
                  class="flex w-[200%] transition-transform duration-300 ease-out"
                  :style="{ transform: `translateX(${currentTab === 'membership' ? '0%' : '-50%'})` }"
                >
                  <div class="w-1/2 shrink-0 pr-2">
                    <TransitionGroup
                      tag="div"
                      class="grid grid-cols-1 md:grid-cols-3 gap-4"
                      enter-active-class="transition duration-200 ease-out"
                      enter-from-class="opacity-0 translate-y-2"
                      enter-to-class="opacity-100 translate-y-0"
                      leave-active-class="transition duration-150 ease-in"
                      leave-from-class="opacity-100 translate-y-0"
                      leave-to-class="opacity-0 translate-y-2"
                    >
                      <div
                        v-for="plan in plans.membership"
                        :key="plan.code"
                        class="rounded-2xl border border-default bg-default p-6 shadow-sm flex flex-col"
                      >
                        <div class="text-center">
                          <h3 class="text-xl font-bold text-highlighted">
                            {{ plan.name }}
                          </h3>
                          <div class="mt-5 text-4xl font-bold text-highlighted">
                            <span class="text-2xl">¥</span>{{ Number(plan.amount).toFixed(0) }}
                          </div>
                          <p class="mt-4 text-sm font-semibold text-toned">
                            {{ plan.desc }}
                          </p>
                          <p class="mt-2 text-sm font-semibold text-toned">
                            {{ unitText(plan) }}
                          </p>
                          <p class="mt-2 text-sm font-semibold text-toned">
                            {{ plan.period || '开通后立即生效' }}
                          </p>
                        </div>
                        <UButton
                          block
                          size="lg"
                          class="mt-6 rounded-full font-bold"
                          :loading="loadingPlan === plan.code"
                          @click="createOrder(plan)"
                        >
                          立即开通
                        </UButton>
                      </div>
                    </TransitionGroup>
                  </div>
                  <div class="w-1/2 shrink-0 pl-2">
                    <TransitionGroup
                      tag="div"
                      class="grid grid-cols-1 md:grid-cols-3 gap-4"
                      enter-active-class="transition duration-200 ease-out"
                      enter-from-class="opacity-0 translate-y-2"
                      enter-to-class="opacity-100 translate-y-0"
                      leave-active-class="transition duration-150 ease-in"
                      leave-from-class="opacity-100 translate-y-0"
                      leave-to-class="opacity-0 translate-y-2"
                    >
                      <div
                        v-for="plan in plans.credits"
                        :key="plan.code"
                        class="rounded-2xl border border-default bg-default p-6 shadow-sm flex flex-col"
                      >
                        <div class="text-center">
                          <h3 class="text-xl font-bold text-highlighted">
                            {{ plan.name }}
                          </h3>
                          <div class="mt-5 text-4xl font-bold text-highlighted">
                            <span class="text-2xl">¥</span>{{ Number(plan.amount).toFixed(0) }}
                          </div>
                          <p class="mt-4 text-sm font-semibold text-toned">
                            {{ `${plan.credits} 积分` }}
                          </p>
                          <p class="mt-2 text-sm font-semibold text-toned">
                            {{ unitText(plan) }}
                          </p>
                          <p class="mt-2 text-sm font-semibold text-toned">
                            积分永久有效
                          </p>
                        </div>
                        <UButton
                          block
                          size="lg"
                          class="mt-6 rounded-full font-bold"
                          :loading="loadingPlan === plan.code"
                          @click="createOrder(plan)"
                        >
                          立即充值
                        </UButton>
                      </div>
                    </TransitionGroup>
                  </div>
                </div>
              </div>
            </template>

            <template v-else>
              <div class="grid gap-4 lg:grid-cols-[1.2fr_0.9fr]">
                <section class="rounded-2xl border border-default bg-default p-4 sm:p-5">
                  <div class="flex items-center justify-between">
                    <p class="font-semibold text-highlighted">
                      订单信息
                    </p>
                    <UBadge
                      :color="orderExpired ? 'error' : 'neutral'"
                      variant="soft"
                    >
                      {{ orderExpired ? '已超时' : `剩余 ${remainingText}` }}
                    </UBadge>
                  </div>
                  <div class="mt-4 rounded-2xl border border-default bg-elevated/40 p-4">
                    <p class="text-sm font-semibold text-dimmed">
                      {{ currentPlan?.name }}
                    </p>
                    <div class="mt-2 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
                      <div>
                        <p class="text-sm text-toned">
                          订单编号：{{ currentOrder?.order.tradeNo }}
                        </p>
                        <p class="mt-2 text-sm text-toned">
                          到账权益：{{ currentPlan?.orderType === 'credits' ? `${currentPlan?.credits || 0} 积分` : `${currentPlan?.name} / ${currentPlan?.period || '按套餐配置'}` }}
                        </p>
                      </div>
                      <p class="text-3xl font-bold text-primary">
                        ¥{{ currentPlan ? Number(currentPlan.amount).toFixed(2) : '0.00' }}
                      </p>
                    </div>
                  </div>
                </section>

                <section class="rounded-2xl border border-default bg-default p-4 sm:p-5">
                  <p class="font-semibold text-highlighted">
                    选择支付方式
                  </p>
                  <div class="mt-4 space-y-3">
                    <button
                      v-for="item in availablePayTypes"
                      :key="item.code"
                      type="button"
                      class="w-full rounded-2xl border px-4 py-4 text-left transition-all"
                      :class="selectedPayType === item.code ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'border-default hover:border-primary/50'"
                      :disabled="reloadingOrder"
                      @click="selectedPayType = item.code"
                    >
                      <div class="flex items-center gap-3">
                        <div
                          class="h-10 w-10 rounded-xl grid place-items-center text-white"
                          :class="item.code === 'wxpay' ? 'bg-[#14c55e]' : item.code === 'qqpay' ? 'bg-[#4d8dff]' : 'bg-[#2f76ff]'"
                        >
                          <UIcon
                            :name="item.icon"
                            class="h-5 w-5"
                          />
                        </div>
                        <div class="min-w-0 flex-1">
                          <p class="font-semibold text-highlighted">
                            {{ item.name }}
                          </p>
                          <p class="text-sm text-dimmed">
                            {{ item.desc }}
                          </p>
                        </div>
                        <UIcon
                          :name="selectedPayType === item.code ? 'i-lucide-check-circle-2' : 'i-lucide-circle'"
                          class="h-5 w-5 text-primary"
                        />
                      </div>
                    </button>
                  </div>
                </section>
              </div>

              <div class="rounded-2xl bg-primary/10 px-4 py-3 text-sm text-primary font-medium">
                <UIcon
                  name="i-lucide-info"
                  class="mr-2 inline-block h-4 w-4"
                />
                {{ orderExpired ? '当前订单已超时，请返回套餐重新生成订单。' : selectedPayType === 'alipay' ? '支付宝付款链接已生成，点击下方按钮会打开支付宝官方收银台。' : selectedPayType === 'wxpay' ? '微信支付已准备好，请使用微信完成付款。' : 'QQ 钱包支付已准备好，请使用 QQ 钱包完成付款。' }}
              </div>

              <div class="rounded-2xl border border-default bg-elevated/40 p-4">
                <p class="font-semibold text-highlighted">
                  {{ selectedPayType === 'alipay' ? '支付宝收银台已准备好' : '支付收银台已准备好' }}
                </p>
                <p class="mt-1 text-sm text-dimmed">
                  {{ orderExpired ? '订单有效期为 15 分钟，超时后不会继续使用旧订单。' : '点击下方按钮会打开对应支付页面。' }}
                </p>
              </div>

              <div class="rounded-2xl bg-elevated/60 px-4 py-3 text-sm text-toned">
                <UIcon
                  name="i-lucide-shield-check"
                  class="mr-2 inline-block h-4 w-4 text-primary"
                />
                真实支付需要在第三方平台完成付款，系统收到官方回调并签名通过后才会入账。
              </div>

              <div class="flex flex-col-reverse gap-3 sm:flex-row">
                <UButton
                  color="neutral"
                  variant="soft"
                  block
                  class="rounded-full"
                  @click="currentStep = 'plans'"
                >
                  返回套餐
                </UButton>
                <UButton
                  block
                  class="rounded-full font-bold"
                  :loading="reloadingOrder"
                  :disabled="orderExpired"
                  @click="openPayUrl"
                >
                  <UIcon
                    name="i-lucide-link"
                    class="mr-2 h-4 w-4"
                  />
                  打开支付收银台
                </UButton>
              </div>
            </template>
          </div>
        </Transition>
      </div>
    </template>
  </UModal>
</template>
