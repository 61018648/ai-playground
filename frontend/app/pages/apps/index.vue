<script setup lang="ts">
import type { ApiApp } from '~/composables/useApi'

const { pageTitle } = useSiteConfig()

useHead(() => ({ title: pageTitle('应用中心') }))

const api = useApi()
const { data: apps, pending, error } = await useAsyncData('apps-page', () => api.get<ApiApp[]>('/apps'), {
  default: () => []
})
</script>

<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 py-6 space-y-5">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-bold text-highlighted">
          应用中心
        </h1>
        <p class="text-sm text-dimmed mt-1">
          选择一个创作应用，进入专属对话窗口生成图片或内容。
        </p>
      </div>
    </div>

    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-alert-circle"
      title="应用列表加载失败"
      description="请联系网站管理员处理。"
    />

    <div
      v-if="pending"
      class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4"
    >
      <USkeleton
        v-for="item in 6"
        :key="item"
        class="h-32 rounded-lg"
      />
    </div>

    <div
      v-else
      class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4"
    >
      <AppCard
        v-for="app in apps"
        :key="app.id"
        :app="app"
      />
    </div>
  </div>
</template>
