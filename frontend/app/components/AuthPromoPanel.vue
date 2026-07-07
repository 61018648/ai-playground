<script setup lang="ts">
// 左侧宣传面板的散落标签,position 用百分比定位以便自适应
interface FloatTag {
  label: string
  icon?: string
  class: string
  style: Record<string, string>
}

const tags: FloatTag[] = [
  {
    label: '无限创意一键落地',
    icon: 'i-lucide-sparkles',
    class: 'bg-violet-500 text-white',
    style: { top: '46%', left: '50%', transform: 'translateX(-50%)' }
  },
  {
    label: '提示词优化',
    icon: 'i-lucide-edit-3',
    class: 'bg-sky-200 text-sky-900',
    style: { top: '58%', left: '24%' }
  },
  {
    label: '一站式高效改图',
    icon: 'i-lucide-image',
    class: 'bg-pink-400 text-white',
    style: { top: '57%', right: '12%' }
  },
  {
    label: '灵感一干杯',
    icon: 'i-lucide-coffee',
    class: 'bg-lime-300 text-lime-900',
    style: { top: '70%', left: '22%' }
  },
  {
    label: 'LuminTreasure',
    class: 'bg-slate-800 text-white border border-white/20',
    style: { top: '70%', right: '14%' }
  },
  {
    label: 'AIGC',
    class: 'bg-white text-slate-900',
    style: { top: '80%', left: '20%' }
  },
  {
    label: '超快捷',
    icon: 'i-lucide-zap',
    class: 'bg-white text-slate-900 -rotate-6',
    style: { top: '80%', left: '38%' }
  },
  {
    label: '图片编辑',
    icon: 'i-lucide-layers',
    class: 'bg-violet-600 text-white',
    style: { top: '80%', right: '14%' }
  },
  {
    label: '极速生图体验',
    icon: 'i-lucide-zap',
    class: 'bg-violet-600 text-white writing-vertical',
    style: { top: '64%', left: '6%' }
  }
]
</script>

<template>
  <div class="relative hidden md:flex flex-col items-center w-[44%] shrink-0 overflow-hidden bg-gradient-to-br from-violet-900 via-purple-800 to-indigo-900 p-8 text-white">
    <!-- 网格背景纹理 -->
    <div
      class="absolute inset-0 opacity-20"
      style="background-image: linear-gradient(rgba(255,255,255,.15) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,.15) 1px, transparent 1px); background-size: 28px 28px;"
    />

    <!-- 顶部 Logo + 标语 -->
    <div class="relative z-10 flex flex-col items-center pt-8 text-center">
      <div class="flex items-center gap-2">
        <div class="flex items-center justify-center w-11 h-11 rounded-2xl bg-gradient-to-br from-sky-400 to-blue-600 shadow-lg">
          <UIcon
            name="i-lucide-sparkles"
            class="w-6 h-6"
          />
        </div>
        <span class="text-3xl font-bold tracking-tight">季星<span class="text-sky-300">Ai</span></span>
      </div>
      <p class="mt-3 text-xs font-medium tracking-widest text-white/70">
        AIGC 创意生产力平台
      </p>
      <h2 class="mt-5 text-2xl font-bold tracking-wide">
        登录一下 灵感开花
      </h2>
    </div>

    <!-- 散落胶囊标签 -->
    <div class="relative z-10 flex-1 w-full">
      <span
        v-for="tag in tags"
        :key="tag.label"
        class="absolute inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-semibold shadow-lg whitespace-nowrap"
        :class="tag.class"
        :style="tag.style"
      >
        <UIcon
          v-if="tag.icon"
          :name="tag.icon"
          class="w-3.5 h-3.5"
        />
        {{ tag.label }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.writing-vertical {
  writing-mode: vertical-rl;
  letter-spacing: 0.15em;
}
</style>
