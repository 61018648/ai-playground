import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const composer = readFileSync(resolve(import.meta.dirname, '../app/components/DrawComposer.vue'), 'utf8')

assert.match(composer, /rewriting\s*=\s*ref\(false\)/)
assert.match(composer, /api\.post<\{ prompt: string \}>\('\/generations\/professional-draw\/rewrite'/)
assert.match(composer, /prompt\.value\s*=\s*result\.prompt/)
assert.match(composer, /:loading="rewriting"/)
