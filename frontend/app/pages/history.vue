<script setup lang="ts">
import type { ApiAsset, ApiBalanceLog, ApiGeneration } from '~/composables/useApi'

useHead({ title: '生成历史 - 摘星AI' })

const api = useApi()
const auth = useAuth()
const activeTab = ref<'generations' | 'balance'>('generations')
const previewOpen = ref(false)
const selectedAsset = ref<(ApiAsset & { prompt?: string, appName?: string, model?: string, isFavorite?: boolean }) | null>(null)

onMounted(() => {
  auth.loadMe()
})

const { data: jobs, pending, error, refresh } = await useAsyncData(
  'generation-history',
  () => auth.token.value ? api.get<ApiGeneration[]>('/generations?limit=30') : Promise.resolve([]),
  {
    default: () => [],
    watch: [auth.token]
  }
)

const { data: balanceLogs, pending: balancePending, error: balanceError, refresh: refreshBalanceLogs } = await useAsyncData(
  'balance-logs',
  () => auth.token.value ? api.get<ApiBalanceLog[]>('/balance-logs?limit=100') : Promise.resolve([]),
  {
    default: () => [],
    watch: [auth.token]
  }
)

const statusLabel = (status: ApiGeneration['status']) => ({
  queued: '排队中',
  running: '生成中',
  succeeded: '已完成',
  failed: '失败'
}[status])

const balanceTypeLabel = (type: string) => ({
  increase: '增加',
  decrease: '减少',
  set: '覆盖'
}[type] || type)

const balanceAmountText = (log: ApiBalanceLog) => {
  if (log.changeType === 'increase') return '+' + log.amount
  if (log.changeType === 'decrease') return '-' + log.amount
  return log.amount
}

const refreshCurrent = () => activeTab.value === 'balance' ? refreshBalanceLogs() : refresh()
const openPreview = (job: ApiGeneration, asset: ApiAsset) => {
  selectedAsset.value = {
    ...asset,
    prompt: job.prompt,
    appName: job.appName || 'AI 绘画作图',
    model: job.model,
    isFavorite: Boolean((asset.meta as Record<string, unknown>)?.isFavorite)
  }
  previewOpen.value = true
}
</script>

