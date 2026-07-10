<script setup lang="ts">
import type { ApiAssistantChatResult, ApiConversationDetail, ApiConversationMessage } from '~/composables/useApi'

const { pageTitle } = useSiteConfig()

useHead(() => ({ title: pageTitle('智能助手') }))

interface AssistantAttachment {
  id: string
  name: string
  mimeType: string
  size: number
  dataUrl: string
  text: string
}

const api = useApi()
const auth = useAuth()
const route = useRoute()
const router = useRouter()
const config = useRuntimeConfig()
const message = useMessage()
const { upsertConversation, loadConversations } = useConversations()

const input = ref('')
const sending = ref(false)
const messages = ref<ApiConversationMessage[]>([])
const conversationID = ref('')
const conversationTitle = ref('智能助手')
const attachments = ref<AssistantAttachment[]>([])
const scrollRef = ref<HTMLElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const copiedCode = ref('')

const routeConversationID = computed(() => {
  const value = route.query.conversation
  return Array.isArray(value) ? String(value[0] || '') : String(value || '')
})
const routePrompt = computed(() => {
  const value = route.query.prompt
  return Array.isArray(value) ? String(value[0] || '') : String(value || '')
})

onMounted(async () => {
  await auth.loadMe()
})

const loadConversation = async (id: string) => {
  if (!auth.token.value || !id) return
  try {
    const detail = await api.get<ApiConversationDetail>('/conversations/' + id)
    conversationID.value = detail.conversation.id
    conversationTitle.value = detail.conversation.title || '智能助手'
    messages.value = detail.messages
    await nextTick()
    scrollToBottom()
  } catch (error) {
    message.error('加载失败', error instanceof Error ? error.message : '会话加载失败')
  }
}

watch(
  [routeConversationID, () => auth.token.value],
  ([id]) => {
    if (id) {
      loadConversation(id)
    }
  },
  { immediate: true }
)

watch(
  routePrompt,
  (value) => {
    if (value && !input.value.trim()) {
      input.value = value
    }
  },
  { immediate: true }
)

watch(
  () => messages.value.map(item => item.content).join(''),
  () => nextTick(scrollToBottom)
)

const startNewChat = () => {
  conversationID.value = ''
  conversationTitle.value = '智能助手'
  messages.value = []
  attachments.value = []
  router.replace({ query: {} })
}

const scrollToBottom = () => {
  if (scrollRef.value) {
    scrollRef.value.scrollTop = scrollRef.value.scrollHeight
  }
}

const formatTime = (value?: string) => value ? new Date(value).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) : ''
const formatSize = (size: number) => {
  if (size >= 1024 * 1024) return (size / 1024 / 1024).toFixed(1) + ' MB'
  if (size >= 1024) return Math.ceil(size / 1024) + ' KB'
  return size + ' B'
}

