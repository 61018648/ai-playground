<script setup lang="ts">
const { banners } = useHomeData()

const current = ref(0)

const prev = () => {
  current.value = (current.value - 1 + banners.length) % banners.length
}
const next = () => {
  current.value = (current.value + 1) % banners.length
}
</script>

<template>
  <section class="space-y-3">
    <!-- banner 行:三张并排展示 -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div
        v-for="banner in banners"
        :key="banner.id"
        class="group relative overflow-hidden rounded-2xl bg-gradient-to-br p-5 h-44 flex flex-col justify-between cursor-pointer transition-transform hover:-translate-y-0.5"
        :class="banner.gradient"
      >
        <!-- 标签 -->
        <div class="flex flex-wrap gap-x-1.5 gap-y-1 text-[11px] font-medium text-slate-700/80">
          <template
            v-for="(tag, i) in banner.tags"
            :key="tag"
          >
            <span>{{ tag }}</span>
            <span
              v-if="i < banner.tags.length - 1"
              class="opacity-40"
            >|</span>
          </template>
        </div>

        <!-- 主标题 -->
        <h3 class="text-2xl font-bold text-slate-900 drop-shadow-sm">
          {{ banner.title }}
        </h3>

        <!-- 胶囊 -->
        <span class="inline-flex self-start items-center px-3 py-1.5 rounded-lg bg-slate-900/85 text-white text-xs font-medium">
          {{ banner.pill }}
        </span>
      </div>
    </div>

    <!-- 控制条:箭头 + 圆点 -->
    <div class="flex items-center justify-center gap-3">
      <UButton
        icon="i-lucide-chevron-left"
        color="neutral"
        variant="ghost"
        size="xs"
        :ui="{ base: 'rounded-full' }"
        aria-label="上一组"
        @click="prev"
      />
      <div class="flex items-center gap-1.5">
        <button
          v-for="(banner, index) in banners"
          :key="banner.id"
          type="button"
          class="h-1.5 rounded-full transition-all"
          :class="index === current ? 'w-6 bg-primary' : 'w-3 bg-default border border-muted'"
          :aria-label="`切换到第 ${index + 1} 组`"
          @click="current = index"
        />
      </div>
      <UButton
        icon="i-lucide-chevron-right"
        color="neutral"
        variant="ghost"
        size="xs"
        :ui="{ base: 'rounded-full' }"
        aria-label="下一组"
        @click="next"
      />
    </div>
  </section>
</template>
