<script setup lang="ts">
import type { ApiMediaAsset } from '~/composables/useApi'

const { pageTitle } = useSiteConfig()

useHead(() => ({ title: pageTitle('媒体库') }))

const api = useApi()
const auth = useAuth()
const route = useRoute()
const activeTab = ref<'all' | 'favorites'>('all')
const previewOpen = ref(false)
const selectedAsset = ref<ApiMediaAsset | null>(null)

onMounted(() => {
  auth.loadMe()
  activeTab.value = route.query.tab === 'favorites' ? 'favorites' : 'all'
})

watch(
  () => route.query.tab,
  (tab) => {
    activeTab.value = tab === 'favorites' ? 'favorites' : 'all'
  }
)

const mediaPath = computed(() => '/media?limit=80' + (activeTab.value === 'favorites' ? '&favorite=1' : ''))
const { data: assets, pending, error, refresh } = await useAsyncData(
  'media-library',
  () => auth.token.value ? api.get<ApiMediaAsset[]>(mediaPath.value) : Promise.resolve([]),
  {
    default: () => [],
    watch: [auth.token, activeTab]
  }
)

const openPreview = (asset: ApiMediaAsset) => {
  selectedAsset.value = asset
  previewOpen.value = true
}

const onFavoriteChanged = (assetId: string, isFavorite: boolean) => {
  assets.value = assets.value.map(item => item.id === assetId ? { ...item, isFavorite } : item)
  if (activeTab.value === 'favorites' && !isFavorite) {
    assets.value = assets.value.filter(item => item.id !== assetId)
  }
}
</script>

<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 py-6 space-y-5">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-xl font-bold text-highlighted">
          媒体库
        </h1>
        <p class="text-sm text-dimmed mt-1">
          管理生成图片，放大查看、下载到本地和收藏作品。
        </p>
      </div>
    </div>

    <div class="flex items-center gap-1 rounded-lg border border-default bg-default p-1 w-fit">
      <UButton
        :color="activeTab === 'all' ? 'primary' : 'neutral'"
        variant="soft"
        size="sm"
        @click="activeTab = 'all'"
      >
        全部作品
      </UButton>
      <UButton
        :color="activeTab === 'favorites' ? 'primary' : 'neutral'"
        variant="soft"
        size="sm"
        icon="i-lucide-heart"
        @click="activeTab = 'favorites'"
      >
        我的收藏
      </UButton>
    </div>

    <UAlert
      v-if="!auth.token.value"
      color="warning"
      variant="soft"
      icon="i-lucide-lock"
      title="请先登录"
      description="登录后会加载你的媒体库作品。"
    />

    <UAlert
      v-else-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-alert-circle"
      title="媒体库加载失败"
      description="请确认后端服务正在运行，并且登录状态有效。"
    />

    <div
      v-if="pending"
      class="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-4"
    >
      <USkeleton
        v-for="item in 8"
        :key="item"
        class="aspect-square rounded-lg"
      />
    </div>

    <div
      v-else-if="auth.token.value && !assets.length"
      class="flex min-h-[22rem] flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-default text-center"
    >
      <UIcon
        name="i-lucide-images"
        class="h-10 w-10 text-dimmed"
      />
      <div>
        <p class="font-medium text-highlighted">
          暂无作品
        </p>
        <p class="text-sm text-dimmed mt-1">
          生成图片后会自动出现在这里。
        </p>
      </div>
    </div>

    <div
      v-else
      class="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-4"
    >
      <button
        v-for="asset in assets"
        :key="asset.id"
        type="button"
        class="group overflow-hidden rounded-lg border border-default bg-default text-left transition hover:-translate-y-0.5 hover:shadow-lg"
        @click="openPreview(asset)"
      >
        <div class="relative aspect-square bg-elevated">
          <img
            :src="asset.thumbnailUrl || asset.url"
            :alt="asset.prompt"
            class="h-full w-full object-cover"
          >
          <div class="absolute inset-x-0 bottom-0 flex items-center justify-between bg-gradient-to-t from-black/70 to-transparent px-3 py-3 opacity-0 transition group-hover:opacity-100">
            <span class="text-xs font-medium text-white truncate">{{ asset.model || 'AI Image' }}</span>
            <UIcon
              v-if="asset.isFavorite"
              name="i-lucide-heart"
              class="h-4 w-4 text-rose-300"
            />
          </div>
        </div>
        <div class="p-3">
          <p class="line-clamp-2 text-sm text-highlighted">
            {{ asset.prompt }}
          </p>
          <p class="mt-2 text-xs text-dimmed">
            {{ new Date(asset.generatedAt).toLocaleString('zh-CN') }}
          </p>
        </div>
      </button>
    </div>

    <MediaPreviewModal
      v-model:open="previewOpen"
      :asset="selectedAsset"
      @favorite-changed="onFavoriteChanged"
    />
  </div>
</template>
