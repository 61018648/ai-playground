<script setup lang="ts">
const { navItems } = useHomeData()
const { isOpen, close } = useSidebar()
</script>

<template>
  <aside
    class="fixed lg:static inset-y-0 left-0 z-50 flex flex-col w-64 shrink-0 h-screen border-r border-default bg-default px-3 py-4 transition-transform duration-300 ease-out lg:translate-x-0"
    :class="isOpen ? 'translate-x-0 shadow-xl' : '-translate-x-full'"
  >
    <!-- 顶部 Logo + 移动端关闭按钮 -->
    <div class="flex items-center justify-between px-1 pb-4">
      <AppLogo />
      <UButton
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="sm"
        class="lg:hidden"
        aria-label="关闭菜单"
        @click="close"
      />
    </div>

    <!-- 主导航 -->
    <SidebarNav
      :items="navItems"
      @navigate="close"
    />

    <!-- 对话面板 -->
    <div class="flex flex-col flex-1 min-h-0 mt-5">
      <SidebarConversations />
    </div>

    <!-- 底部按钮 -->
    <SidebarFooter class="mt-3" />
  </aside>
</template>
