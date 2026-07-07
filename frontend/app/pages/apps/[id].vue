<script setup lang="ts">
import type { ApiApp, ApiAssistantChatResult, ApiGeneration } from '~/composables/useApi'

interface AppChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  imageUrl?: string
  status?: 'idle' | 'loading' | 'done' | 'error'
  createdAt: string
}

const route = useRoute()
const api = useApi()
const auth = useAuth()
const message = useMessage()

const appId = computed(() => String(route.params.id || ''))
const queryPrompt = computed(() => {
  const raw = route.query.prompt
  return Array.isArray(raw) ? String(raw[0] || '') : String(raw || '')
})

const { data: app, pending, error } = await useAsyncData(
  () => `app-detail-${appId.value}`,
  () => api.get<ApiApp>('/apps/' + encodeURIComponent(appId.value)),
  { default: () => null, watch: [appId] }
)

const input = ref('')
const sending = ref(false)
const chatMessages = ref<AppChatMessage[]>([])
const messagesViewport = ref<HTMLElement | null>(null)

const appTypeLabel = computed(() => app.value?.appType === 'text' ? '文本应用' : '生图应用')
const appTypeIcon = computed(() => app.value?.appType === 'text' ? 'i-lucide-file-text' : 'i-lucide-image')
const chargeLabel = computed(() => {
  const level = auth.user.value?.membershipLevel || 'free'
  if (!app.value) return '0'
  if (level === 'v2') return app.value.priceV2
  if (level === 'v1') return app.value.priceV1
  return app.value.priceFree
})

const renderPrompt = (value: string) => {
  const template = app.value?.promptTemplate?.trim() || '{{prompt}}'
  return template.replaceAll('{{prompt}}', value.trim() || '你的需求')
}

const quickPrompts = computed(() => {
  if (app.value?.appType === 'text') {
    return [
      '帮我生成 10 个适合小红书的标题和正文开头',
      '把这段话改写得更专业、更有转化力',
      '围绕新品发布写一份社媒推广文案'
    ]
  }
  return [
    '生成一张高端科技感产品海报，亮色背景，商业广告质感',
    '生成一张电商主图，主体清晰，干净背景，高级光影',
    '生成一张适合朋友圈传播的活动宣传海报'
  ]
})

const nowISO = () => new Date().toISOString()
const newMessageID = (prefix: string) => `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`

const resetConversation = (currentApp: ApiApp | null) => {
  chatMessages.value = []
  if (!currentApp) return
  chatMessages.value.push({
    id: newMessageID('welcome'),
    role: 'assistant',
    content: `你好，我是「${currentApp.name}」应用。把你的需求发给我，我会来完成${currentApp.appType === 'text' ? '文本生成' : '图片生成'}。`,
    status: 'idle',
    createdAt: nowISO()
  })
}

watch(
  [app, queryPrompt],
  ([currentApp, prompt]) => {
    resetConversation(currentApp)
    input.value = prompt || ''
  },
  { immediate: true }
)

onMounted(() => {
  auth.loadMe()
})

const scrollToBottom = async () => {
  await nextTick()
  if (messagesViewport.value) {
    messagesViewport.value.scrollTop = messagesViewport.value.scrollHeight
  }
}

const useQuickPrompt = (value: string) => {
  input.value = value
}

const appendAssistant = (messageData: Partial<AppChatMessage>) => {
  chatMessages.value.push({
    id: newMessageID('assistant'),
    role: 'assistant',
    content: messageData.content || '',
    imageUrl: messageData.imageUrl,
    status: messageData.status || 'done',
    createdAt: nowISO()
  })
}

const submitAppMessage = async () => {
  if (!app.value || sending.value) return
  const prompt = input.value.trim()
  if (!prompt) {
    message.error('请输入需求', '请先描述你想生成的内容')
    return
  }
  if (!auth.token.value) {
    appendAssistant({
      content: '请先登录后再使用应用生成内容。',
      status: 'error'
    })
    await scrollToBottom()
    return
  }

  const userContent = prompt
  input.value = ''
  chatMessages.value.push({
    id: newMessageID('user'),
    role: 'user',
    content: userContent,
    createdAt: nowISO()
  })
  const loadingID = newMessageID('assistant-loading')
  chatMessages.value.push({
    id: loadingID,
    role: 'assistant',
    content: app.value.appType === 'text' ? '正在调用文本接口生成内容...' : '正在调用生图接口生成图片...',
    status: 'loading',
    createdAt: nowISO()
  })
  sending.value = true
  await scrollToBottom()

  try {
    if (app.value.appType === 'text') {
      const result = await api.post<ApiAssistantChatResult>('/assistant/chat', {
        message: renderPrompt(userContent),
        stream: false
      })
      const answer = [...result.messages].reverse().find(item => item.role === 'assistant')?.content || '已完成生成。'
      const target = chatMessages.value.find(item => item.id === loadingID)
      if (target) {
        target.content = answer
        target.status = 'done'
      }
    } else {
      const generation = await api.post<ApiGeneration>('/generations', {
        appId: app.value.id,
        prompt: userContent,
        params: {
          source: 'app-chat',
          appCode: app.value.code
        }
      })
      const asset = generation.assets?.[0]
      const target = chatMessages.value.find(item => item.id === loadingID)
      if (target) {
        target.content = asset ? '图片已生成，结果如下。' : '任务已提交，可在生成历史查看结果。'
        target.imageUrl = asset?.url
        target.status = 'done'
      }
    }
  } catch (error) {
    const target = chatMessages.value.find(item => item.id === loadingID)
    if (target) {
      target.content = error instanceof Error ? error.message : '应用生成失败，请稍后重试。'
      target.status = 'error'
    }
  } finally {
    sending.value = false
    await scrollToBottom()
  }
}

