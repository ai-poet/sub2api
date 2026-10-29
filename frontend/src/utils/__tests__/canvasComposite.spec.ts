import { afterEach, describe, expect, it, vi } from 'vitest'
import { buildMaskBlob, fitEdge } from '../canvasComposite'

class TestImageData {
  data: Uint8ClampedArray

  constructor(width: number, height: number) {
    this.data = new Uint8ClampedArray(width * height * 4)
  }
}

function maskCanvas(width: number, height: number, pixels: number[]): HTMLCanvasElement {
  return {
    width,
    height,
    getContext: () => ({
      getImageData: () => ({ data: new Uint8ClampedArray(pixels) }),
    }),
  } as unknown as HTMLCanvasElement
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('canvasComposite', () => {
  it('fitEdge keeps small images and scales large ones proportionally', () => {
    expect(fitEdge(1024, 1536, 2048)).toEqual({ width: 1024, height: 1536 })
    expect(fitEdge(4096, 2048, 2048)).toEqual({ width: 2048, height: 1024 })
  })

  it('omits a completely empty mask', async () => {
    const createElement = vi.spyOn(document, 'createElement')
    await expect(buildMaskBlob(maskCanvas(1, 2, new Array(8).fill(0)))).resolves.toBeNull()
    expect(createElement).not.toHaveBeenCalled()
  })

  it('turns painted pixels transparent and preserves unpainted pixels as opaque', async () => {
    vi.stubGlobal('ImageData', TestImageData)
    let output: TestImageData | null = null
    const exported = {
      width: 0,
      height: 0,
      getContext: () => ({
        putImageData: (value: TestImageData) => {
          output = value
        },
      }),
      toBlob: (callback: (blob: Blob | null) => void) => {
        callback(new Blob(['mask'], { type: 'image/png' }))
      },
    }
    const originalCreateElement = document.createElement.bind(document)
    vi.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
      if (tagName === 'canvas') return exported as unknown as HTMLCanvasElement
      return originalCreateElement(tagName)
    })

    // 两个像素：第一个未涂抹，第二个有红色笔迹。
    const pixels = [0, 0, 0, 0, 239, 68, 68, 180]
    const blob = await buildMaskBlob(maskCanvas(2, 1, pixels))

    expect(blob?.type).toBe('image/png')
    expect(output).not.toBeNull()
    expect(Array.from(output!.data)).toEqual([
      255, 255, 255, 255,
      255, 255, 255, 0,
    ])
  })
})
