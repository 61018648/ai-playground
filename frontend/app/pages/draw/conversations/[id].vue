<script setup lang="ts">
const route = useRoute()

await navigateTo('/draw-chat/' + String(route.params.id || ''), { replace: true })
</script>

<template>
  <div />
</template>
