<script setup lang="ts">
const { siteName } = useSiteConfig()
const logoMain = computed(() => siteName.value.replace(/AI$/i, '').trim() || siteName.value)
const logoSuffix = computed(() => siteName.value.match(/AI$/i)?.[0] || '')
</script>

<template>
  <div class="flex items-center gap-2">
    <div class="flex items-center justify-center w-9 h-9 rounded-xl bg-gradient-to-br from-blue-500 to-indigo-500 text-white shadow-sm">
      <UIcon
        name="i-lucide-sparkles"
        class="w-5 h-5"
      />
    </div>
    <span class="text-xl font-bold tracking-tight text-highlighted">
      {{ logoMain }}
      <span
        v-if="logoSuffix"
        class="text-primary"
      >{{ logoSuffix }}</span>
    </span>
  </div>
</template>
