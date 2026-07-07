import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const component = readFileSync(
  resolve(import.meta.dirname, '../app/components/DrawComposer.vue'),
  'utf8'
)

assert.match(component, /aria-label="润色\/改写"/)
assert.match(component, /icon="i-lucide-wand-sparkles"/)
assert.match(component, /title="润色\/改写提示词"/)
assert.match(component, /w-10 h-10 sm:w-auto sm:px-4/)
