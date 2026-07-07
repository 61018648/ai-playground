import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const admin = readFileSync(resolve(import.meta.dirname, '../app/pages/admin.vue'), 'utf8')

const categoryOptions = admin.slice(
  admin.indexOf('const providerCategoryOptions = ['),
  admin.indexOf('const providerCategoryLabel')
)
assert.match(categoryOptions, /value: 'general'/)
assert.match(categoryOptions, /value: 'general_text'/)
assert.doesNotMatch(categoryOptions, /professional_drawing/)
assert.doesNotMatch(categoryOptions, /assistant_chat/)

assert.match(admin, /const drawProviderOptions = computed/)
assert.match(admin, /item\.category === 'general'/)
assert.match(admin, /const rewriteProviderOptions = computed/)
assert.match(admin, /item\.category === 'general_text'/)

const providerEditor = admin.slice(
  admin.indexOf(':title="providerForm.id ? \'编辑接口\' : \'新建接口\'"'),
  admin.indexOf('<UFormField label="接口类型">')
)
assert.match(providerEditor, /<select/)
assert.match(providerEditor, /v-model="providerForm.category"/)
assert.match(providerEditor, /v-for="option in providerCategoryOptions"/)
assert.doesNotMatch(providerEditor, /<USelect/)

assert.match(admin, /provider:\s*'openai'/)
assert.doesNotMatch(admin, /value: 'custom'/)
assert.doesNotMatch(admin, /value: 'placeholder'/)
