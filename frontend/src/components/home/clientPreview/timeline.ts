/* 首屏客户端演示的时间线：19 秒一轮，每一帧的状态都由时间点推出来。
 * Agent 部分占约 14 秒，画图只占约 4 秒：
 * 1. 内置 Agent 干活：用户消息之后工具行依次出现
 * 2. 打开模型选择器，依次停在 内置 Agent → Claude Code → Codex CLI → 内置 Agent
 * 3. 这一轮做完：工具行收成「已工作 N 秒」，出现回复和改动文件卡，余额扣一次
 * 4. 侧栏点「画图」切到画图页，打出提示词、提交、生成中、出图，余额再扣一次
 * 5. 淡出，回到开头
 */

export type PreviewScene = 'agent' | 'image'
export type ImageJobState = 'idle' | 'drawing' | 'done'

export const CYCLE_MS = 19000
export const TOOL_ROW_COUNT = 6

const ROW_START = 800
const ROW_STAGGER = 900
const PICKER_OPEN = 6200
const PICKER_CLOSE = 10000
/** [时间点, 选择器里停留的 Agent 下标]，下标对应 ModelPicker 的 agents 顺序 */
const PICKER_STEPS: ReadonlyArray<readonly [number, number]> = [
  [PICKER_OPEN, 0],
  [7200, 2],
  [8200, 3],
  [9200, 0],
]
const RUN_DONE = 10800
const IMAGE_ROW_ACTIVE = 13600
const IMAGE_SCENE_AT = 14000
const TYPE_START = 14200
const TYPE_END = 15200
const IMAGE_SUBMIT = 15400
const IMAGE_DONE = 16800
const FADE_OUT = 18200
/** 下一轮开头先保持透明，等画图页褪掉、对话页就位后再淡入，避免两页叠在一起 */
const FADE_IN_AT = 400
/** 开始计时时这一轮已经跑了多少秒，让「工作中」和「已工作」的数字看起来像真的任务 */
const WORK_SECONDS_OFFSET = 31

/** reduced motion 用的静态帧：工具行全部出现、选择器还没打开，对话一眼看全 */
export const STATIC_T = 5600

export interface PreviewFrame {
  scene: PreviewScene
  /** 已经出现的工具行数 */
  visibleRows: number
  /** 这一轮是否已完成（工具行收起，出现回复和改动文件卡） */
  runDone: boolean
  pickerOpen: boolean
  /** 选择器左栏当前停留的 Agent 下标 */
  pickerIndex: number
  /** 侧栏「画图」行是否处于选中态 */
  imageRowActive: boolean
  /** 画图提示词打出来的比例，0~1 */
  typed: number
  /** 提示词是否已经提交（提交后输入框清空） */
  submitted: boolean
  imageJob: ImageJobState
  fading: boolean
  workSeconds: number
  drawSeconds: number
  /** 已经扣过几次费：0 / 1（Agent 这一轮）/ 2（再加一张图） */
  charges: number
}

function clamp01(value: number): number {
  return Math.min(1, Math.max(0, value))
}

export function deriveFrame(tMs: number): PreviewFrame {
  const t = ((tMs % CYCLE_MS) + CYCLE_MS) % CYCLE_MS

  let pickerIndex = PICKER_STEPS[0][1]
  for (const [at, index] of PICKER_STEPS) {
    if (t >= at) pickerIndex = index
  }

  return {
    scene: t >= IMAGE_SCENE_AT ? 'image' : 'agent',
    visibleRows:
      t < ROW_START ? 0 : Math.min(TOOL_ROW_COUNT, Math.floor((t - ROW_START) / ROW_STAGGER) + 1),
    runDone: t >= RUN_DONE,
    pickerOpen: t >= PICKER_OPEN && t < PICKER_CLOSE,
    pickerIndex,
    imageRowActive: t >= IMAGE_ROW_ACTIVE,
    typed: clamp01((t - TYPE_START) / (TYPE_END - TYPE_START)),
    submitted: t >= IMAGE_SUBMIT,
    imageJob: t < IMAGE_SUBMIT ? 'idle' : t < IMAGE_DONE ? 'drawing' : 'done',
    fading: t >= FADE_OUT || t < FADE_IN_AT,
    workSeconds: WORK_SECONDS_OFFSET + Math.floor(Math.min(t, RUN_DONE) / 1000),
    drawSeconds: Math.max(0, Math.floor((Math.min(t, IMAGE_DONE) - IMAGE_SUBMIT) / 1000)),
    charges: (t >= RUN_DONE ? 1 : 0) + (t >= IMAGE_DONE ? 1 : 0),
  }
}
