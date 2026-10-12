import { afterEach, describe, expect, it, vi } from 'vitest'

import { applyDarkClass } from '../themeSwitch'

describe('applyDarkClass', () => {
  afterEach(() => {
    document.documentElement.classList.remove('dark', 'theme-switching')
    vi.restoreAllMocks()
  })

  it('切换主题时冻结过渡两个渲染帧再恢复', () => {
    const frames: FrameRequestCallback[] = []
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      frames.push(callback)
      return frames.length
    })
    const root = document.documentElement

    applyDarkClass(true)
    expect(root.classList.contains('dark')).toBe(true)
    expect(root.classList.contains('theme-switching')).toBe(true)

    frames.shift()?.(0)
    expect(root.classList.contains('theme-switching')).toBe(true)
    frames.shift()?.(16)
    expect(root.classList.contains('theme-switching')).toBe(false)
  })

  it('主题没有变化时不冻结过渡', () => {
    const raf = vi.spyOn(window, 'requestAnimationFrame')
    applyDarkClass(false)
    expect(raf).not.toHaveBeenCalled()
    expect(document.documentElement.classList.contains('theme-switching')).toBe(false)
  })
})
