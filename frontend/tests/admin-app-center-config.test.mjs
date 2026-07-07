import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const admin = readFileSync(resolve(import.meta.dirname, '../app/pages/admin.vue'), 'utf8')
const api = readFileSync(resolve(import.meta.dirname, '../app/composables/useApi.ts'), 'utf8')

assert.match(api, /appType: 'image' \| 'text'/)

assert.match(admin, /const appTypeOptions = \[/)
assert.match(admin, /value: 'image'/)
assert.match(admin, /value: 'text'/)
assert.match(admin, /appType: 'image'/)
assert.match(admin, /appProviderOptionsByType/)
assert.match(admin, /provider\.category === expectedProviderCategory/)
assert.match(admin, /appTypeLabel\(app\.appType/)
assert.match(admin, /selectValue\(appForm\.providerId\)/)

const appEditor = admin.slice(
  admin.indexOf(':title="appForm.id ? \'编辑应用\' : \'新建应用\'"'),
  admin.indexOf('<UFormField label="接口配置"')
)
assert.match(appEditor, /<UFormField label="应用类型">/)
assert.match(appEditor, /v-model="appForm.appType"/)

const savePayload = admin.slice(
  admin.indexOf('const payload = {'),
  admin.indexOf('if \\(appForm.id\\)')
)
assert.match(savePayload, /appType: selectValue\(appForm\.appType\)/)
