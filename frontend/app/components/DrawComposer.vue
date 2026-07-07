<script setup lang="ts">
import type { ApiApp, ApiConversationMessage, ApiDrawConversationResult } from '~/composables/useApi'

const prompt = defineModel<string>('prompt', { default: '' })
const api = useApi()
const auth = useAuth()
const message = useMessage()
const router = useRouter()
const { upsertConversation, loadConversations } = useConversations()
const { data: apps } = await useAsyncData('draw-pricing-apps', () => api.get<ApiApp[]>('/apps'), {
  default: () => []
})

// 画质
const qualities = ['1K', '2K', '4K']
const quality = ref('1K')
const sending = ref(false)
const rewriting = ref(false)
const confirmOpen = ref(false)
const messages = ref<ApiConversationMessage[]>([])

// 图片比例
const ratios = [
  { label: '1:1', value: '1:1', icon: 'i-lucide-square' },
  { label: '4:3', value: '4:3', icon: 'i-lucide-rectangle-horizontal' },
  { label: '3:4', value: '3:4', icon: 'i-lucide-rectangle-vertical' },
  { label: '16:9', value: '16:9', icon: 'i-lucide-rectangle-horizontal' },
  { label: '9:16', value: '9:16', icon: 'i-lucide-rectangle-vertical' }
]
const ratio = ref('1:1')

