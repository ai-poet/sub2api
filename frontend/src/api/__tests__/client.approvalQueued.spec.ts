import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { apiClient } from '../client'
import { APPROVAL_PENDING_CODE, APPROVAL_QUEUED_EVENT, isApprovalQueued } from '@/utils/approval'

// 直接调用响应拦截器的成功分支：运维管理员的写请求被排队时后端返回 202 + approval_request_id，
// 拦截器必须把它转成拒绝（APPROVAL_PENDING）并广播 approval-queued，普通 200 / 其它 202 不受影响。
type Fulfilled = (response: AxiosResponse) => AxiosResponse | Promise<AxiosResponse>

function responseFulfilled(): Fulfilled {
  const handlers = (apiClient.interceptors.response as unknown as { handlers: Array<{ fulfilled: Fulfilled }> }).handlers
  const handler = handlers.find((h) => typeof h?.fulfilled === 'function')
  if (!handler) throw new Error('response interceptor not registered')
  return handler.fulfilled
}

function makeResponse(status: number, body: unknown): AxiosResponse {
  return {
    status,
    statusText: '',
    headers: {},
    config: { headers: {} } as InternalAxiosRequestConfig,
    data: body
  }
}

describe('apiClient response interceptor — approval queued (202)', () => {
  const dispatchSpy = vi.spyOn(window, 'dispatchEvent')

  beforeEach(() => {
    dispatchSpy.mockClear()
  })

  it('rejects a queued write with APPROVAL_PENDING and broadcasts approval-queued', async () => {
    const approval = { approval_request_id: 42, status: 'pending', action: 'admin.users.balance.create', target_summary: 'u@x.io', expires_at: '2026-09-19T00:00:00Z' }
    const promise = responseFulfilled()(makeResponse(202, { code: 0, message: 'accepted', data: approval }))

    await expect(promise).rejects.toMatchObject({ status: 202, code: APPROVAL_PENDING_CODE, approval })
    const err = await promise.catch((e) => e)
    expect(isApprovalQueued(err)).toBe(true)

    const queued = dispatchSpy.mock.calls.map(([event]) => event as CustomEvent).find((event) => event.type === APPROVAL_QUEUED_EVENT)
    expect(queued).toBeTruthy()
    expect(queued!.detail).toEqual(approval)
  })

  it('unwraps ordinary successes and leaves a 202 without an approval id alone', async () => {
    const ok = await responseFulfilled()(makeResponse(200, { code: 0, message: 'success', data: { id: 1 } }))
    expect(ok.data).toEqual({ id: 1 })

    const accepted = await responseFulfilled()(makeResponse(202, { code: 0, message: 'accepted', data: { task_id: 'abc' } }))
    expect(accepted.data).toEqual({ task_id: 'abc' })

    expect(dispatchSpy.mock.calls.some(([event]) => (event as Event).type === APPROVAL_QUEUED_EVENT)).toBe(false)
  })

  it('still rejects business errors carried in a 200 envelope', async () => {
    await expect(responseFulfilled()(makeResponse(200, { code: 40001, message: 'nope' }))).rejects.toMatchObject({ code: 40001, message: 'nope' })
  })
})
