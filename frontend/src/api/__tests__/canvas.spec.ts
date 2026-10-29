import { afterEach, describe, expect, it, vi } from 'vitest'
import { extractOutputImages, fetchOutputImage, submitCanvasTask, type CanvasTask } from '../canvas'

function request(overrides: Partial<Parameters<typeof submitCanvasTask>[1]> = {}) {
  return {
    image: null,
    mask: null,
    prompt: 'a small boat',
    model: 'gpt-image-1',
    size: '1024x1024',
    n: 1,
    ...overrides,
  }
}

function acceptedResponse() {
  return {
    ok: true,
    json: async () => ({ id: 'imgtask_1', task_id: 'imgtask_1', status: 'processing' }),
  } as Response
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('canvas api', () => {
  it('uses generations for an empty canvas', async () => {
    const fetchMock = vi.fn().mockResolvedValue(acceptedResponse())
    vi.stubGlobal('fetch', fetchMock)

    await submitCanvasTask('sk-test', request())

    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('http://localhost:3000/v1/images/generations/async')
    expect(options.headers).toMatchObject({ Authorization: 'Bearer sk-test' })
    const form = options.body as FormData
    expect(form.get('image')).toBeNull()
    expect(form.get('mask')).toBeNull()
  })

  it('uploads the composite and mask through edits', async () => {
    const fetchMock = vi.fn().mockResolvedValue(acceptedResponse())
    vi.stubGlobal('fetch', fetchMock)
    const image = new Blob(['image'], { type: 'image/png' })
    const mask = new Blob(['mask'], { type: 'image/png' })

    await submitCanvasTask('sk-test', request({ image, mask }))

    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('http://localhost:3000/v1/images/edits/async')
    const form = options.body as FormData
    expect(form.get('image')).toBeInstanceOf(File)
    expect(form.get('mask')).toBeInstanceOf(File)
    expect(form.get('model')).toBe('gpt-image-1')
  })

  it('extracts URL and base64 outputs and decodes base64 without another request', async () => {
    const task = {
      result: {
        data: [{ url: 'https://cdn.example.com/1.png' }, { b64_json: 'aGVsbG8=' }, {}],
      },
    } as CanvasTask
    const images = extractOutputImages(task)
    expect(images).toEqual([
      { url: 'https://cdn.example.com/1.png', b64: undefined },
      { url: undefined, b64: 'aGVsbG8=' },
    ])

    const blob = await fetchOutputImage(images[1])
    expect(blob?.size).toBe(5)
    expect(blob?.type).toBe('image/png')
  })
})
