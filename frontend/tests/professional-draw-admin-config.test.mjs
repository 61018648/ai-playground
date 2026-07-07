import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const admin = readFileSync(resolve(import.meta.dirname, '../app/pages/admin.vue'), 'utf8')

assert.match(admin, /\{ key: 'professional', label: '专业绘图'/)
assert.match(admin, /professionalDrawForm/)
assert.match(admin, /drawProviderId/)
assert.match(admin, /rewriteProviderId/)
assert.match(admin, /saveSetting\('professional_draw'/)
assert.match(admin, /生图接口/)
assert.match(admin, /润色接口/)
assert.doesNotMatch(admin, /section === 'professional' \|\| section === 'assistant'/)
assert.match(admin, /:key="section"/)

const professionalSection = admin.slice(
  admin.indexOf('section === \'professional\''),
  admin.indexOf('section === \'api\' || section === \'assistant\'')
)
assert.doesNotMatch(professionalSection, /<USelect/)
assert.match(professionalSection, /<select/)