<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 py-6 space-y-5">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-bold text-highlighted">
          生成历史
        </h1>
        <p class="text-sm text-dimmed mt-1">
          查看当前账号的生图任务和结果。
        </p>
      </div>
    
    </div>

    <div class="flex items-center gap-1 rounded-lg border border-default bg-default p-1 w-fit">
      <UButton
        :color="activeTab === 'generations' ? 'primary' : 'neutral'"
        variant="soft"
        size="sm"
        @click="activeTab = 'generations'"
      >
        生成记录
      </UButton>
      <UButton
        :color="activeTab === 'balance' ? 'primary' : 'neutral'"
        variant="soft"
        size="sm"
        @click="activeTab = 'balance'"
      >
        余额明细
      </UButton>
    </div>

    <UAlert
      v-if="!auth.token.value"
      color="warning"
      variant="soft"
      icon="i-lucide-lock"
      title="请先登录"
      description="登录后会加载你的生成历史。"
    />

    <UAlert
      v-else-if="activeTab === 'generations' && error"
      color="error"
      variant="soft"
      icon="i-lucide-alert-circle"
      title="历史记录加载失败"
      description="请确认 Go 后端正在运行，并且登录状态有效。"
    />

    <UAlert
      v-else-if="activeTab === 'balance' && balanceError"
      color="error"
      variant="soft"
      icon="i-lucide-alert-circle"
      title="余额明细加载失败"
      description="请确认 Go 后端正在运行，并且登录状态有效。"
    />

    <div
      v-if="activeTab === 'generations' && pending"
      class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4"
    >
      <USkeleton
        v-for="item in 6"
        :key="item"
        class="h-56 rounded-lg"
      />
    </div>

    <div
      v-else-if="activeTab === 'generations' && auth.token.value && jobs.length === 0"
      class="flex flex-col items-center justify-center min-h-[20rem] gap-3 text-center border border-dashed border-default rounded-lg"
    >
      <UIcon
        name="i-lucide-clock"
        class="w-8 h-8 text-dimmed"
      />
      <div>
        <p class="font-medium text-highlighted">
          暂无生成记录
        </p>
        <p class="text-sm text-dimmed">
          从应用中心创建第一张图片。
        </p>
      </div>
    </div>

    <div
      v-else-if="activeTab === 'generations'"
      class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4"
    >
      <article
        v-for="job in jobs"
        :key="job.id"
        class="rounded-lg border border-default bg-default overflow-hidden"
      >
        <div class="aspect-square bg-elevated flex items-center justify-center">
          <button
            v-if="job.assets?.[0]?.url"
            type="button"
            class="h-full w-full"
            @click="openPreview(job, job.assets[0])"
          >
            <img
              :src="job.assets[0].url"
              :alt="job.prompt"
              class="h-full w-full object-cover"
            >
          </button>
          <UIcon
            v-else
            name="i-lucide-image"
            class="w-10 h-10 text-dimmed"
          />
        </div>
        <div class="p-4 space-y-3">
          <div class="flex items-center justify-between gap-2">
            <UBadge
              color="primary"
              variant="soft"
              class="rounded"
            >
              {{ job.appName || '通用生图' }}
            </UBadge>
            <UBadge
              :color="job.status === 'succeeded' ? 'success' : job.status === 'failed' ? 'error' : 'warning'"
              variant="soft"
            >
              {{ statusLabel(job.status) }}
            </UBadge>
          </div>
          <p class="text-sm text-highlighted line-clamp-2">
            {{ job.prompt }}
          </p>
          <div class="flex items-center justify-between text-xs text-dimmed">
            <span>{{ job.model }}</span>
            <span>{{ new Date(job.createdAt).toLocaleString('zh-CN') }}</span>
          </div>
        </div>
      </article>
    </div>

    <div
      v-else-if="balancePending"
      class="rounded-lg border border-default bg-default p-4 space-y-3"
    >
      <USkeleton
        v-for="item in 5"
        :key="item"
        class="h-12 rounded"
      />
    </div>

    <div
      v-else-if="auth.token.value && balanceLogs.length === 0"
      class="flex flex-col items-center justify-center min-h-[16rem] gap-3 text-center border border-dashed border-default rounded-lg"
    >
      <UIcon
        name="i-lucide-wallet"
        class="w-8 h-8 text-dimmed"
      />
      <div>
        <p class="font-medium text-highlighted">
          暂无余额明细
        </p>
        <p class="text-sm text-dimmed">
          充值、扣费和后台调整会显示在这里。
        </p>
      </div>
    </div>

    <div
      v-else
      class="overflow-hidden rounded-lg border border-default bg-default"
    >
      <div class="grid grid-cols-[0.7fr_0.8fr_0.8fr_0.8fr_1fr_1fr] gap-3 px-4 py-3 text-xs font-medium text-dimmed border-b border-default">
        <span>类型</span>
        <span>金额</span>
        <span>变动前</span>
        <span>变动后</span>
        <span>备注</span>
        <span>时间</span>
      </div>
      <div
        v-for="log in balanceLogs"
        :key="log.id"
        class="grid grid-cols-[0.7fr_0.8fr_0.8fr_0.8fr_1fr_1fr] gap-3 px-4 py-3 text-sm border-b border-default last:border-b-0 items-center"
      >
        <UBadge
          :color="log.changeType === 'increase' ? 'success' : log.changeType === 'decrease' ? 'warning' : 'primary'"
          variant="soft"
        >
          {{ balanceTypeLabel(log.changeType) }}
        </UBadge>
        <span
          class="font-medium"
          :class="log.changeType === 'increase' ? 'text-success' : log.changeType === 'decrease' ? 'text-warning' : 'text-highlighted'"
        >
          {{ balanceAmountText(log) }}
        </span>
        <span class="text-dimmed">{{ log.balanceBefore }}</span>
        <span class="text-highlighted">{{ log.balanceAfter }}</span>
        <span class="truncate text-toned">{{ log.note || '-' }}</span>
        <span class="text-dimmed">{{ new Date(log.createdAt).toLocaleString('zh-CN') }}</span>
      </div>
    </div>

    <MediaPreviewModal
      v-model:open="previewOpen"
      :asset="selectedAsset"
      @favorite-changed="refresh()"
    />
  </div>
</template>
