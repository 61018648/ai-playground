<script setup lang="ts">
const open = defineModel<boolean>('open', { default: false })
</script>

<template>
  <UModal
    v-model:open="open"
    :ui="{
      overlay: 'bg-default/30 backdrop-blur-md',
      content: 'max-w-3xl w-full sm:max-w-3xl max-h-[90dvh] overflow-y-auto rounded-2xl'
    }"
  >
    <template #content="{ close }">
      <div class="relative flex md:min-h-[34rem]">
        <!-- 左侧宣传面板 -->
        <AuthPromoPanel />

        <!-- 右侧表单 -->
        <AuthForm @success="close" />

        <!-- 关闭按钮 -->
        <UButton
          icon="i-lucide-x"
          color="neutral"
          variant="ghost"
          size="sm"
          class="absolute top-4 right-4 rounded-full z-20"
          aria-label="关闭"
          @click="close"
        />
      </div>
    </template>
  </UModal>
</template>
