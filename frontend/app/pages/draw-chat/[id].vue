<script setup lang="ts">
import type { ApiAsset, ApiConversationDetail, ApiConversationMessage, ApiDrawConversationResult, ApiGeneration } from '~/composables/useApi'

useHead({ title: '绘画会话 - 季星AI' })

const api = useApi()
const auth = useAuth()
const route = useRoute()
const { loadConversations } = useConversations()
const conversationID = computed(() => String(route.params.id || ''))
const message = useMessage()

onMounted(() => {
  auth.loadMe()
})

const { data: detail, pending, error, refresh } = await useAsyncData(
  () => 'draw-chat-' + conversationID.value,
  () => auth.token.value && conversationID.value
    ? api.get<ApiConversationDetail>('/conversations/' + conversationID.value)
    : Promise.resolve(null),
  { default: () => null, watch: [conversationID, auth.token], server: false }
)

const job = computed<ApiGeneration | undefined>(() => detail.value?.job)
const messages = computed<ApiConversationMessage[]>(() => detail.value?.messages || [])
const isGenerating = computed(() => !job.value || job.value.status === 'queued' || job.value.status === 'running')
const firstAsset = computed(() => job.value?.assets?.[0])
const streamingText = ref('')
const streamSource = ref('')
const input = ref('')
const sending = ref(false)
const previewOpen = ref(false)
const selectedAsset = ref<(ApiAsset & { prompt?: string, appName?: string, model?: string, isFavorite?: boolean }) | null>(null)

let refreshTimer: ReturnType<typeof setInterval> | null = null
let typeTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  refreshTimer = setInterval(() => {
    if (isGenerating.value) {
      refresh()
    } else {
      loadConversations()
    }
  }, 1800)
})

onBeforeUnmount(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  if (typeTimer) clearInterval(typeTimer)
})

watch(
  () => messages.value.findLast(item => item.role === 'assistant')?.content || '',
  (content) => {
    if (!content || content === streamSource.value) return
    streamSource.value = content
    streamingText.value = ''
    if (typeTimer) clearInterval(typeTimer)
    let index = 0
    typeTimer = setInterval(() => {
      streamingText.value = content.slice(0, index + 1)
      index += 1
      if (index >= content.length && typeTimer) {
        clearInterval(typeTimer)
        typeTimer = null
      }
    }, 55)
  },
  { immediate: true }
)

const messageText = (item: ApiConversationMessage) => {
  if (item.role !== 'assistant') return item.content
  const latest = messages.value.findLast(message => message.role === 'assistant')
  return latest?.id === item.id ? streamingText.value || item.content : item.content
}

const messageJobID = (item: ApiConversationMessage) => String((item.meta as Record<string, unknown>)?.jobId || '')
const messageAssetURL = (item: ApiConversationMessage) => String((item.meta as Record<string, unknown>)?.assetUrl || '')
const isMessageGenerating = (item: ApiConversationMessage) => {
  if (item.role !== 'assistant') return false
  const id = messageJobID(item)
  return Boolean(id && id === job.value?.id && isGenerating.value && !firstAsset.value)
}
const messageImageURL = (item: ApiConversationMessage) => {
  const assetURL = messageAssetURL(item)
  if (assetURL) return assetURL
  if (messageJobID(item) && messageJobID(item) === job.value?.id && firstAsset.value) return firstAsset.value.url
  return ''
}

const formatDate = (value?: string) => value ? new Date(value).toLocaleString('zh-CN') : '-'
const openPreview = (asset: ApiAsset, prompt?: string) => {
  selectedAsset.value = {
    ...asset,
    prompt: prompt || job.value?.prompt || detail.value?.conversation.title || '',
    appName: detail.value?.conversation.appName || '专业绘画',
    model: job.value?.model || '',
    isFavorite: Boolean((asset.meta as Record<string, unknown>)?.isFavorite)
  }
  previewOpen.value = true
}

const onFavoriteChanged = (assetId: string, isFavorite: boolean) => {
  if (!detail.value?.job?.assets) return
  detail.value.job.assets = detail.value.job.assets.map(asset => asset.id === assetId
    ? { ...asset, meta: { ...(asset.meta || {}), isFavorite } }
    : asset)
  if (selectedAsset.value?.id === assetId) {
    selectedAsset.value = { ...selectedAsset.value, isFavorite }
  }
}

const openMessagePreview = (item: ApiConversationMessage) => {
  const url = messageImageURL(item)
  if (!url) return
  if (messageJobID(item) === job.value?.id && firstAsset.value) {
    openPreview(firstAsset.value, job.value?.prompt)
    return
  }
  selectedAsset.value = {
    id: messageJobID(item) || item.id,
    jobId: messageJobID(item),
    kind: 'image',
    url,
    thumbnailUrl: url,
    width: 0,
    height: 0,
    mimeType: 'image/png',
    sortOrder: 0,
    meta: item.meta || {},
    createdAt: item.createdAt,
    prompt: item.content,
    appName: detail.value?.conversation.appName || '专业绘画',
    model: String((item.meta as Record<string, unknown>)?.model || '')
  }
  previewOpen.value = true
}

