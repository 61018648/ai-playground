<script setup lang="ts">
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

const route = useRoute()
const api = useApi()
const { siteTitle, siteDescription, siteKeywords } = useSiteConfig()

useSeoMeta({
  title: siteTitle,
  description: siteDescription,
  ogTitle: siteTitle,
  ogDescription: siteDescription,
  keywords: siteKeywords,
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
