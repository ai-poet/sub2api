import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  CanvasInputError,
  extractTaskOutputs,
  fetchTaskOutputBlob,
  getCanvasTask,
  isSafeOutputUrl,
  submitCanvasTask,
  type CanvasSubmitRequest,
  type CanvasTask,
} from '../canvas'

function request(overrides: Partial<CanvasSubmitRequest> = {}): CanvasSubmitRequest {
  return {
    operation: 'generate',
    images: [],
    mask: null,
    prompt: 'a small boat',
    model: 'gpt-image-1',
    size: '1024x1024',
    count: 1,
    ...overrides,
  }
}

function acceptedResponse() {
  return {
    ok: true,
    json: async () => ({ id: 'imgtask_1', task_id: 'imgtask_1', status: 'processing' }),
  } as Response
}

function image(label = 'image'): Blob {
  return new Blob([label], { type: 'image/png' })
}

const PNG_HEAD = new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0])
const JPEG_HEAD = new Uint8Array([0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0, 0, 0])
const WEBP_HEAD = new Uint8Array([
  0x52, 0x49, 0x46, 0x46, 0x24, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50,
])

function toB64(bytes: Uint8Array): string {
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary)
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('canvas submit', () => {
  it('posts JSON to generations for text-to-image', async () => {
    const fetchMock = vi.fn().mockResolvedValue(acceptedResponse())
    vi.stubGlobal('fetch', fetchMock)
    const controller = new AbortController()

    await submitCanvasTask('sk-test', request({ count: 2, quality: 'high' }), controller.signal)

    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('http://localhost:3000/v1/images/generations/async')
    expect(options.headers).toMatchObject({
      Authorization: 'Bearer sk-test',
      'Content-Type': 'application/json',
    })
    expect(options.signal).toBe(controller.signal)
    expect(options.body).not.toBeInstanceOf(FormData)
    expect(JSON.parse(options.body as string)).toEqual({
      model: 'gpt-image-1',
      prompt: 'a small boat',
      size: '1024x1024',
      n: 2,
      quality: 'high',
    })
  })

  it('rejects generate requests that carry sources or a mask', async () => {
    const fetchMock = vi.fn().mockResolvedValue(acceptedResponse())
    vi.stubGlobal('fetch', fetchMock)

    await expect(submitCanvasTask('sk-test', request({ images: [image()] }))).rejects.toBeInstanceOf(
      CanvasInputError,
    )
    await expect(
      submitCanvasTask('sk-test', request({ mask: image('mask') })),
    ).rejects.toBeInstanceOf(CanvasInputError)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('rejects blank model or prompt and invalid counts before any request', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    await expect(submitCanvasTask('sk-test', request({ model: '  ' }))).rejects.toBeInstanceOf(
      CanvasInputError,
    )
    await expect(submitCanvasTask('sk-test', request({ prompt: '' }))).rejects.toBeInstanceOf(
      CanvasInputError,
    )
    await expect(submitCanvasTask('sk-test', request({ prompt: '   ' }))).rejects.toBeInstanceOf(
      CanvasInputError,
    )
    await expect(submitCanvasTask('sk-test', request({ count: 0 }))).rejects.toBeInstanceOf(
      CanvasInputError,
    )
    await expect(submitCanvasTask('sk-test', request({ count: 1.5 }))).rejects.toBeInstanceOf(
      CanvasInputError,
    )
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('rejects empty source blobs and an edit that carries a mask', async () => {
    await expect(
      submitCanvasTask(
        'sk-test',
        request({ operation: 'edit', images: [image(), new Blob([])] }),
      ),
    ).rejects.toBeInstanceOf(CanvasInputError)
    await expect(
      submitCanvasTask(
        'sk-test',
        request({ operation: 'edit', images: [image()], mask: image('mask') }),
      ),
    ).rejects.toBeInstanceOf(CanvasInputError)
  })

  it('rejects more than 16 edit sources', async () => {
    await expect(
      submitCanvasTask(
        'sk-test',
        request({
          operation: 'edit',
          images: Array.from({ length: 17 }, () => image()),
        }),
      ),
    ).rejects.toBeInstanceOf(CanvasInputError)
  })

  it('posts a single source under "image" for edit', async () => {
    const fetchMock = vi.fn().mockResolvedValue(acceptedResponse())
    vi.stubGlobal('fetch', fetchMock)

    await submitCanvasTask(
      'sk-test',
      request({ operation: 'edit', images: [image('one')] }),
    )

    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('http://localhost:3000/v1/images/edits/async')
    const form = options.body as FormData
    expect(form.get('image')).toBeInstanceOf(File)
    expect(form.getAll('image[]')).toHaveLength(0)
    expect(form.get('mask')).toBeNull()
    expect(form.get('model')).toBe('gpt-image-1')
  })

  it('posts multiple sources under "image[]" for edit', async () => {
    const fetchMock = vi.fn().mockResolvedValue(acceptedResponse())
    vi.stubGlobal('fetch', fetchMock)

    await submitCanvasTask(
      'sk-test',
      request({ operation: 'edit', images: [image('one'), image('two'), image('three')] }),
    )

    const [, options] = fetchMock.mock.calls[0] as [string, RequestInit]
    const form = options.body as FormData
    expect(form.get('image')).toBeNull()
    expect(form.getAll('image[]')).toHaveLength(3)
  })

  it('names uploads with the extension matching their real bytes', async () => {
    const fetchMock = vi.fn().mockResolvedValue(acceptedResponse())
    vi.stubGlobal('fetch', fetchMock)

    await submitCanvasTask(
      'sk-test',
      request({
        operation: 'edit',
        images: [
          new Blob([JPEG_HEAD], { type: 'image/jpeg' }),
          // Undeclared type: extension falls back to magic-byte sniffing.
          new Blob([WEBP_HEAD]),
          new Blob([PNG_HEAD], { type: 'image/png' }),
        ],
      }),
    )

    const [, options] = fetchMock.mock.calls[0] as [string, RequestInit]
    const files = (options.body as FormData).getAll('image[]') as File[]
    expect(files.map((file) => file.name)).toEqual([
      'canvas-0.jpg',
      'canvas-1.webp',
      'canvas-2.png',
    ])
  })

  it('rejects edit without sources', async () => {
    await expect(
      submitCanvasTask('sk-test', request({ operation: 'edit', images: [] })),
    ).rejects.toBeInstanceOf(CanvasInputError)
  })

  it('posts image and mask for inpaint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(acceptedResponse())
    vi.stubGlobal('fetch', fetchMock)

    await submitCanvasTask(
      'sk-test',
      request({ operation: 'inpaint', images: [image()], mask: image('mask') }),
    )

    const [, options] = fetchMock.mock.calls[0] as [string, RequestInit]
    const form = options.body as FormData
    expect(form.get('image')).toBeInstanceOf(File)
    expect(form.get('mask')).toBeInstanceOf(File)
    expect(form.getAll('image[]')).toHaveLength(0)
  })

  it('rejects inpaint without exactly one image and a non-empty mask', async () => {
    const mask = image('mask')
    await expect(
      submitCanvasTask('sk-test', request({ operation: 'inpaint', images: [image()], mask: null })),
    ).rejects.toBeInstanceOf(CanvasInputError)
    await expect(
      submitCanvasTask(
        'sk-test',
        request({ operation: 'inpaint', images: [image(), image('two')], mask }),
      ),
    ).rejects.toBeInstanceOf(CanvasInputError)
    await expect(
      submitCanvasTask(
        'sk-test',
        request({ operation: 'inpaint', images: [image()], mask: new Blob([]) }),
      ),
    ).rejects.toBeInstanceOf(CanvasInputError)
  })

  it('surfaces the gateway error body and never retries a submission', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 402,
      statusText: 'Payment Required',
      json: async () => ({ error: { message: 'quota exceeded', code: 'insufficient_quota' } }),
    } as Response)
    vi.stubGlobal('fetch', fetchMock)

    const failure = await submitCanvasTask('sk-test', request()).catch((error: Error) => error)
    expect(failure.message).toBe('quota exceeded')
    expect((failure as { status?: number }).status).toBe(402)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('passes the abort signal through to polling', async () => {
    let captured: AbortSignal | undefined
    const fetchMock = vi.fn().mockImplementation((_url: string, options?: RequestInit) => {
      captured = options?.signal as AbortSignal
      // Mirrors real fetch: the request rejects when the signal aborts.
      return new Promise((_resolve, reject) => {
        captured?.addEventListener('abort', () =>
          reject(new DOMException('aborted', 'AbortError')),
        )
      })
    })
    vi.stubGlobal('fetch', fetchMock)
    const controller = new AbortController()

    const pending = getCanvasTask('sk-test', 'imgtask_9', controller.signal)
    pending.catch(() => {})

    const [url] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('http://localhost:3000/v1/images/tasks/imgtask_9')
    // The caller's signal drives a linked internal signal, never passed raw.
    expect(captured).toBeInstanceOf(AbortSignal)
    expect(captured?.aborted).toBe(false)
    controller.abort()
    expect(captured?.aborted).toBe(true)
    await expect(pending).rejects.toThrow()
  })
})

describe('canvas outputs', () => {
  it('keeps original indexes when earlier entries are malformed', () => {
    const task = {
      result: {
        data: [
          {},
          { url: 'https://cdn.example.com/1.png' },
          { b64_json: 'aGVsbG8=' },
        ],
      },
    } as CanvasTask

    expect(extractTaskOutputs(task)).toEqual([
      { index: 0, url: undefined, b64: undefined },
      { index: 1, url: 'https://cdn.example.com/1.png', b64: undefined },
      { index: 2, url: undefined, b64: 'aGVsbG8=' },
    ])
  })

  it('decodes base64 without a network call', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    const blob = await fetchTaskOutputBlob({ b64: 'aGVsbG8=' })
    expect(blob?.size).toBe(5)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('falls back to a safe URL when base64 decoding fails', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      blob: async () => new Blob(['remote']),
    } as Response)
    vi.stubGlobal('fetch', fetchMock)

    const blob = await fetchTaskOutputBlob({
      b64: '%%%not-base64%%%',
      url: 'https://cdn.example.com/1.png',
    })
    expect(blob?.size).toBe(6)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('sniffs the real MIME type from decoded base64 bytes', async () => {
    const webp = await fetchTaskOutputBlob({ b64: toB64(WEBP_HEAD) })
    expect(webp?.type).toBe('image/webp')
    const jpeg = await fetchTaskOutputBlob({ b64: toB64(JPEG_HEAD) })
    expect(jpeg?.type).toBe('image/jpeg')
  })

  it('refuses unsafe or credentialed output URLs and preserves fetch failures as null', async () => {
    const fetchMock = vi.fn().mockRejectedValue(new Error('CORS'))
    vi.stubGlobal('fetch', fetchMock)

    expect(isSafeOutputUrl('javascript:alert(1)')).toBe(false)
    expect(isSafeOutputUrl('data:image/png;base64,AA==')).toBe(false)
    expect(isSafeOutputUrl('https://user:pass@cdn.example.com/1.png')).toBe(false)
    expect(isSafeOutputUrl('https://cdn.example.com/1.png')).toBe(true)

    expect(await fetchTaskOutputBlob({ url: 'javascript:alert(1)' })).toBeNull()
    expect(await fetchTaskOutputBlob({ url: 'https://user:pass@cdn.example.com/1.png' })).toBeNull()
    expect(fetchMock).not.toHaveBeenCalled()

    expect(await fetchTaskOutputBlob({ url: 'https://cdn.example.com/1.png' })).toBeNull()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('propagates a caller abort instead of reporting a fetch failure', async () => {
    const fetchMock = vi.fn().mockImplementation((_url: string, options?: RequestInit) =>
      new Promise((_resolve, reject) => {
        options?.signal?.addEventListener('abort', () =>
          reject(new DOMException('aborted', 'AbortError')),
        )
      }),
    )
    vi.stubGlobal('fetch', fetchMock)
    const controller = new AbortController()

    const pending = fetchTaskOutputBlob(
      { url: 'https://cdn.example.com/1.png' },
      controller.signal,
    )
    controller.abort()
    await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
  })
})

describe('canvas outputs via gateway proxy', () => {
  it('fetches HTTP output through the same-origin content endpoint with the original key', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      blob: async () => new Blob(['proxied'], { type: 'image/png' }),
    } as Response)
    vi.stubGlobal('fetch', fetchMock)

    const blob = await fetchTaskOutputBlob(
      { url: 'http://cdn.example.com/a.png?sig=x' },
      undefined,
      { apiKey: 'sk-orig', taskId: 'task_9', index: 2 },
    )
    expect(blob?.size).toBe(7)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toContain('/v1/images/tasks/task_9/outputs/2/content')
    expect(url).not.toContain('cdn.example.com')
    expect((options.headers as Record<string, string>).Authorization).toBe('Bearer sk-orig')
    expect((options.headers as Record<string, string>).Accept).toBe('image/*')
    expect(url).not.toContain('sk-orig')
  })

  it('never falls back to the external URL when the gateway content request fails', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: false, status: 502 } as Response)
    vi.stubGlobal('fetch', fetchMock)

    const blob = await fetchTaskOutputBlob(
      { url: 'http://cdn.example.com/a.png' },
      undefined,
      { apiKey: 'sk-orig', taskId: 'task_9', index: 0 },
    )
    expect(blob).toBeNull()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock.mock.calls[0][0]).toContain('/v1/images/tasks/')
  })

  it('rejects malformed source locally without any request', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    for (const source of [
      { apiKey: '', taskId: 't', index: 0 },
      { apiKey: 'k', taskId: ' ', index: 0 },
      { apiKey: 'k', taskId: 't', index: -1 },
      { apiKey: 'k', taskId: 't', index: 1.5 },
    ]) {
      expect(await fetchTaskOutputBlob({ url: 'http://a/x.png' }, undefined, source)).toBeNull()
    }
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