useHead(() => ({
  title: app.value ? `${app.value.name} - 应用对话 - 摘星AI` : '应用对话 - 摘星AI'
}))
</script>

<template>
  <div class="mx-auto flex h-[calc(100vh-5rem)] max-w-6xl flex-col px-4 py-4 sm:px-6">
    <UButton
      to="/apps"
      icon="i-lucide-arrow-left"
      color="neutral"
      variant="ghost"
      size="sm"
      class="mb-3 self-start"
    >
      返回应用中心
    </UButton>

    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-alert-circle"
      title="应用加载失败"
      description="请稍后重试或联系网站管理员。"
    />

    <div
      v-else-if="pending || !app"
      class="grid flex-1 grid-rows-[auto_1fr_auto] gap-3"
    >
      <USkeleton class="h-20 rounded-lg" />
      <USkeleton class="rounded-lg" />
      <USkeleton class="h-24 rounded-lg" />
    </div>

    <section
      v-else
      class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-default bg-default"
    >
      <header class="flex items-start justify-between gap-4 border-b border-default px-4 py-4 sm:px-5">
        <div class="flex min-w-0 items-start gap-3">
          <div
            class="grid h-12 w-12 shrink-0 place-items-center rounded-lg"
            :class="app.iconColor"
          >
            <UIcon
              :name="app.icon || appTypeIcon"
              class="h-6 w-6"
            />
          </div>
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <UBadge
                color="primary"
                variant="soft"
                class="rounded"
              >
                应用对话
              </UBadge>
              <UBadge
                color="neutral"
                variant="soft"
                class="rounded"
              >
                {{ appTypeLabel }}
              </UBadge>
              <UBadge
                color="warning"
                variant="soft"
                class="rounded"
              >
                消耗 {{ chargeLabel }}
              </UBadge>
            </div>
            <h1 class="mt-2 truncate text-xl font-bold text-highlighted">
              {{ app.name }}
            </h1>
            <p class="mt-1 line-clamp-2 text-sm text-dimmed">
              {{ app.description || '输入需求后，应用会结合后台预设词和接口配置完成生成。' }}
            </p>
          </div>
        </div>
        <UButton
          to="/history"
          color="neutral"
          variant="soft"
          icon="i-lucide-clock"
          class="hidden shrink-0 sm:inline-flex"
        >
          生成历史
        </UButton>
      </header>

      <div
        ref="messagesViewport"
        class="min-h-0 flex-1 space-y-4 overflow-y-auto bg-elevated/30 px-4 py-5 sm:px-6"
      >
        <div
          v-for="item in chatMessages"
          :key="item.id"
          class="flex items-start gap-3"
          :class="item.role === 'user' ? 'justify-end' : 'justify-start'"
        >
          <div
            v-if="item.role === 'assistant'"
            class="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-primary text-inverted"
          >
            <UIcon
              :name="appTypeIcon"
              class="h-5 w-5"
            />
          </div>
          <div
            class="max-w-[88%] rounded-lg px-4 py-3 shadow-sm sm:max-w-[72%]"
            :class="item.role === 'user'
              ? 'bg-primary text-inverted'
              : item.status === 'error'
                ? 'border border-error/30 bg-error/10 text-highlighted'
                : 'border border-default bg-default text-highlighted'"
          >
            <div class="whitespace-pre-wrap text-sm leading-6">
              {{ item.content }}
            </div>
            <img
              v-if="item.imageUrl"
              :src="item.imageUrl"
              alt="应用生成图片"
              class="mt-3 max-h-[420px] w-full rounded-lg object-contain"
            >
            <div
              v-if="item.status === 'loading'"
              class="mt-3 flex items-center gap-2 text-xs opacity-75"
            >
              <UIcon
                name="i-lucide-loader-circle"
                class="h-4 w-4 animate-spin"
              />
              正在生成
            </div>
          </div>
        </div>
      </div>

      <footer class="border-t border-default bg-default px-4 py-4 sm:px-5">
        <div class="mb-3 flex gap-2 overflow-x-auto pb-1">
          <button
            v-for="item in quickPrompts"
            :key="item"
            type="button"
            class="h-8 shrink-0 rounded-full border border-default px-3 text-xs text-muted transition hover:border-primary/40 hover:text-highlighted"
            @click="useQuickPrompt(item)"
          >
            {{ item }}
          </button>
        </div>
        <div class="flex items-end gap-2">
          <UTextarea
            v-model="input"
            autoresize
            :rows="2"
            :maxrows="6"
            placeholder="输入你的需求，按当前应用预设生成..."
            class="flex-1"
            @keydown.enter.exact.prevent="submitAppMessage"
          />
          <UButton
            color="primary"
            icon="i-lucide-send"
            :loading="sending"
            :disabled="!input.trim()"
            class="h-10 shrink-0"
            @click="submitAppMessage"
          >
            发送
          </UButton>
        </div>
      </footer>
    </section>
  </div>
</template>
