<script setup lang="ts">
const { messages } = useMessage()

type AlertColor = 'error' | 'primary' | 'success' | 'neutral'

const colorOf = (type: string): AlertColor => ({
  success: 'success',
  error: 'error',
  info: 'primary'
}[type] as AlertColor || 'neutral')

const iconOf = (type: string) => ({
  success: 'i-lucide-check-circle',
  error: 'i-lucide-alert-circle',
  info: 'i-lucide-info'
}[type] || 'i-lucide-info')
</script>

<template>
  <Teleport to="body">
    <div class="fixed top-4 right-4 z-[999999999] w-[min(24rem,calc(100vw-2rem))] space-y-2 pointer-events-none">
      <TransitionGroup
        enter-active-class="transition duration-200 ease-out"
        enter-from-class="opacity-0 translate-y-2"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition duration-150 ease-in"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 translate-y-2"
      >
        <UAlert
          v-for="message in messages"
          :key="message.id"
          :color="colorOf(message.type)"
          variant="soft"
          :icon="iconOf(message.type)"
          :title="message.title"
          :description="message.description"
          class="pointer-events-auto shadow-2xl"
        />
      </TransitionGroup>
    </div>
  </Teleport>
</template>
