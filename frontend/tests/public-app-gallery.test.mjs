import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const appCard = readFileSync(resolve(root, 'app/components/AppCard.vue'), 'utf8')
const appDetail = readFileSync(resolve(root, 'app/pages/apps/[id].vue'), 'utf8')
const styleGallery = readFileSync(resolve(root, 'app/components/StyleGallery.vue'), 'utf8')
const homeData = readFileSync(resolve(root, 'app/composables/useHomeData.ts'), 'utf8')

assert.ok(existsSync(resolve(root, 'app/pages/apps/[id].vue')), 'app detail page should exist')
assert.ok(existsSync(resolve(root, 'app/pages/apps/index.vue')), 'app list should live at app/pages/apps/index.vue so dynamic child routes work')
assert.equal(existsSync(resolve(root, 'app/pages/apps.vue')), false, 'top-level apps.vue should not shadow /apps/:id')

assert.match(appCard, /<NuxtLink/)
assert.match(appCard, /:to="appDetailTo"/)

assert.match(appDetail, /应用对话/)
assert.match(appDetail, /chatMessages/)
assert.match(appDetail, /messagesViewport/)
assert.match(appDetail, /api\.post<ApiGeneration>\('\/generations'/)
assert.match(appDetail, /api\.post<ApiAssistantChatResult>\('\/assistant\/chat'/)

assert.match(homeData, /GalleryWork/)
assert.match(homeData, /samplePrompt/)
assert.match(homeData, /targetAppCode/)
assert.match(homeData, /galleryWorks/)

assert.match(styleGallery, /画廊广场/)
assert.match(styleGallery, /一键同款/)
assert.match(styleGallery, /galleryWorks/)
assert.match(styleGallery, /targetAppCode/)
