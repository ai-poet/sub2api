import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppLayout.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AppLayout admin compliance gate', () => {
  it('does not mount page content while the compliance acknowledgement dialog is pending', () => {
    // 首次进入管理面时后端对所有管理接口返回 423；页面内容延后挂载，避免首屏请求风暴刷出成串错误 toast。
    expect(componentSource).toContain('<slot v-if="!complianceBlocking" />')
    expect(componentSource).toContain(
      'const complianceBlocking = computed(() => authStore.isAdmin && complianceStore.shouldShow)'
    )
  })
})
