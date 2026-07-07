<script setup lang="ts">
import type { DrawCategory } from '~/composables/useDrawData'

defineProps<{ category: DrawCategory, selected?: boolean }>()
defineEmits<{ select: [] }>()
</script>

<template>
  <button
    type="button"
    class="group flex items-center gap-3 w-full p-4 rounded-2xl border text-left transition-all"
    :class="selected
      ? 'border-primary bg-primary/5 ring-1 ring-primary shadow-sm'
      : 'border-default bg-default hover:border-primary/40 hover:shadow-sm'"
    @click="$emit('select')"
  >
    <span
      class="flex items-center justify-center w-10 h-10 rounded-xl text-white shrink-0"
      :class="category.color"
    >
      <UIcon
        :name="category.icon"
        class="w-5 h-5"
      />
    </span>
    <span class="min-w-0 flex-1">
      <span class="block text-sm font-semibold text-highlighted truncate">
        {{ category.name }}
      </span>
      <span class="block text-xs text-dimmed truncate">
        {{ category.description }}
      </span>
    </span>
    <UIcon
      v-if="selected"
      name="i-lucide-check-circle-2"
      class="w-4 h-4 text-primary shrink-0"
    />
  </button>
</template>
