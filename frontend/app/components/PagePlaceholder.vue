<script setup lang="ts">
defineProps<{
  title: string
  description?: string
  icon?: string
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center min-h-[calc(100vh-7rem)] gap-4 text-center">
    <div class="flex items-center justify-center w-16 h-16 rounded-2xl bg-primary/10 text-primary">
      <UIcon
        :name="icon ?? 'i-lucide-sparkles'"
        class="w-8 h-8"
      />
    </div>
    <div class="space-y-1">
      <h1 class="text-2xl font-bold text-highlighted">
        {{ title }}
      </h1>
      <p
        v-if="description"
        class="text-sm text-dimmed max-w-md"
      >
        {{ description }}
      </p>
    </div>
    <UButton
      to="/"
      color="neutral"
      variant="soft"
      icon="i-lucide-arrow-left"
    >
      返回画廊广场
    </UButton>
  </div>
</template>
