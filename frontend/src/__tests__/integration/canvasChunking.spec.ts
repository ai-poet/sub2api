// @vitest-environment node
import { describe, it, expect } from 'vitest'
import { loadConfigFromFile } from 'vite'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// Regression for the production TDZ "Cannot access 'Ge' before initialization":
// main entry statically imports canvas modules that manualChunks left in vendor-misc,
// creating an entry <-> feature chunk cycle. Every canvas-exclusive module must be
// pinned to feature-canvas-studio.
const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '../../..')

const CANVAS_EXCLUSIVE = [
  'src/components/canvas/CanvasEditor.vue',
  'src/components/canvas/CanvasToolbar.vue',
  'src/views/user/CanvasStudioView.vue',
  'src/composables/useCanvasStudio.ts',
  'src/api/canvas.ts',
  'src/utils/canvasGeometry.ts',
  'src/utils/canvasScene.ts',
  'src/utils/canvasMask.ts',
  'src/utils/canvasComposite.ts',
  'src/utils/canvasLocalStore.ts',
]
const NOT_CANVAS = [
  'src/main.ts',
  'src/api/auth.ts',
  'src/api/client.ts',
  'src/views/HomeView.vue',
]

async function getManualChunks() {
  const loaded = await loadConfigFromFile(
    { command: 'build', mode: 'production' },
    path.join(root, 'vite.config.ts'),
    root,
  )
  if (!loaded) throw new Error('vite.config.ts failed to load')
  const mc = loaded.config.build?.rollupOptions?.output
  const chunker = Array.isArray(mc) ? mc[0]?.manualChunks : mc?.manualChunks
  if (typeof chunker !== 'function') throw new Error('manualChunks callback not found')
  return chunker as (id: string) => string | undefined
}

describe('canvas chunk pinning', () => {
  it('pins every canvas-exclusive module to feature-canvas-studio (posix + windows paths)', async () => {
    const chunker = await getManualChunks()
    for (const rel of CANVAS_EXCLUSIVE) {
      const posix = path.join(root, rel)
      expect(chunker(posix), rel).toBe('feature-canvas-studio')
      expect(chunker(posix.split('/').join('\\')), `${rel} (windows)`).toBe('feature-canvas-studio')
    }
  })
  it('pins fabric to feature-canvas-studio', async () => {
    const chunker = await getManualChunks()
    expect(chunker('/x/node_modules/fabric/dist/fabric.mjs')).toBe('feature-canvas-studio')
  })
  it('does not pin shared entry modules to the canvas chunk', async () => {
    const chunker = await getManualChunks()
    for (const rel of NOT_CANVAS) {
      expect(chunker(path.join(root, rel)), rel).not.toBe('feature-canvas-studio')
    }
  })
})
