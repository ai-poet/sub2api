import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

// 单次循环时长（ms）。所有阶段都从这个时钟派生，保证 sidebar / stream / composer 同步。
export const CYCLE_MS = 20000

// stream 阶段相对整个循环的起点
const STREAM_START = 4000
// draft 阶段重新开始（回到输入态）
const RESET_AT = 19200

// prompt 打字区间
const TYPE_START = 400
const TYPE_END = 3400
// 发送按钮脉冲
const SEND_AT = 3650
const SEND_DURATION = 420

// 以下时间都相对 STREAM_START
const USER_SHOW = 0
const INTRO_TYPE_START = 300
const INTRO_TYPE_END = 1900
const COMPLETE_AT = 14600

export interface ToolStepDef {
  id: string
  icon: string
  label: string
  detail: string
}

// 步骤模拟：相对 STREAM_START 的 run/done 时间
interface StepTiming {
  run: number
  done: number
}

const STEP_TIMINGS: StepTiming[] = [
  { run: 2000, done: 3300 },
  { run: 3300, done: 4700 },
  { run: 4700, done: 6300 },
  { run: 6300, done: 7700 },
  { run: 7700, done: 9100 },
  { run: 9100, done: 10700 },
  { run: 10700, done: 12200 },
]

const FINAL_LINE1_START = 12400
const FINAL_LINE1_END = 13600
const FINAL_LINE2_START = 13600
const FINAL_LINE2_END = 14400

export type StepState = 'pending' | 'running' | 'done'

export interface StreamFrame {
  stage: 'draft' | 'stream'
  // draft
  promptRatio: number
  caretOn: boolean
  sendPulse: number
  // stream
  showUser: boolean
  introVisible: boolean
  introRatio: number
  steps: StepState[]
  // 当前正在执行的步骤下标（-1 表示无），working dots 贴在它后面
  runningStep: number
  // 流式输出阶段（intro 打完且首个步骤尚未出现之前也算 running）
  agentRunning: boolean
  finalVisible: boolean
  finalRatio1: number
  finalRatio2: number
  complete: boolean
  // 侧边栏当前高亮的工作区下标（在 selectable 列表中）
  activeWorktree: number
  liveComposerVisible: boolean
}

export interface TimelineControls {
  frame: Ref<StreamFrame>
  // 用户手动选中的 selectable 索引；null = 自动轮转。
  manualWorktree: Ref<number | null>
  // 切换到指定工作区并从头重放
  selectWorktree: (index: number) => void
  // 重置时钟从头跑一次
  restart: () => void
}

function clamp01(value: number): number {
  if (value <= 0) return 0
  if (value >= 1) return 1
  return value
}

function ratio(elapsed: number, start: number, end: number): number {
  return clamp01((elapsed - start) / (end - start))
}

export function deriveFrame(
  t: number,
  stepCount: number,
  worktreeCount: number,
  manualIndex: number | null = null,
): StreamFrame {
  const inStream = t >= STREAM_START && t < RESET_AT
  const rel = t - STREAM_START

  const steps: StepState[] = []
  let runningStep = -1
  for (let i = 0; i < stepCount; i += 1) {
    const timing = STEP_TIMINGS[i] ?? STEP_TIMINGS[STEP_TIMINGS.length - 1]
    if (!inStream || rel < timing.run) {
      steps.push('pending')
    } else if (rel < timing.done) {
      steps.push('running')
      if (runningStep === -1) runningStep = i
    } else {
      steps.push('done')
    }
  }

  const introVisible = inStream && rel >= INTRO_TYPE_START
  const allStepsDone = inStream && rel >= (STEP_TIMINGS[stepCount - 1]?.done ?? 0)
  const complete = inStream && rel >= COMPLETE_AT
  // agent 仍在工作：intro 出现后、所有步骤完成前；步骤跑完进入总结即收起
  const agentRunning = introVisible && !allStepsDone

  // 手动选中时钉死；否则按 ~3.8s 自动轮转
  let activeWorktree = 0
  if (worktreeCount > 0) {
    if (manualIndex !== null) {
      activeWorktree = ((manualIndex % worktreeCount) + worktreeCount) % worktreeCount
    } else {
      activeWorktree = Math.floor(t / 3800) % worktreeCount
    }
  }

  return {
    stage: inStream ? 'stream' : 'draft',
    promptRatio: ratio(t, TYPE_START, TYPE_END),
    caretOn: Math.floor(t / 500) % 2 === 0,
    sendPulse: t >= SEND_AT && t < SEND_AT + SEND_DURATION
      ? Math.sin(((t - SEND_AT) / SEND_DURATION) * Math.PI)
      : 0,
    showUser: inStream && rel >= USER_SHOW,
    introVisible,
    introRatio: ratio(rel, INTRO_TYPE_START, INTRO_TYPE_END),
    steps,
    runningStep,
    agentRunning,
    finalVisible: allStepsDone,
    finalRatio1: ratio(rel, FINAL_LINE1_START, FINAL_LINE1_END),
    finalRatio2: ratio(rel, FINAL_LINE2_START, FINAL_LINE2_END),
    complete,
    activeWorktree,
    liveComposerVisible: inStream,
  }
}

function prefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

// 降级帧：直接展示完成态，不做动画
function staticFrame(stepCount: number, manualIndex: number | null): StreamFrame {
  return {
    stage: 'stream',
    promptRatio: 1,
    caretOn: false,
    sendPulse: 0,
    showUser: true,
    introVisible: true,
    introRatio: 1,
    steps: Array.from({ length: stepCount }, () => 'done' as StepState),
    runningStep: -1,
    agentRunning: false,
    finalVisible: true,
    finalRatio1: 1,
    finalRatio2: 1,
    complete: true,
    activeWorktree: manualIndex ?? 0,
    liveComposerVisible: true,
  }
}

export function useTimeline(stepCount: number, worktreeCount: number): TimelineControls {
  const manualWorktree = ref<number | null>(null)
  const frame = ref<StreamFrame>(deriveFrame(0, stepCount, worktreeCount, manualWorktree.value))
  let rafId = 0
  let start = 0
  let lastTick = -100
  let reduced = false

  function loop(now: number) {
    if (start === 0) start = now
    const t = (now - start) % CYCLE_MS
    if (Math.abs(t - lastTick) >= 50 || t < lastTick) {
      lastTick = t
      frame.value = deriveFrame(t, stepCount, worktreeCount, manualWorktree.value)
    }
    rafId = requestAnimationFrame(loop)
  }

  function restart() {
    if (reduced) {
      frame.value = staticFrame(stepCount, manualWorktree.value)
      return
    }
    if (typeof performance !== 'undefined') {
      start = performance.now()
    } else {
      start = 0
    }
    lastTick = -100
    // 立刻把帧拨到 t=0，UI 同步重启
    frame.value = deriveFrame(0, stepCount, worktreeCount, manualWorktree.value)
  }

  function selectWorktree(index: number) {
    manualWorktree.value = index
    restart()
  }

  onMounted(() => {
    if (prefersReducedMotion()) {
      reduced = true
      frame.value = staticFrame(stepCount, manualWorktree.value)
      return
    }
    rafId = requestAnimationFrame(loop)
  })

  onBeforeUnmount(() => {
    if (rafId) cancelAnimationFrame(rafId)
  })

  return { frame, manualWorktree, selectWorktree, restart }
}
