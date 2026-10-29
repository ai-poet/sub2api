/**
 * canvasApi.ts 提交画布生成任务并轮询结果。
 *
 * 画布不新增后端接口：合成好的图片走既有的 /v1/images/edits/async，
 * 有源图时是图生图，没有源图时退回 /v1/images/generations/async。
 * 鉴权用用户自己的 API Key，和批量生图页的既有做法一致。
 */

import { buildGatewayUrl } from './client'

/** 异步任务的状态。 */
export type CanvasTaskStatus = 'processing' | 'completed' | 'failed' | string

/**
 * CanvasSubmitResult 是提交后拿到的任务句柄。
 */
export interface CanvasSubmitResult {
  /** 任务 ID，用于轮询。 */
  id: string
  /** 任务对象名。 */
  task_id: string
  /** 任务状态。 */
  status: CanvasTaskStatus
  /** 创建时间（秒时间戳）。 */
  created_at: number
  /** 过期时间（秒时间戳）。 */
  expires_at: number
  /** 轮询地址。 */
  poll_url?: string
}

/**
 * CanvasTask 是轮询得到的任务状态。
 */
export interface CanvasTask {
  id: string
  task_id: string
  status: CanvasTaskStatus
  http_status?: number
  /** 成功时的上游响应体（OpenAI images 格式）。 */
  result?: { data?: Array<{ url?: string; b64_json?: string }> } | null
  /** 失败时的错误体。 */
  error?: { code?: string; message?: string; type?: string } | null
  created_at: number
  completed_at?: number | null
  expires_at: number
}

/**
 * CanvasGenerateRequest 描述一次画布提交。
 */
export interface CanvasGenerateRequest {
  /** 合成后的图片字节。有源图时走 edits，没有时走 generations。 */
  image: Blob | null
  /** 遮罩字节。没有笔刷痕迹时为 null。 */
  mask: Blob | null
  /** 提示词。 */
  prompt: string
  /** 模型名。 */
  model: string
  /** 尺寸，如 1024x1024。 */
  size: string
  /** 生成张数。 */
  n?: number
  /** 质量档位。 */
  quality?: string
  /** 背景模式。 */
  background?: string
}

/**
 * parseCanvasError 把网关的错误响应翻译成 Error。
 * 错误体优先取 error.message，其次 message，最后状态码文案。
 */
async function parseCanvasError(response: Response): Promise<Error> {
  let message = response.statusText || `HTTP ${response.status}`
  let code: string | number = response.status
  try {
    const body = await response.json()
    const err = body?.error
    if (typeof err?.message === 'string' && err.message) {
      message = err.message
    } else if (typeof body?.message === 'string' && body.message) {
      message = body.message
    }
    if (err?.code) {
      code = err.code
    }
  } catch {
    // 响应体不是 JSON，保留状态码文案。
  }
  const error = new Error(message)
  ;(error as { code?: string | number }).code = code
  ;(error as { status?: number }).status = response.status
  return error
}

/**
 * authHeaders 构造带 API Key 的请求头。
 */
function authHeaders(apiKey: string, extra?: HeadersInit): HeadersInit {
  return {
    Authorization: `Bearer ${apiKey}`,
    ...extra,
  }
}

/**
 * submitCanvasTask 提交一次画布生成任务。
 *
 * 有源图时走 /v1/images/edits/async（携带 image 与可选 mask），
 * 无源图时走 /v1/images/generations/async（纯文生图）。
 */
export async function submitCanvasTask(
  apiKey: string,
  request: CanvasGenerateRequest,
): Promise<CanvasSubmitResult> {
  const hasImage = request.image !== null && request.image.size > 0
  const path = hasImage ? '/v1/images/edits/async' : '/v1/images/generations/async'

  const form = new FormData()
  form.append('model', request.model)
  form.append('prompt', request.prompt)
  form.append('size', request.size)
  form.append('n', String(request.n ?? 1))
  if (request.quality) {
    form.append('quality', request.quality)
  }
  if (request.background) {
    form.append('background', request.background)
  }
  if (hasImage && request.image) {
    form.append('image', request.image, 'canvas.png')
  }
  if (hasImage && request.mask) {
    form.append('mask', request.mask, 'mask.png')
  }

  const response = await fetch(buildGatewayUrl(path), {
    method: 'POST',
    headers: authHeaders(apiKey),
    body: form,
  })
  if (!response.ok) {
    throw await parseCanvasError(response)
  }
  return response.json()
}

/**
 * getCanvasTask 轮询一次任务状态。
 */
export async function getCanvasTask(apiKey: string, taskId: string): Promise<CanvasTask> {
  const response = await fetch(
    buildGatewayUrl(`/v1/images/tasks/${encodeURIComponent(taskId)}`),
    { headers: authHeaders(apiKey) },
  )
  if (!response.ok) {
    throw await parseCanvasError(response)
  }
  return response.json()
}

/**
 * CanvasOutputImage 是从任务结果里提取出的一张图。
 */
export interface CanvasOutputImage {
  /** 图片直链。 */
  url?: string
  /** base64 内容。上游只返回这一种时使用。 */
  b64?: string
}

/**
 * extractOutputImages 从任务结果里取出图片列表。
 *
 * 结果体是 OpenAI images 格式：data 数组里每项要么有 url，要么有 b64_json。
 * 上游也可能直接返回单张图的 url 字段，两种都兼容。
 */
export function extractOutputImages(task: CanvasTask): CanvasOutputImage[] {
  const data = task.result?.data
  if (Array.isArray(data)) {
    const images: CanvasOutputImage[] = []
    for (const item of data) {
      const url = typeof item?.url === 'string' ? item.url : undefined
      const b64 = typeof item?.b64_json === 'string' ? item.b64_json : undefined
      if (url || b64) {
        images.push({ url, b64 })
      }
    }
    return images
  }
  const single = (task.result as { url?: string } | null)?.url
  return typeof single === 'string' && single ? [{ url: single }] : []
}

/**
 * fetchOutputImage 取回一张输出图的字节。
 *
 * 直链直接取；只有 base64 时在浏览器侧解码成 Blob，
 * 这样上层拿到的永远是可渲染的 Blob。
 */
export async function fetchOutputImage(image: CanvasOutputImage): Promise<Blob | null> {
  if (image.url) {
    try {
      const response = await fetch(image.url)
      if (!response.ok) {
        return null
      }
      return await response.blob()
    } catch {
      return null
    }
  }
  if (image.b64) {
    try {
      const binary = atob(image.b64)
      const bytes = new Uint8Array(binary.length)
      for (let i = 0; i < binary.length; i += 1) {
        bytes[i] = binary.charCodeAt(i)
      }
      return new Blob([bytes], { type: 'image/png' })
    } catch {
      return null
    }
  }
  return null
}
