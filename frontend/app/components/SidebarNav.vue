<script setup lang="ts">
import type { NavItem } from '~/composables/useHomeData'

defineProps<{ items: NavItem[] }>()
defineEmits<{ navigate: [] }>()
</script>

<template>
  <nav class="flex flex-col gap-1">
    <NuxtLink
      v-for="item in items"
      :key="item.to"
      :to="item.to"
      class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors text-left text-muted hover:bg-elevated hover:text-highlighted"
      exact-active-class="!bg-primary/10 !text-primary"
      @click="$emit('navigate')"
    >
      <UIcon
        :name="item.icon"
        class="w-5 h-5 shrink-0"
      />
      <span>{{ item.label }}</span>
    </NuxtLink>
  </nav>
</template>
