import { getAuthToken, refreshToken as refreshAuthToken } from './auth'
import { getLocale } from '@/i18n'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

export type ModelMirrorVerdict =
  | 'pending'
  | 'max_pure'
  | 'official_api'
  | 'reverse_proxy'
  | 'likely_not_claude'

export interface ModelMirrorKnowledgeProbe {
  id: string
  prompt: string
  expected_keywords: string[]
  pass_mode: 'any' | 'all'
  weight: number
  enabled: boolean
}

export interface ModelMirrorVerifyRequest {
  api_endpoint: string
  api_key: string
  api_model: string
}

export interface ModelMirrorCheckResult {
  id: string
  label: string
  weight: number
  pass: boolean
  detail: string
  status?: 'pass' | 'fail' | 'info' | ''
}

export interface ModelMirrorDonePayload {
  score: number
  verdict: ModelMirrorVerdict
  total_checks: number
  response_excerpt?: string
  thinking_excerpt?: string
  upstream_model?: string
}

interface VerifyCallbacks {
  onStep?: (message: string) => void
  onCheck?: (result: ModelMirrorCheckResult) => void
  onDone?: (payload: ModelMirrorDonePayload) => void
  onError?: (message: string) => void
}

export async function verifyModelMirror(
  request: ModelMirrorVerifyRequest,
  callbacks: VerifyCallbacks,
  signal?: AbortSignal
): Promise<void> {
  let response = await doModelMirrorRequest(request, signal)

  if (response.status === 401) {
    await refreshAuthToken()
    response = await doModelMirrorRequest(request, signal)
  }

  if (!response.ok) {
    const message = await parseModelMirrorError(response)
    throw new Error(message)
  }

  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error('Unable to read verification stream')
  }

  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) {
      break
    }

    buffer += decoder.decode(value, { stream: true })
    const chunks = buffer.split('\n\n')
    buffer = chunks.pop() || ''

    for (const chunk of chunks) {
      dispatchModelMirrorEvent(chunk, callbacks)
    }
  }

  if (buffer.trim()) {
    dispatchModelMirrorEvent(buffer, callbacks)
  }
}

async function doModelMirrorRequest(
  request: ModelMirrorVerifyRequest,
  signal?: AbortSignal
): Promise<Response> {
  const token = getAuthToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'Accept-Language': getLocale()
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  return fetch(`${API_BASE_URL}/tools/model-mirror/verify`, {
    method: 'POST',
    headers,
    body: JSON.stringify(request),
    signal
  })
}

function dispatchModelMirrorEvent(chunk: string, callbacks: VerifyCallbacks) {
  const lines = chunk
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)

  let event = ''
  const dataLines: string[] = []

  for (const line of lines) {
    if (line.startsWith('event:')) {
      event = line.slice(6).trim()
      continue
    }
    if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trim())
    }
  }

  if (!event || dataLines.length === 0) {
    return
  }

  try {
    const payload = JSON.parse(dataLines.join('\n'))
    switch (event) {
      case 'step':
        callbacks.onStep?.(String(payload.message || ''))
        break
      case 'check':
        callbacks.onCheck?.(payload as ModelMirrorCheckResult)
        break
      case 'done':
        callbacks.onDone?.(payload as ModelMirrorDonePayload)
        break
      case 'error':
        callbacks.onError?.(String(payload.message || 'Verification failed'))
        break
    }
  } catch {
    // ignore malformed SSE payload
  }
}

async function parseModelMirrorError(response: Response): Promise<string> {
  const contentType = response.headers.get('content-type') || ''
  if (contentType.includes('application/json')) {
    const data = (await response.json().catch(() => null)) as
      | { message?: string; data?: { message?: string } }
      | null
    if (data?.message) {
      return data.message
    }
    if (data?.data?.message) {
      return data.data.message
    }
  }

  const text = await response.text().catch(() => '')
  return text || `HTTP ${response.status}`
}

export const modelMirrorAPI = {
  verifyModelMirror
}

export default modelMirrorAPI