const escapeHtml = (value: string) => value
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;')
  .replace(/'/g, '&#39;')

const inlineMarkdown = (value: string) => escapeHtml(value)
  .replace(/`([^`]+)`/g, '<code>$1</code>')
  .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>')

const codeBlock = (code: string, language = '') => {
  const encoded = encodeURIComponent(code)
  const label = escapeHtml(language || 'code')
  return [
    '<div class="code-card">',
    '<div class="code-toolbar">',
    '<span>' + label + '</span>',
    '<button type="button" class="code-copy" data-code="' + encoded + '">复制</button>',
    '</div>',
    '<pre><code>',
    escapeHtml(code),
    '</code></pre>',
    '</div>'
  ].join('')
}

const renderMarkdown = (value: string) => {
  const lines = value.split(/\r?\n/)
  const html: string[] = []
  let inCode = false
  let codeLanguage = ''
  let codeLines: string[] = []
  let inList = false
  for (const line of lines) {
    if (line.trim().startsWith('```')) {
      if (inList) {
        html.push('</ul>')
        inList = false
      }
      if (inCode) {
        html.push(codeBlock(codeLines.join('\n'), codeLanguage))
        codeLines = []
        codeLanguage = ''
        inCode = false
      } else {
        codeLanguage = line.trim().replace(/^```/, '').trim()
        inCode = true
      }
      continue
    }
    if (inCode) {
      codeLines.push(line)
      continue
    }
    const heading = line.match(/^(#{1,3})\s+(.+)$/)
    if (heading) {
      if (inList) {
        html.push('</ul>')
        inList = false
      }
      const level = (heading[1] || '').length + 2
      html.push(`<h${level}>${inlineMarkdown(heading[2] || '')}</h${level}>`)
      continue
    }
    const item = line.match(/^\s*[-*]\s+(.+)$/)
    if (item) {
      if (!inList) {
        html.push('<ul>')
        inList = true
      }
      html.push('<li>' + inlineMarkdown(item[1] || '') + '</li>')
      continue
    }
    if (inList) {
      html.push('</ul>')
      inList = false
    }
    html.push(line.trim() ? '<p>' + inlineMarkdown(line) + '</p>' : '<br>')
  }
  if (inList) html.push('</ul>')
  if (inCode) html.push(codeBlock(codeLines.join('\n'), codeLanguage))
  return html.join('')
}

const copyCode = async (event: MouseEvent) => {
  const target = event.target as HTMLElement
  const button = target.closest<HTMLButtonElement>('.code-copy')
  if (!button?.dataset.code) return
  const code = decodeURIComponent(button.dataset.code)
  try {
    await navigator.clipboard.writeText(code)
    copiedCode.value = code
    button.textContent = '已复制'
    setTimeout(() => {
      button.textContent = '复制'
      if (copiedCode.value === code) copiedCode.value = ''
    }, 1200)
  } catch {
    message.error('复制失败', '请手动复制代码')
  }
}

const messageAttachments = (item: ApiConversationMessage) => {
  const value = item.meta?.attachments
  return Array.isArray(value) ? value as Array<{ name: string, mimeType: string, size: number }> : []
}

const readFile = (file: File) => new Promise<AssistantAttachment>((resolve, reject) => {
  const reader = new FileReader()
  reader.onerror = () => reject(new Error('附件读取失败'))
  reader.onload = () => {
    resolve({
      id: crypto.randomUUID(),
      name: file.name,
      mimeType: file.type || 'application/octet-stream',
      size: file.size,
      dataUrl: String(reader.result || ''),
      text: ''
    })
  }
  if (file.type.startsWith('text/') || /\.(md|txt|json|csv|log)$/i.test(file.name)) {
    reader.onload = () => {
      resolve({
        id: crypto.randomUUID(),
        name: file.name,
        mimeType: file.type || 'text/plain',
        size: file.size,
        dataUrl: '',
        text: String(reader.result || '').slice(0, 20000)
      })
    }
    reader.readAsText(file)
    return
  }
  reader.readAsDataURL(file)
})

const onFilesSelected = async (event: Event) => {
  const files = Array.from((event.target as HTMLInputElement).files || [])
  if (!files.length) return
  try {
    const next = await Promise.all(files.slice(0, 6).map(readFile))
    attachments.value = [...attachments.value, ...next].slice(0, 6)
  } catch (error) {
    message.error('附件失败', error instanceof Error ? error.message : '附件读取失败')
  } finally {
    if (fileInputRef.value) fileInputRef.value.value = ''
  }
}

const removeAttachment = (id: string) => {
  attachments.value = attachments.value.filter(item => item.id !== id)
}

const parseSSE = async (response: Response, handlers: {
  delta: (content: string) => void
  done: (result: ApiAssistantChatResult) => void
  error: (error: string) => void
}) => {
  const reader = response.body?.getReader()
  if (!reader) throw new Error('浏览器不支持流式读取')
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const events = buffer.split('\n\n')
    buffer = events.pop() || ''
    for (const raw of events) {
      const event = raw.split('\n').find(line => line.startsWith('event:'))?.slice(6).trim() || 'message'
      const data = raw.split('\n').filter(line => line.startsWith('data:')).map(line => line.slice(5).trim()).join('\n')
      if (!data) continue
      const payload = JSON.parse(data)
      if (event === 'delta') handlers.delta(String(payload.content || ''))
      if (event === 'done') handlers.done(payload as ApiAssistantChatResult)
      if (event === 'error') handlers.error(String(payload.error || '流式请求失败'))
    }
  }
}

const sendMessage = async () => {
  const content = input.value.trim()
  if ((!content && !attachments.value.length) || sending.value) return
  if (!auth.token.value) {
    message.error('请先登录', '登录后可以使用智能助手')
    return
  }

  const now = new Date().toISOString()
  const tempUser: ApiConversationMessage = {
    id: 'temp-user-' + Date.now(),
    conversationId: conversationID.value,
    role: 'user',
    content: content || '请分析附件内容',
    meta: {
      attachments: attachments.value.map(item => ({
        name: item.name,
        mimeType: item.mimeType,
        size: item.size
      }))
    },
    createdAt: now
  }
  const tempAssistant: ApiConversationMessage = {
    id: 'temp-assistant-' + Date.now(),
    conversationId: conversationID.value,
    role: 'assistant',
    content: '',
    meta: {},
    createdAt: now
  }

  const outgoingAttachments = attachments.value
  input.value = ''
  attachments.value = []
  sending.value = true
  messages.value = [...messages.value, tempUser, tempAssistant]

  try {
    const response = await fetch(`${config.public.apiBase}/assistant/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${auth.token.value}`
      },
      body: JSON.stringify({
        conversationId: conversationID.value,
        message: tempUser.content,
        stream: true,
        attachments: outgoingAttachments.map(({ id, ...item }) => item)
      })
    })
    if (!response.ok) {
      const data = await response.json().catch(() => null) as { error?: string } | null
      throw new Error(data?.error || '智能助手请求失败')
    }
    const finalState: { result?: ApiAssistantChatResult } = {}
    await parseSSE(response, {
      delta: (delta) => {
        tempAssistant.content += delta
        messages.value = [...messages.value]
      },
      done: (result) => {
        finalState.result = result
      },
      error: (error) => {
        throw new Error(error)
      }
    })
    const savedResult = finalState.result
    if (savedResult) {
      conversationID.value = savedResult.conversation.id
      conversationTitle.value = savedResult.conversation.title || '智能助手'
      messages.value = [
        ...messages.value.filter(item => item.id !== tempUser.id && item.id !== tempAssistant.id),
        ...savedResult.messages
      ]
      upsertConversation(savedResult.conversation)
      await loadConversations()
      if (routeConversationID.value !== savedResult.conversation.id) {
        router.replace({ query: { conversation: savedResult.conversation.id } })
      }
    }
  } catch (error) {
    messages.value = messages.value.filter(item => item.id !== tempUser.id && item.id !== tempAssistant.id)
    attachments.value = outgoingAttachments
    message.error('发送失败', error instanceof Error ? error.message : '智能助手请求失败')
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <div class="h-full min-h-0 flex flex-col bg-[#f4f7fb] dark:bg-[#070b14] overflow-hidden">
    <div
      ref="scrollRef"
      class="flex-1 min-h-0 overflow-y-auto px-3 sm:px-6 py-6"
    >
      <div class="max-w-4xl mx-auto space-y-6 pb-4">
        <UAlert
          v-if="!auth.token.value"
          color="warning"
          variant="soft"
          icon="i-lucide-lock"
          title="请先登录"
          description="登录后可以使用智能助手并同步到左侧最近会话。"
        />

        <div class="rounded-2xl border border-slate-200 bg-white shadow-[0_18px_50px_rgba(15,23,42,0.08)] px-5 py-5 sm:px-6 flex items-center justify-between gap-4 dark:border-white/10 dark:bg-[#111827]/95 dark:shadow-[0_18px_60px_rgba(0,0,0,0.35)]">
          <div class="flex items-center gap-4 min-w-0">
            <div class="h-14 w-14 shrink-0 rounded-2xl bg-blue-600 text-white grid place-items-center shadow-sm dark:bg-blue-500 dark:shadow-blue-500/20">
              <UIcon
                name="i-lucide-bot"
                class="h-8 w-8"
              />
            </div>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2 mb-1">
                <UBadge
                  color="primary"
                  variant="soft"
                  class="rounded-full"
                >
                  智能助手
                </UBadge>
                <UBadge
                  color="warning"
                  variant="soft"
                  class="rounded-full"
                >
                  推荐
                </UBadge>
                <UBadge
                  color="primary"
                  variant="soft"
                  class="rounded-full"
                >
                  gpt-5.5
                </UBadge>
              </div>
              <h2 class="text-xl sm:text-2xl font-bold text-slate-950 truncate dark:text-white">
                智能助手
              </h2>
              <p class="text-sm text-slate-500 mt-1 dark:text-slate-400">
                支持文本创作、问答、方案拆解和灵感整理，帮助你更快完成内容与任务。
              </p>
            </div>
          </div>
          <UButton
            icon="i-lucide-plus-circle"
            color="neutral"
            variant="ghost"
            size="sm"
            class="shrink-0 text-slate-900 dark:text-slate-100 dark:hover:bg-white/10"
            @click="startNewChat()"
          >
            新对话
          </UButton>
        </div>

        <div
          v-for="item in messages"
          :key="item.id"
          class="flex items-start gap-3"
          :class="item.role === 'user' ? 'justify-end' : 'justify-start'"
        >
          <div
            v-if="item.role === 'assistant'"
            class="mt-6 h-10 w-10 shrink-0 rounded-full bg-blue-600 text-white grid place-items-center dark:bg-blue-500 dark:ring-1 dark:ring-blue-300/20"
          >
            <UIcon
              name="i-lucide-bot"
              class="h-6 w-6"
            />
          </div>
          <div
            class="max-w-[86%] sm:max-w-[72%]"
            :class="item.role === 'user' ? 'text-right' : ''"
          >
            <div
              class="mb-2 text-xs text-slate-500 dark:text-slate-500"
              :class="item.role === 'user' ? 'text-right' : 'text-left'"
            >
              {{ item.role === 'user' ? '我' : '智能助手' }} / {{ formatTime(item.createdAt) }}
            </div>
            <div
              class="rounded-2xl border px-4 py-3 text-sm text-left"
              :class="item.role === 'user' ? 'bg-blue-50 border-blue-100 text-slate-950 rounded-tr-md dark:bg-blue-500/18 dark:border-blue-400/30 dark:text-slate-50' : 'bg-white border-slate-200 text-slate-950 rounded-tl-md dark:bg-[#111827] dark:border-white/10 dark:text-slate-100'"
            >
              <div
                v-if="messageAttachments(item).length"
                class="mb-3 flex flex-wrap gap-2"
              >
                <span
                  v-for="file in messageAttachments(item)"
                  :key="file.name"
                  class="inline-flex items-center gap-1.5 rounded-lg bg-default/20 border border-default/40 px-2 py-1 text-xs dark:bg-white/5 dark:border-white/10 dark:text-slate-300"
                >
                  <UIcon
                    name="i-lucide-paperclip"
                    class="h-3.5 w-3.5"
                  />
                  {{ file.name }}
                </span>
              </div>
              <div
                v-if="item.role === 'assistant'"
                class="assistant-md"
                @click="copyCode"
                v-html="renderMarkdown(item.content || (sending ? '正在思考...' : ''))"
              />
              <p
                v-else
                class="whitespace-pre-wrap leading-6"
              >
                {{ item.content }}
              </p>
            </div>
          </div>
          <div
            v-if="item.role === 'user'"
            class="mt-6 h-10 w-10 shrink-0 rounded-full overflow-hidden bg-slate-200 border border-white shadow-sm dark:bg-slate-800 dark:border-white/10"
          >
            <img
              v-if="auth.user.value?.avatarUrl"
              :src="auth.user.value.avatarUrl"
              :alt="auth.user.value.nickname || auth.user.value.email"
              class="h-full w-full object-cover"
            >
            <div
              v-else
              class="h-full w-full grid place-items-center text-sm font-bold text-slate-700 dark:text-slate-200"
            >
              {{ (auth.user.value?.nickname || auth.user.value?.email || '我').slice(0, 1).toUpperCase() }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="shrink-0 border-t border-slate-200 bg-white/90 px-3 sm:px-6 py-3 dark:border-white/10 dark:bg-[#0b1220]/95">
      <div
        v-if="attachments.length"
        class="max-w-4xl mx-auto mb-2 flex gap-2 overflow-x-auto pb-1"
      >
        <div
          v-for="file in attachments"
          :key="file.id"
          class="shrink-0 inline-flex items-center gap-2 rounded-lg border border-blue-100 bg-blue-50 px-3 py-2 text-xs max-w-64 dark:border-blue-400/20 dark:bg-blue-500/10"
        >
          <UIcon
            :name="file.mimeType.startsWith('image/') ? 'i-lucide-image' : 'i-lucide-file-text'"
            class="h-4 w-4 text-primary"
          />
          <span class="min-w-0">
            <span class="block truncate text-highlighted">{{ file.name }}</span>
            <span class="block text-dimmed">{{ formatSize(file.size) }}</span>
          </span>
          <button
            type="button"
            class="grid h-6 w-6 place-items-center rounded-md hover:bg-white dark:hover:bg-white/10"
            @click="removeAttachment(file.id)"
          >
            <UIcon
              name="i-lucide-x"
              class="h-3.5 w-3.5"
            />
          </button>
        </div>
      </div>

      <form
        class="max-w-4xl mx-auto rounded-2xl border-2 border-blue-500 bg-white px-3 py-3 shadow-sm space-y-3 dark:border-blue-400/70 dark:bg-[#111827] dark:shadow-[0_18px_50px_rgba(0,0,0,0.35)]"
        @submit.prevent="sendMessage()"
      >
        <div class="flex items-end gap-2">
          <input
            ref="fileInputRef"
            type="file"
            class="hidden"
            multiple
            accept="image/*,.txt,.md,.json,.csv,.log"
            @change="onFilesSelected"
          >
          <UButton
            type="button"
            icon="i-lucide-paperclip"
            color="neutral"
            variant="ghost"
            :disabled="sending || !auth.token.value"
            @click="fileInputRef?.click()"
          />
          <UTextarea
            v-model="input"
            :rows="2"
            autoresize
            class="flex-1"
            :ui="{ base: 'border-0 shadow-none resize-none focus:ring-0 bg-transparent dark:text-slate-100 dark:placeholder:text-slate-500' }"
            placeholder="输入你想聊的问题、文案或方案"
            :disabled="sending || !auth.token.value"
            @keydown.enter.exact.prevent="sendMessage()"
          />
          <UButton
            type="submit"
            icon="i-lucide-arrow-up"
            :loading="sending"
            class="rounded-full px-5 bg-blue-500 hover:bg-blue-600"
            :disabled="(!input.trim() && !attachments.length) || !auth.token.value"
          >
            发送
          </UButton>
        </div>
        <div class="flex flex-wrap items-center gap-2 border-t border-slate-100 pt-2 text-xs dark:border-white/10">
          <span class="rounded-full border border-slate-200 px-2 py-1 text-slate-600 dark:border-white/10 dark:text-slate-400">{{ input.length }} / 4000 字</span>
          <span class="rounded-full border border-blue-200 bg-blue-50 px-2 py-1 text-blue-600 dark:border-blue-400/20 dark:bg-blue-500/10 dark:text-blue-300">gpt-5.5</span>
          <span class="rounded-full border border-blue-200 bg-blue-50 px-2 py-1 text-blue-600 dark:border-blue-400/20 dark:bg-blue-500/10 dark:text-blue-300">可以发送</span>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.assistant-md :deep(p) {
  margin: 0.35rem 0;
  line-height: 1.7;
}

.assistant-md :deep(h3),
.assistant-md :deep(h4),
.assistant-md :deep(h5) {
  margin: 0.75rem 0 0.35rem;
  font-weight: 700;
}

.assistant-md :deep(ul) {
  margin: 0.35rem 0;
  padding-left: 1.2rem;
  list-style: disc;
}

.assistant-md :deep(.code-card) {
  margin: 0.75rem 0;
  overflow: hidden;
  border: 1px solid #dbe3ef;
  border-radius: 0.75rem;
  background: #0f172a;
}

.assistant-md :deep(.code-toolbar) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  border-bottom: 1px solid rgba(148, 163, 184, 0.25);
  color: #cbd5e1;
  font-size: 0.75rem;
}

.assistant-md :deep(.code-copy) {
  border-radius: 999px;
  padding: 0.2rem 0.65rem;
  color: #bfdbfe;
  background: rgba(37, 99, 235, 0.22);
}

.assistant-md :deep(.code-copy:hover) {
  background: rgba(37, 99, 235, 0.36);
}

.assistant-md :deep(pre) {
  margin: 0;
  padding: 0.9rem;
  overflow-x: auto;
  color: #e2e8f0;
}

.assistant-md :deep(code) {
  font-size: 0.85em;
  border-radius: 0.35rem;
  padding: 0.1rem 0.25rem;
  background: #eef4ff;
  color: #1e40af;
}

.assistant-md :deep(pre code) {
  padding: 0;
  background: transparent;
  color: inherit;
}

.assistant-md :deep(a) {
  color: #2563eb;
  text-decoration: underline;
}

:global(.dark) .assistant-md :deep(.code-card) {
  border-color: rgba(255, 255, 255, 0.1);
  background: #020617;
}

:global(.dark) .assistant-md :deep(.code-toolbar) {
  border-bottom-color: rgba(148, 163, 184, 0.18);
  color: #94a3b8;
}

:global(.dark) .assistant-md :deep(.code-copy) {
  color: #bfdbfe;
  background: rgba(59, 130, 246, 0.18);
}

:global(.dark) .assistant-md :deep(.code-copy:hover) {
  background: rgba(59, 130, 246, 0.28);
}

:global(.dark) .assistant-md :deep(code) {
  background: rgba(59, 130, 246, 0.14);
  color: #bfdbfe;
}

:global(.dark) .assistant-md :deep(pre code) {
  background: transparent;
  color: inherit;
}

:global(.dark) .assistant-md :deep(a) {
  color: #93c5fd;
}
</style>
