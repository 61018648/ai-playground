import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const files = [
  '../app/app.vue',
  '../app/components/AppLogo.vue',
  '../app/components/HomeHeader.vue',
  '../app/pages/admin.vue',
  '../app/pages/assistant.vue',
  '../app/pages/apps/index.vue',
  '../app/pages/apps/[id].vue',
  '../app/pages/draw.vue',
  '../app/pages/draw-chat/[id].vue',
  '../app/pages/history.vue',
  '../app/pages/media.vue',
  '../app/pages/profile.vue',
  '../app/pages/tools.vue'
]

for (const file of files) {
  const content = readFileSync(resolve(import.meta.dirname, file), 'utf8')
  assert.doesNotMatch(content, /摘星AI|季星AI/, `${file} should read the site name from backend settings`)
}

const siteConfig = readFileSync(resolve(import.meta.dirname, '../app/composables/useSiteConfig.ts'), 'utf8')
assert.match(siteConfig, /siteName/)
assert.match(siteConfig, /pageTitle/)
