import { computed, reactive, ref, shallowRef, watch } from 'vue'
import {
  CHAPTERS,
  CYCLE_MS,
  FADE_IN_MS,
  STATIC_T,
  chapterProgressAt,
  deriveFrame,
  settleT,
  type PreviewFrame,
} from './timeline'

/** 帧状态的刷新粒度：100ms 足够驱动所有状态切换，避免每帧重建 frame 对象 */
const STEP_MS = 100
/** 单帧最多推进的时间：标签页被节流或刚恢复时不会一下跳过好几段 */
const MAX_DT_MS = 250

export type PauseReason = 'user' | 'hover' | 'focus' | 'offscreen' | 'hidden'

/**
 * 首屏客户端演示的时钟。
 * 播放进度按帧累加（不是 now - start），所以暂停、切走标签页、滚出视口后都从停下的地方接着播；
 * 任一暂停原因成立就停掉 rAF。减少动态效果时不启动 rAF，章节按钮在各章的静态帧之间切换。
 */
export function usePreviewClock(reducedMotion: boolean) {
  let elapsed = reducedMotion ? STATIC_T : 0
  let lastStep = -1
  let lastNow: number | null = null
  let rafId = 0
  let active = false

  const frame = shallowRef<PreviewFrame>(deriveFrame(elapsed))
  const progress = ref(chapterProgressAt(elapsed))
  const staticIndex = ref(0)
  const reasons = reactive<Record<PauseReason, boolean>>({
    user: false,
    hover: false,
    focus: false,
    offscreen: false,
    hidden: false,
  })

  const paused = computed(() => Object.values(reasons).some(Boolean))
  /** 用户自己造成的暂停（按钮、悬停、键盘聚焦），章节导航的按钮显示的是这个 */
  const interactivePaused = computed(() => reasons.user || reasons.hover || reasons.focus)
  const currentChapter = computed(() => (reducedMotion ? staticIndex.value : frame.value.chapterIndex))

  function render(t: number) {
    const step = Math.floor(t / STEP_MS) * STEP_MS
    if (step === lastStep) return
    lastStep = step
    frame.value = deriveFrame(step)
    progress.value = chapterProgressAt(step)
  }

  function tick(now: number) {
    rafId = 0
    if (paused.value || !active) return
    if (lastNow !== null) {
      const dt = Math.min(Math.max(now - lastNow, 0), MAX_DT_MS)
      elapsed = (elapsed + dt) % CYCLE_MS
    }
    lastNow = now
    render(elapsed)
    rafId = requestAnimationFrame(tick)
  }

  function startLoop() {
    if (reducedMotion || !active || paused.value || rafId) return
    if (typeof requestAnimationFrame !== 'function') return
    lastNow = null
    rafId = requestAnimationFrame(tick)
  }

  function stopLoop() {
    if (rafId && typeof cancelAnimationFrame === 'function') cancelAnimationFrame(rafId)
    rafId = 0
    lastNow = null
  }

  watch(paused, (value) => {
    if (value) stopLoop()
    else startLoop()
  })

  function setPaused(reason: PauseReason, value: boolean) {
    if (reasons[reason] === value) return
    // 用户让它停下时，挪到最近的不透明帧，不停在淡入淡出的空白上
    const interactive = reason === 'user' || reason === 'hover' || reason === 'focus'
    if (value && interactive && !interactivePaused.value && !reducedMotion) {
      elapsed = settleT(elapsed)
      render(elapsed)
    }
    reasons[reason] = value
  }

  function togglePaused() {
    if (interactivePaused.value) {
      reasons.user = false
      reasons.hover = false
      reasons.focus = false
    } else {
      setPaused('user', true)
    }
  }

  function jumpTo(index: number) {
    const chapter = CHAPTERS[index]
    if (!chapter) return
    if (reducedMotion) {
      staticIndex.value = index
      frame.value = deriveFrame(chapter.staticT)
      return
    }
    // 播放中从这一章开头（跳过淡入）接着播；暂停时停在这一章的代表帧
    elapsed = paused.value ? chapter.staticT : chapter.start + FADE_IN_MS
    lastStep = -1
    lastNow = null
    render(elapsed)
  }

  function start() {
    active = true
    startLoop()
  }

  function dispose() {
    active = false
    stopLoop()
  }

  return {
    frame,
    progress,
    paused,
    interactivePaused,
    currentChapter,
    setPaused,
    togglePaused,
    jumpTo,
    start,
    dispose,
  }
}
