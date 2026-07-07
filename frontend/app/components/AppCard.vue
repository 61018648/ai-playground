<script setup lang="ts">
import type { ApiApp } from '~/composables/useApi'
import type { AppItem } from '~/composables/useHomeData'

const props = defineProps<{ app: AppItem | ApiApp }>()

const appDetailTo = computed(() => {
  const id = String(props.app.id || '')
  if (id.includes('-')) return `/apps/${id}`
  return props.app.appType === 'text' ? '/assistant' : '/draw'
})
</script>

<template>
  <NuxtLink
    :to="appDetailTo"
    class="group flex items-start gap-3 w-full p-4 rounded-xl border border-default bg-default hover:border-primary/40 hover:shadow-sm transition-all cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
  >
    <div
      class="flex items-center justify-center w-10 h-10 rounded-lg shrink-0"
      :class="app.iconColor"
    >
      <UIcon
        :name="app.icon"
        class="w-5 h-5"
      />
    </div>
    <div class="min-w-0 space-y-1">
      <div class="flex items-center gap-2">
        <h4 class="text-sm font-semibold text-highlighted truncate">
          {{ app.name }}
        </h4>
        <UBadge
          color="neutral"
          variant="soft"
          size="sm"
          class="shrink-0 rounded"
        >
          {{ app.category }}
        </UBadge>
      </div>
      <p class="text-xs text-dimmed line-clamp-2">
        {{ app.description }}
      </p>
    </div>
  </NuxtLink>
</template>
