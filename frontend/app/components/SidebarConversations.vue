<script setup lang="ts">
import type { ApiConversation } from '~/composables/useApi'

// 侧边栏下半部:最近对话 / 助手列表 切换 + 空态
const auth = useAuth()
const { conversations, pending, loadConversations } = useConversations()
const tabs = [
  { key: 'recent', label: '最近对话', icon: 'i-lucide-message-square' },
  { key: 'assistants', label: '助手列表', icon: 'i-lucide-users' }
]
const activeTab = ref('recent')

onMounted(() => {
  loadConversations()
})

watch(
  () => auth.token.value,
  () => loadConversations()
)

const shortTitle = (item: ApiConversation) => item.title.length > 18 ? item.title.slice(0, 18) + '...' : item.title
const conversationTo = (item: ApiConversation) => item.kind === 'assistant' ? '/assistant?conversation=' + item.id : '/draw-chat/' + item.id
const conversationIcon = (item: ApiConversation) => item.kind === 'assistant' ? 'i-lucide-bot' : 'i-lucide-image'
const conversationSubtitle = (item: ApiConversation) => item.kind === 'assistant' ? '智能助手' : item.appName || '自定义绘画'
</script>

<template>
  <div class="flex flex-col flex-1 min-h-0">
    <!-- tab 切换 -->
    <div class="flex items-center gap-1 px-1 py-1 rounded-lg bg-elevated/60">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        type="button"
        class="flex items-center justify-center gap-1.5 flex-1 px-2 py-1.5 rounded-md text-xs font-medium transition-colors"
        :class="activeTab === tab.key
          ? 'bg-default text-highlighted shadow-sm'
          : 'text-muted hover:text-highlighted'"
        @click="activeTab = tab.key"
      >
        <UIcon
          :name="tab.icon"
          class="w-4 h-4"
        />
        {{ tab.label }}
      </button>
    </div>

    <!-- 过滤行 -->
    <div class="flex items-center justify-between mt-3 px-1">
      <UBadge
        color="primary"
        variant="soft"
        size="sm"
        class="rounded-full"
      >
        全部
      </UBadge>
      <UButton
        icon="i-lucide-more-horizontal"
        color="neutral"
        variant="ghost"
        size="xs"
        aria-label="更多"
      />
    </div>

    <div
      v-if="activeTab === 'recent' && conversations.length"
      class="mt-3 space-y-1 overflow-y-auto pr-1"
    >
      <NuxtLink
        v-for="item in conversations"
        :key="item.id"
        :to="conversationTo(item)"
        class="flex items-start gap-2 rounded-lg px-2 py-2 text-sm transition-colors hover:bg-elevated"
      >
        <span class="mt-0.5 h-7 w-7 rounded-lg bg-primary/10 text-primary grid place-items-center shrink-0">
          <UIcon
            :name="conversationIcon(item)"
            class="h-4 w-4"
          />
        </span>
        <span class="min-w-0">
          <span class="block font-medium text-highlighted truncate">{{ shortTitle(item) }}</span>
          <span class="block text-xs text-dimmed truncate">{{ conversationSubtitle(item) }}</span>
        </span>
      </NuxtLink>
    </div>

    <div
      v-else-if="pending"
      class="mt-3 space-y-2"
    >
      <USkeleton
        v-for="item in 3"
        :key="item"
        class="h-11 rounded-lg"
      />
    </div>

    <!-- 空态 -->
    <div
      v-else
      class="flex flex-col items-center justify-center flex-1 gap-3 py-10 text-center rounded-xl border border-dashed border-default mt-3"
    >
      <div class="flex items-center justify-center w-12 h-12 rounded-full bg-elevated text-dimmed">
        <UIcon
          name="i-lucide-message-circle"
          class="w-6 h-6"
        />
      </div>
      <div class="space-y-1">
        <p class="text-sm font-medium text-toned">
          暂无最近对话
        </p>
        <p class="text-xs text-dimmed">
          登录后会加载你的会话记录。
        </p>
      </div>
      <UButton
        color="primary"
        variant="link"
        size="sm"
        trailing-icon="i-lucide-arrow-right"
      >
        去生成
      </UButton>
    </div>
  </div>
</template>
