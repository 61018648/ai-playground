<script setup lang="ts">
import type { ApiApp } from '~/composables/useApi'

const api = useApi()
const { styleTabs, galleryWorks } = useHomeData()

const activeTab = ref('全部')
const prompt = ref('')
const { data: apps } = await useAsyncData('gallery-target-apps', () => api.get<ApiApp[]>('/apps'), {
  default: () => []
})

const filteredWorks = computed(() => {
  if (activeTab.value === '全部') return galleryWorks
  return galleryWorks.filter(work => work.category === activeTab.value)
})

const appByCode = computed(() => new Map(apps.value.map(app => [app.code, app])))

const targetAppCode = (workCode: string) => workCode

const workTargetTo = (work: (typeof galleryWorks)[number]) => {
  const app = appByCode.value.get(work.targetAppCode)
  const query = { prompt: work.samplePrompt }
  if (app) return { path: `/apps/${app.id}`, query }
  return work.targetAppCode === 'xiaohongshu-copy' || work.targetAppCode === 'wechat-title'
    ? { path: '/assistant', query }
    : { path: '/draw', query }
}

const createFromPrompt = computed(() => ({
  path: '/draw',
  query: prompt.value.trim() ? { prompt: prompt.value.trim() } : undefined
}))
</script>

<template>
  <section class="space-y-5">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <div class="flex items-center gap-2">
          <UIcon
            name="i-lucide-images"
            class="h-5 w-5 text-primary"
          />
          <h2 class="text-xl font-bold text-highlighted">
            画廊广场
          </h2>
        </div>
        <p class="text-sm text-dimmed mt-1">
          精选可复用的运营案例，查看提示词后可一键生成同款。
        </p>
      </div>
      <UButton
        to="/apps"
        color="neutral"
        variant="soft"
        trailing-icon="i-lucide-arrow-right"
        class="self-start sm:self-auto"
      >
        查看全部应用
      </UButton>
    </div>

    <div class="flex gap-2 overflow-x-auto pb-1">
      <button
        v-for="tab in styleTabs"
        :key="tab"
        type="button"
        class="h-9 shrink-0 rounded-full px-4 text-sm font-medium transition-colors"
        :class="tab === activeTab
          ? 'bg-primary text-inverted'
          : 'border border-default bg-default text-muted hover:text-highlighted'"
        @click="activeTab = tab"
      >
        {{ tab }}
      </button>
    </div>

    <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="work in filteredWorks"
        :key="work.id"
        class="group overflow-hidden rounded-lg border border-default bg-default shadow-sm transition hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md"
      >
        <div class="relative aspect-[4/3] overflow-hidden bg-elevated">
          <img
            :src="work.imageUrl"
            :alt="work.title"
            class="h-full w-full object-cover transition duration-500 group-hover:scale-105"
            loading="lazy"
          >
          <div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/75 via-black/20 to-transparent p-4 text-white">
            <div class="flex items-center gap-2">
              <span
                class="h-2 w-2 rounded-full"
                :class="work.accent"
              />
              <span class="text-xs font-medium">{{ work.category }}</span>
              <UBadge
                v-if="work.featured"
                color="warning"
                variant="solid"
                size="sm"
                class="rounded"
              >
                精选
              </UBadge>
            </div>
            <h3 class="mt-2 text-lg font-bold leading-tight">
              {{ work.title }}
            </h3>
            <p class="mt-1 text-sm text-white/80">
              {{ work.subtitle }}
            </p>
          </div>
        </div>
        <div class="space-y-3 p-4">
          <p class="line-clamp-2 text-sm text-dimmed">
            {{ work.samplePrompt }}
          </p>
          <div class="flex items-center justify-between gap-3">
            <UBadge
              color="neutral"
              variant="soft"
              class="rounded"
            >
              {{ targetAppCode(work.targetAppCode) }}
            </UBadge>
            <UButton
              :to="workTargetTo(work)"
              color="primary"
              size="sm"
              icon="i-lucide-sparkles"
              class="shrink-0"
            >
              一键同款
            </UButton>
          </div>
        </div>
      </article>
    </div>

    <div class="mx-auto flex w-full max-w-2xl items-center gap-2 rounded-lg border border-default bg-default p-2 shadow-sm">
      <UInput
        v-model="prompt"
        variant="none"
        size="lg"
        placeholder="输入你的创意，直接进入专业绘图"
        :ui="{ root: 'flex-1', base: 'bg-transparent' }"
      />
      <UButton
        color="primary"
        size="md"
        icon="i-lucide-send"
        class="shrink-0"
        :to="createFromPrompt"
      >
        生成
      </UButton>
    </div>
  </section>
</template>