// 参考图(支持多张)
interface RefImage {
  id: number
  name: string
  url: string
}
const refImages = ref<RefImage[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
let uid = 0

const pickFiles = () => fileInput.value?.click()

const onFiles = (e: Event) => {
  const files = (e.target as HTMLInputElement).files
  if (!files) return
  for (const file of Array.from(files)) {
    if (!file.type.startsWith('image/')) continue
    refImages.value.push({ id: uid++, name: file.name, url: URL.createObjectURL(file) })
  }
  // 允许重复选择同一文件
  if (fileInput.value) fileInput.value.value = ''
}

const removeImage = (id: number) => {
  const target = refImages.value.find(i => i.id === id)
  if (target) URL.revokeObjectURL(target.url)
  refImages.value = refImages.value.filter(i => i.id !== id)
}

onBeforeUnmount(() => {
  refImages.value.forEach(i => URL.revokeObjectURL(i.url))
})

const canSend = computed(() => prompt.value.trim().length > 0 || refImages.value.length > 0)
const professionalApp = computed(() => apps.value.find(app => app.code === 'ai-drawing'))
const membershipLabel = computed(() => ({
  free: 'Free',
  v1: 'V1会员',
  v2: 'V2会员'
}[auth.user.value?.membershipLevel || 'free']))

const chargeAmount = computed(() => {
  const app = professionalApp.value
  const level = auth.user.value?.membershipLevel || 'free'
  if (!app) return '0'
  if (level === 'v2') return app.priceV2
  if (level === 'v1') return app.priceV1
  return app.priceFree
})

const currentBalance = computed(() => auth.user.value?.balance || '0')

const requestSummary = computed(() => [
  { label: '模式', value: '专业绘画' },
  { label: '画质', value: quality.value },
  { label: '比例', value: ratio.value },
  { label: '参考图', value: refImages.value.length ? `${refImages.value.length} 张` : '无' }
])

onMounted(() => {
  auth.loadMe()
})

const onRewritePrompt = async () => {
  if (rewriting.value) return
  if (!prompt.value.trim()) {
    message.info('润色/改写', '先输入你的设计想法,再进行润色/改写。')
    return
  }
  if (!auth.token.value) {
    message.error('请先登录', '登录后可以使用提示词润色')
    return
  }
  rewriting.value = true
  try {
    const result = await api.post<{ prompt: string }>('/generations/professional-draw/rewrite', {
      prompt: prompt.value
    })
    prompt.value = result.prompt
    message.success('润色完成', '提示词已更新')
  } catch (error) {
    message.error('润色失败', error instanceof Error ? error.message : '请稍后重试')
  } finally {
    rewriting.value = false
  }
}

const onSend = async () => {
  if (!canSend.value || sending.value) return
  if (!auth.token.value) {
    message.error('请先登录', '登录后可以提交专业绘画任务')
    return
  }
  await auth.loadMe()
  confirmOpen.value = true
}

const confirmSubmit = async () => {
  if (sending.value) return
  sending.value = true
  try {
    const result = await api.post<ApiDrawConversationResult>('/generations/professional-draw', {
      prompt: prompt.value || '请根据参考图生成专业绘画作品',
      params: {
        quality: quality.value,
        ratio: ratio.value,
        refImages: refImages.value.map(i => i.name)
      }
    })
    messages.value.push(...(Array.isArray(result.messages) ? result.messages : []), {
      id: 'draw-reminder-' + Date.now(),
      conversationId: result.conversationId,
      role: 'assistant',
      content: '你可以先离开页面，待生图成功后前往历史记录查看生图结果。',
      meta: {},
      createdAt: new Date().toISOString()
    })
    auth.user.value = result.user
    confirmOpen.value = false
    prompt.value = ''
    message.success('提交成功', `已扣费 ${result.charged}`)
    upsertConversation({
      id: result.conversationId,
      userId: result.user.id,
      appId: result.job.appId,
      appName: result.job.appName || 'AI 绘画作图',
      kind: 'draw',
      title: result.job.prompt,
      createdAt: result.job.createdAt,
      updatedAt: result.job.createdAt
    })
    loadConversations()
    await router.push('/draw-chat/' + result.conversationId)
  } catch (error) {
    message.error('提交失败', error instanceof Error ? error.message : '请稍后重试')
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <div
      v-if="messages.length"
      class="rounded-2xl border border-default bg-default p-4 sm:p-5 space-y-4"
    >
      <div
        v-for="item in messages"
        :key="item.id"
        class="flex"
        :class="item.role === 'user' ? 'justify-end' : 'justify-start'"
      >
        <div
          class="max-w-[82%] rounded-2xl px-4 py-3 text-sm"
          :class="item.role === 'user' ? 'bg-primary text-inverted' : 'bg-elevated text-highlighted'"
        >
          <p class="whitespace-pre-wrap">
            {{ item.content }}
          </p>
          <p class="mt-2 text-xs opacity-75">
            {{ new Date(item.createdAt).toLocaleString('zh-CN') }}
          </p>
        </div>
      </div>
    </div>

    <div class="rounded-3xl border border-default bg-default shadow-sm p-4 sm:p-5">
      <!-- 参考图预览 -->
      <div
        v-if="refImages.length"
        class="flex flex-wrap gap-2 mb-3"
      >
        <div
          v-for="img in refImages"
          :key="img.id"
          class="relative w-16 h-16 rounded-lg overflow-hidden border border-default group"
        >
          <img
            :src="img.url"
            :alt="img.name"
            class="w-full h-full object-cover"
          >
          <button
            type="button"
            class="absolute top-0.5 right-0.5 flex items-center justify-center w-5 h-5 rounded-full bg-black/60 text-white opacity-0 group-hover:opacity-100 transition-opacity"
            aria-label="移除参考图"
            @click="removeImage(img.id)"
          >
            <UIcon
              name="i-lucide-x"
              class="w-3 h-3"
            />
          </button>
        </div>
      </div>

      <div class="flex items-start gap-2">
        <!-- 多行输入 -->
        <UTextarea
          v-model="prompt"
          :rows="5"
          autoresize
          variant="none"
          placeholder="和我聊聊,你想要什么设计。"
          class="min-w-0 flex-1"
          :ui="{ base: 'resize-none text-base bg-transparent' }"
          @keydown.enter.exact.prevent="onSend"
        />

        <UButton
          icon="i-lucide-wand-sparkles"
          color="primary"
          variant="soft"
          size="md"
          class="w-10 h-10 sm:w-auto sm:px-4 rounded-full shrink-0"
          aria-label="润色/改写"
          title="润色/改写提示词"
          :disabled="rewriting"
          :loading="rewriting"
          @click="onRewritePrompt"
        />
      </div>

      <!-- 隐藏文件输入 -->
      <input
        ref="fileInput"
        type="file"
        accept="image/*"
        multiple
        class="hidden"
        @change="onFiles"
      >

      <!-- 底部工具栏 -->
      <div class="flex flex-wrap items-center justify-between gap-2 mt-3">
        <div class="flex flex-wrap items-center gap-3">
          <!-- 添加参考图 -->
          <UButton
            icon="i-lucide-image-plus"
            color="neutral"
            variant="outline"
            size="md"
            class="rounded-full"
            @click="pickFiles"
          >
            参考图
          </UButton>

          <!-- 画质 -->
          <USelectMenu
            v-model="quality"
            :items="qualities"
            size="md"
            icon="i-lucide-gem"
            :search-input="false"
            class="rounded-full min-w-24"
          />

          <!-- 比例 -->
          <USelectMenu
            v-model="ratio"
            :items="ratios"
            value-key="value"
            size="md"
            icon="i-lucide-ratio"
            :search-input="false"
            class="rounded-full min-w-24"
          />
        </div>

        <UButton
          icon="i-lucide-send"
          color="neutral"
          size="lg"
          class="rounded-full px-5 font-medium"
          :disabled="!canSend"
          :loading="sending"
          @click="onSend"
        >
          发送
        </UButton>
      </div>

      <UModal
        v-model:open="confirmOpen"
        title="确认专业绘画任务"
        :ui="{ content: 'max-w-lg' }"
      >
        <template #body>
          <div class="space-y-4">
            <div class="rounded-lg border border-default divide-y divide-default">
              <div
                v-for="item in requestSummary"
                :key="item.label"
                class="flex items-center justify-between px-3 py-2 text-sm"
              >
                <span class="text-dimmed">{{ item.label }}</span>
                <span class="font-medium text-highlighted">{{ item.value }}</span>
              </div>
            </div>

            <div class="rounded-lg bg-elevated p-3 space-y-2 text-sm">
              <div class="flex items-center justify-between">
                <span class="text-dimmed">会员等级</span>
                <span class="font-medium text-highlighted">{{ membershipLabel }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-dimmed">当前余额</span>
                <span class="font-medium text-highlighted">{{ currentBalance }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-dimmed">本次扣费</span>
                <span class="font-semibold text-primary">{{ chargeAmount }}</span>
              </div>
            </div>
          </div>
        </template>
        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton
              color="neutral"
              variant="soft"
              @click="confirmOpen = false"
            >
              取消
            </UButton>
            <UButton
              :loading="sending"
              @click="confirmSubmit()"
            >
              确认提交
            </UButton>
          </div>
        </template>
      </UModal>
    </div>
  </div>
</template>
