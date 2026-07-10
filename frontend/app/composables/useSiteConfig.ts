import type { ApiSiteSetting } from '~/composables/useApi'

const DEFAULT_SITE_NAME = 'AI Playground'
const DEFAULT_SITE_SUFFIX = 'AI 创作广场'
const DEFAULT_DESCRIPTION = 'AI 创作平台，提供 AI 绘画、电商视觉、文案创作等一站式智能创作工具。'
const DEFAULT_KEYWORDS = 'AI生图,AI绘画,电商视觉,AI创作'

const settingText = (value: unknown) => typeof value === 'string' ? value.trim() : ''

export const useSiteConfig = () => {
  const api = useApi()
  const { data: publicSettings, refresh } = useAsyncData<ApiSiteSetting[]>(
    'public-settings',
    () => api.get<ApiSiteSetting[]>('/public/settings'),
    { default: () => [] }
  )

  const seoSetting = computed(() => publicSettings.value.find(item => item.key === 'seo')?.value || {})
  const siteName = computed(() => {
    const configuredName = settingText(seoSetting.value.siteName)
    if (configuredName) return configuredName

    const configuredTitle = settingText(seoSetting.value.title)
    if (configuredTitle.includes(' - ')) return configuredTitle.split(' - ')[0]?.trim() || DEFAULT_SITE_NAME
    return configuredTitle || DEFAULT_SITE_NAME
  })
  const siteTitle = computed(() => settingText(seoSetting.value.title) || `${siteName.value} - ${DEFAULT_SITE_SUFFIX}`)
  const siteDescription = computed(() => settingText(seoSetting.value.description) || DEFAULT_DESCRIPTION)
  const siteKeywords = computed(() => settingText(seoSetting.value.keywords) || DEFAULT_KEYWORDS)
  const pageTitle = (pageName: string) => {
    const name = pageName.trim()
    return name ? `${name} - ${siteName.value}` : siteTitle.value
  }

  return {
    publicSettings,
    refresh,
    seoSetting,
    siteName,
    siteTitle,
    siteDescription,
    siteKeywords,
    pageTitle
  }
}
