<script setup lang="ts">
import type { ApiApp } from '~/composables/useApi'

const api = useApi()
const { apps: fallbackApps } = useHomeData()
const { data, pending, error } = await useAsyncData('home-apps', () => api.get<ApiApp[]>('/apps'), {
  default: () => []
})
const apps = computed(() => data.value.length > 0 ? data.value : fallbackApps)

// 每页展示的卡片数,按页轮播
const perPage = 4

const pages = computed(() => {
  const result = []
  for (let i = 0; i < apps.value.length; i += perPage) {
    result.push(apps.value.slice(i, i + perPage))
  }
  return result
})

const current = ref(0)

const prev = () => {
  current.value = (current.value - 1 + pages.value.length) % pages.value.length
}
const next = () => {
  current.value = (current.value + 1) % pages.value.length
}
</script>

<template>
  <section class="space-y-4">
    <!-- 标题行 -->
    <div class="flex items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <h2 class="text-lg font-bold text-highlighted shrink-0">
          应用广场
        </h2>
        <UBadge
          v-if="pending"
          color="neutral"
          variant="soft"
          size="sm"
        >
          加载中
        </UBadge>
        <UBadge
          v-else-if="error"
          color="warning"
          variant="soft"
          size="sm"
        >
          本地演示
        </UBadge>
      </div>
      <UButton
        color="warning"
        variant="soft"
        size="sm"
        trailing-icon="i-lucide-arrow-right"
        class="rounded-full shrink-0"
        to="/apps"
      >
        <span class="hidden sm:inline">查看全部应用与创作模板</span>
        <span class="sm:hidden">全部应用</span>
      </UButton>
    </div>

    <!-- 轮播视窗 -->
    <div class="relative">
      <div class="overflow-hidden">
        <div
          class="flex transition-transform duration-300 ease-out"
          :style="{ transform: `translateX(-${current * 100}%)` }"
        >
          <div
            v-for="(group, index) in pages"
            :key="index"
            class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 w-full shrink-0"
          >
            <AppCard
              v-for="app in group"
              :key="app.id"
              :app="app"
            />
          </div>
        </div>
      </div>

      <!-- 两侧切换箭头 -->
      <template v-if="pages.length > 1">
        <UButton
          icon="i-lucide-chevron-left"
          color="neutral"
          variant="solid"
          size="sm"
          class="absolute left-1 lg:left-0 top-1/2 lg:-translate-x-1/2 -translate-y-1/2 rounded-full shadow-md z-10"
          aria-label="上一页"
          @click="prev"
        />
        <UButton
          icon="i-lucide-chevron-right"
          color="neutral"
          variant="solid"
          size="sm"
          class="absolute right-1 lg:right-0 top-1/2 lg:translate-x-1/2 -translate-y-1/2 rounded-full shadow-md z-10"
          aria-label="下一页"
          @click="next"
        />
      </template>
    </div>

    <!-- 圆点 -->
    <div
      v-if="pages.length > 1"
      class="flex items-center justify-center gap-1.5"
    >
      <button
        v-for="(group, index) in pages"
        :key="index"
        type="button"
        class="h-1.5 rounded-full transition-all"
        :class="index === current ? 'w-6 bg-primary' : 'w-3 bg-default border border-muted'"
        :aria-label="`切换到第 ${index + 1} 页`"
        @click="current = index"
      />
    </div>
  </section>
</template>
