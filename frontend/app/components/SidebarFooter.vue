<script setup lang="ts">
const api = useApi()
const auth = useAuth()
const message = useMessage()
const { loadConversations, clearConversations } = useConversations()
const clearing = ref(false)
const clearOpen = ref(false)

const clearDrawList = async () => {
  if (!auth.token.value) return
  try {
    clearing.value = true
    try {
      await api.post<{ ok: boolean }>('/conversations/clear-draw')
    } catch (error) {
      const text = error instanceof Error ? error.message : ''
      if (!text.includes('405') && !text.includes('Method Not Allowed')) throw error
      await api.delete<{ ok: boolean }>('/conversations/clear-draw')
    }
    await loadConversations()
    clearConversations()
    clearOpen.value = false
    message.success('清空成功', '所有会话已清除')
  } catch (error) {
    message.error('清空失败', error instanceof Error ? error.message : '清空会话失败')
  } finally {
    clearing.value = false
  }
}
</script>

<template>
  <div class="flex flex-col gap-2 pt-3 border-t border-default">
    <UButton
      color="neutral"
      variant="ghost"
      size="sm"
      icon="i-lucide-info"
      class="justify-center"
    >
      关于我们
    </UButton>
    <div class="grid grid-cols-2 gap-2">
      <UButton
        color="neutral"
        variant="ghost"
        size="sm"
        icon="i-lucide-trash-2"
        class="justify-center"
        :disabled="!auth.token.value"
        @click="clearOpen = true"
      >
        清空列表
      </UButton>
      <UButton
        color="neutral"
        variant="soft"
        icon="i-lucide-plus-circle"
        class="justify-center"
      >
        自定应用
      </UButton>
    </div>

    <UModal
      v-model:open="clearOpen"
      :ui="{ content: 'max-w-md rounded-2xl' }"
    >
      <template #content>
        <div class="p-6">
          <div class="flex items-start gap-3">
            <div class="h-11 w-11 shrink-0 rounded-full bg-amber-100 text-amber-600 grid place-items-center">
              <UIcon
                name="i-lucide-exclamation-circle"
                class="h-6 w-6"
              />
            </div>
            <div class="min-w-0">
              <h3 class="text-lg font-bold text-highlighted">
                清空列表
              </h3>
              <p class="mt-2 text-sm text-toned">
                确定要清空所有会话吗？
              </p>
            </div>
            <button
              type="button"
              class="ml-auto -mt-1 text-dimmed hover:text-highlighted"
              @click="clearOpen = false"
            >
              <UIcon
                name="i-lucide-x"
                class="h-5 w-5"
              />
            </button>
          </div>

          <div class="mt-6 flex justify-end gap-2">
            <UButton
              color="neutral"
              variant="soft"
              @click="clearOpen = false"
            >
              取消
            </UButton>
            <UButton
              color="warning"
              :loading="clearing"
              @click="clearDrawList"
            >
              清空列表
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
