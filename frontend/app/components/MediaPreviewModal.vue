<script setup lang="ts">
import type { ApiAsset, ApiMediaAsset } from '~/composables/useApi'

type PreviewAsset = (ApiAsset | ApiMediaAsset) & {
  prompt?: string
  appName?: string
  model?: string
  isFavorite?: boolean
}

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{
  asset: PreviewAsset | null
}>()
const emit = defineEmits<{
  favoriteChanged: [assetId: string, isFavorite: boolean]
}>()

const api = useApi()
const message = useMessage()
const favoriting = ref(false)
const localFavorite = ref(false)

watch(
  () => props.asset?.id,
  () => {
    localFavorite.value = Boolean(props.asset?.isFavorite)
  },
  { immediate: true }
)

const filename = computed(() => {
  const id = props.asset?.id || Date.now().toString()
  const mime = props.asset?.mimeType || 'image/png'
  const ext = mime.includes('jpeg') || mime.includes('jpg') ? 'jpg' : 'png'
  return `image-ai-${id.slice(0, 8)}.${ext}`
})

const downloadImage = () => {
  if (!props.asset?.url) return
  const link = document.createElement('a')
  link.href = props.asset.url
  link.download = filename.value
  link.rel = 'noopener'
  document.body.appendChild(link)
  link.click()
  link.remove()
}

const toggleFavorite = async () => {
  if (!props.asset?.id || favoriting.value) return
  favoriting.value = true
  try {
    if (localFavorite.value) {
      await api.delete<{ ok: boolean }>('/media/' + props.asset.id + '/favorite')
      localFavorite.value = false
      message.success('已取消收藏', '作品已从收藏中移除')
    } else {
      await api.post<{ ok: boolean }>('/media/' + props.asset.id + '/favorite')
      localFavorite.value = true
      message.success('收藏成功', '作品已加入媒体库收藏')
    }
    emit('favoriteChanged', props.asset.id, localFavorite.value)
  } catch (error) {
    message.error('操作失败', error instanceof Error ? error.message : '收藏状态更新失败')
  } finally {
    favoriting.value = false
  }
}
</script>

<template>
  <UModal
    v-model:open="open"
    :ui="{ content: 'max-w-6xl overflow-hidden rounded-2xl p-0' }"
  >
    <template #content>
      <div class="grid max-h-[86vh] grid-cols-1 lg:grid-cols-[minmax(0,1fr)_20rem] bg-default">
        <div class="min-h-[22rem] bg-black grid place-items-center overflow-auto">
          <img
            v-if="asset?.url"
            :src="asset.url"
            :alt="asset.prompt || '生成图片'"
            class="max-h-[82vh] w-auto max-w-full object-contain"
          >
        </div>

        <aside class="flex min-h-0 flex-col border-l border-default">
          <div class="flex items-center justify-between gap-3 border-b border-default px-5 py-4">
            <div>
              <p class="text-sm font-semibold text-highlighted">
                作品预览
              </p>
              <p class="text-xs text-dimmed mt-0.5">
                {{ asset?.appName || 'AI 绘画作图' }}
              </p>
            </div>
            <UButton
              icon="i-lucide-x"
              color="neutral"
              variant="ghost"
              aria-label="关闭"
              @click="open = false"
            />
          </div>

          <div class="flex-1 min-h-0 overflow-y-auto px-5 py-4 space-y-4">
            <div>
              <p class="text-xs font-medium text-dimmed">
                提示词
              </p>
              <p class="mt-2 text-sm leading-6 text-highlighted whitespace-pre-wrap">
                {{ asset?.prompt || '暂无提示词' }}
              </p>
            </div>
            <div class="grid grid-cols-2 gap-3 text-sm">
              <div class="rounded-lg bg-elevated px-3 py-2">
                <p class="text-xs text-dimmed">
                  模型
                </p>
                <p class="mt-1 truncate text-highlighted">
                  {{ asset?.model || '-' }}
                </p>
              </div>
              <div class="rounded-lg bg-elevated px-3 py-2">
                <p class="text-xs text-dimmed">
                  格式
                </p>
                <p class="mt-1 truncate text-highlighted">
                  {{ asset?.mimeType || 'image/png' }}
                </p>
              </div>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-2 border-t border-default p-4">
            <UButton
              icon="i-lucide-download"
              color="neutral"
              variant="soft"
              block
              @click="downloadImage"
            >
              下载
            </UButton>
            <UButton
              :icon="localFavorite ? 'i-lucide-heart-off' : 'i-lucide-heart'"
              :color="localFavorite ? 'warning' : 'primary'"
              :loading="favoriting"
              block
              @click="toggleFavorite"
            >
              {{ localFavorite ? '取消收藏' : '收藏' }}
            </UButton>
          </div>
        </aside>
      </div>
    </template>
  </UModal>
</template>