const submitFollowUp = async () => {
  const content = input.value.trim()
  if (!content || sending.value || !detail.value?.conversation.id) return
  if (!auth.token.value) {
    message.error('请先登录', '登录后可以继续绘画对话')
    return
  }
  sending.value = true
  try {
    const result = await api.post<ApiDrawConversationResult>('/generations/professional-draw', {
      conversationId: detail.value.conversation.id,
      prompt: content,
      params: {
        quality: String(job.value?.params?.quality || '1K'),
        ratio: String(job.value?.params?.ratio || '1:1'),
        context: true
      }
    })
    input.value = ''
    detail.value = {
      ...detail.value,
      messages: [...(detail.value.messages || []), ...result.messages],
      job: result.job,
      conversation: {
        ...detail.value.conversation,
        updatedAt: result.job.createdAt
      }
    }
    await loadConversations()
    message.success('已提交', `已扣费 ${result.charged}`)
    await nextTick()
    refresh()
  } catch (error) {
    message.error('提交失败', error instanceof Error ? error.message : '请稍后重试')
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <div class="max-w-5xl mx-auto px-4 sm:px-6 py-6 pb-32">
    <UAlert
      v-if="!auth.token.value"
      color="warning"
      variant="soft"
      icon="i-lucide-lock"
      title="请先登录"
      description="登录后可以查看绘画会话。"
    />

    <div
      v-else
      class="space-y-5"
    >
      <UAlert
        v-if="error"
        color="error"
        variant="soft"
        icon="i-lucide-alert-circle"
        title="会话加载失败"
        description="请确认后端服务已重启到最新版本，或该会话仍然存在。"
      />

      <div class="flex items-center justify-between gap-3">
        <div>
          <h1 class="text-xl font-bold text-highlighted">
            {{ detail?.conversation.title || '自定义绘画' }}
          </h1>
          <p class="text-sm text-dimmed mt-1">
            {{ detail?.conversation.appName || '专业绘画' }} · {{ formatDate(detail?.conversation.createdAt) }}
          </p>
        </div>
      </div>

      <div class="rounded-2xl border border-default bg-default p-4 sm:p-6 space-y-5 min-h-[32rem]">
        <div
          v-if="pending && !detail"
          class="space-y-4"
        >
          <USkeleton class="h-16 w-2/3 ml-auto rounded-2xl" />
          <USkeleton class="h-64 w-full rounded-2xl" />
        </div>

        <template v-else>
          <div
            v-for="item in messages"
            :key="item.id"
            class="flex"
            :class="item.role === 'user' ? 'justify-end' : 'justify-start'"
          >
            <div
              class="max-w-[86%] rounded-2xl px-4 py-3 text-sm"
              :class="item.role === 'user' ? 'bg-primary text-inverted' : 'bg-elevated text-highlighted'"
            >
              <p class="whitespace-pre-wrap">
                {{ messageText(item) }}
              </p>
              <div
                v-if="item.role === 'assistant'"
                class="mt-3"
              >
                <div
                  v-if="isMessageGenerating(item)"
                  class="w-72 max-w-full aspect-square rounded-xl border border-default bg-default/70 grid place-items-center overflow-hidden"
                >
                  <div class="text-center space-y-3">
                    <UIcon
                      name="i-lucide-loader-circle"
                      class="h-8 w-8 mx-auto animate-spin text-primary"
                    />
                    <p class="text-sm text-toned">
                      正在加载生成图片...
                    </p>
                  </div>
                </div>
                <button
                  v-else-if="messageImageURL(item)"
                  type="button"
                  class="block overflow-hidden rounded-xl border border-default text-left transition hover:shadow-lg"
                  @click="openMessagePreview(item)"
                >
                  <img
                    :src="messageImageURL(item)"
                    :alt="job?.prompt || '生成图片'"
                    class="w-full max-w-xl object-cover"
                  >
                </button>
              </div>
              <p class="mt-2 text-xs opacity-75">
                {{ formatDate(item.createdAt) }}
              </p>
            </div>
          </div>

          <div
            v-if="!messages.length"
            class="grid min-h-[22rem] place-items-center text-sm text-dimmed"
          >
            暂无会话内容
          </div>
        </template>
      </div>

      <div class="fixed inset-x-0 bottom-0 z-20 border-t border-default bg-default/95 px-4 py-3 backdrop-blur">
        <form
          class="mx-auto flex max-w-5xl items-end gap-2 rounded-2xl border border-default bg-elevated/60 p-3"
          @submit.prevent="submitFollowUp"
        >
          <UTextarea
            v-model="input"
            :rows="2"
            autoresize
            class="flex-1"
            placeholder="继续描述你想调整的内容，例如：保持人物不变，把背景换成夜晚赛博城市。"
            :disabled="sending || !auth.token.value"
            @keydown.enter.exact.prevent="submitFollowUp"
          />
          <UButton
            type="submit"
            icon="i-lucide-send"
            :loading="sending"
            :disabled="!input.trim() || !auth.token.value"
            class="rounded-full px-5"
          >
            发送
          </UButton>
        </form>
      </div>
    </div>

    <MediaPreviewModal
      v-model:open="previewOpen"
      :asset="selectedAsset"
      @favorite-changed="onFavoriteChanged"
    />
  </div>
</template>

