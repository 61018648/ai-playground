<script setup lang="ts">
const { pageTitle } = useSiteConfig()

useHead(() => ({ title: pageTitle('更多工具') }))
</script>

<template>
  <PagePlaceholder
    title="更多工具"
    description="发现更多实用的 AI 创作工具。"
    icon="i-lucide-wrench"
  />
</template>
