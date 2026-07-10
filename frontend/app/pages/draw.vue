<script setup lang="ts">
import type { DrawCategory } from '~/composables/useDrawData'

const { pageTitle } = useSiteConfig()

useHead(() => ({ title: pageTitle('专业绘画') }))

const { categories } = useDrawData()
const route = useRoute()

const prompt = ref('')
const selectedId = ref<number | null>(null)
const routePrompt = computed(() => {
  const value = route.query.prompt
  return Array.isArray(value) ? String(value[0] || '') : String(value || '')
})

const selectCategory = (category: DrawCategory) => {
  if (selectedId.value === category.id) {
    // 再次点击取消选中并清空预设提示词
    selectedId.value = null
    if (prompt.value === category.prompt) prompt.value = ''
    return
  }
  selectedId.value = category.id
  prompt.value = category.prompt
}

watch(
  routePrompt,
  (value) => {
    if (value && !prompt.value.trim()) {
      prompt.value = value
    }
  },
  { immediate: true }
)
</script>

<template>
  <div class="max-w-5xl mx-auto pt-6 sm:pt-10">
    <!-- 标题 -->
    <h1 class="text-2xl sm:text-3xl font-bold text-center text-highlighted">
      AI团队 - 让商业设计 好看又见效
    </h1>

    <!-- 对话输入区 -->
    <div class="mt-8">
      <DrawComposer v-model:prompt="prompt" />
    </div>

    <!-- 分类卡片 -->
    <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4 mt-4">
      <DrawCategoryCard
        v-for="category in categories"
        :key="category.id"
        :category="category"
        :selected="selectedId === category.id"
        @select="selectCategory(category)"
      />
    </div>
  </div>
</template>
