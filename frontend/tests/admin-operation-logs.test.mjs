import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const admin = readFileSync(resolve(import.meta.dirname, '../app/pages/admin.vue'), 'utf8')

assert.match(admin, /const taskLogChannel = \(log: ApiTaskLog\) =>/)
assert.match(admin, /meta\.providerName/)
assert.match(admin, /meta\.model/)

const taskLogs = admin.slice(
  admin.indexOf('<template v-else>', admin.indexOf("logTab === 'login'")),
  admin.indexOf('</template>', admin.indexOf('<template v-else>', admin.indexOf("logTab === 'login'")))
)
assert.match(taskLogs, /<span>渠道<\/span>/)
assert.match(taskLogs, /taskLogChannel\(log\)/)
