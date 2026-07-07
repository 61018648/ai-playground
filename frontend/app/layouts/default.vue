<script setup lang="ts">
const { isOpen, close } = useSidebar()
const route = useRoute()
const isAssistantPage = computed(() => route.path === '/assistant')

// 路由变化时关闭抽屉(移动端)
watch(() => route.fullPath, () => close())
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-muted/30">
    <AppSidebar />

    <!-- 移动端遮罩 -->
    <Transition
      enter-active-class="transition-opacity duration-300"
      leave-active-class="transition-opacity duration-300"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div
        v-if="isOpen"
        class="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm lg:hidden"
        @click="close"
      />
    </Transition>

    <div class="flex flex-col flex-1 min-w-0">
      <HomeHeader />

      <main
        class="flex-1 min-h-0"
        :class="isAssistantPage ? 'overflow-hidden' : 'overflow-y-auto'"
      >
        <div :class="isAssistantPage ? 'h-full min-h-0' : 'min-h-full px-4 sm:px-6 py-6'">
          <slot />
        </div>
      </main>
    </div>
  </div>
</template>
