<script setup lang="ts">
import type { ApiSiteSetting } from '~/composables/useApi'

useHead({
  meta: [
    { name: 'viewport', content: 'width=device-width, initial-scale=1' }
  ],
  link: [
    { rel: 'icon', href: '/favicon.ico' }
  ],
  htmlAttrs: {
    lang: 'zh-CN'
  }
})

const api = useApi()
const route = useRoute()
const { data: publicSettings } = await useAsyncData('public-settings', () => api.get<ApiSiteSetting[]>('/public/settings'), {
  default: () => []
})
const seoSetting = computed(() => publicSettings.value.find(item => item.key === 'seo')?.value || {})
const title = computed(() => String(seoSetting.value.title || '摘星AI - AI 创作广场'))
const description = computed(() => String(seoSetting.value.description || '摘星AI 创作平台,提供 AI 绘画、电商视觉、文案创作等一站式智能创作工具。'))
const keywords = computed(() => String(seoSetting.value.keywords || 'AI生图,AI绘画,电商视觉,AI创作'))

useSeoMeta({
  title,
  description,
  ogTitle: title,
  ogDescription: description,
  keywords,
  twitterCard: 'summary_large_image'
})

onMounted(() => {
  const invite = route.query.invite || route.query.inviteCode
  if (typeof invite !== 'string' || !invite.trim()) return
  const code = invite.trim()
  const key = 'affiliate_visit_' + code
  if (sessionStorage.getItem(key)) return
  sessionStorage.setItem(key, '1')
  api.post('/affiliate/visit/' + encodeURIComponent(code)).catch(() => {})
})
</script>

<template>
  <UApp>
    <AppMessage />
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </UApp>
</template>
