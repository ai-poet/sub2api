/* 首屏客户端演示的时间线：约 17.7 秒一轮，分三幕，每一帧的状态都由时间点推出来。
 * 1. 计划 → 执行：计划模式下先交计划（右侧计划面板），批准后派两个子智能体并行查找，再改代码、跑测试
 * 2. 智能体团队：/agent-teams 拟好三人团队，在团队面板确认启动，任务按依赖逐个完成，T2 检查没过返工一轮
 * 3. 画图：打出提示词、提交、生成中、出图
 * 节奏尽量紧：等待审批、做完后的停留都控制在 1 秒上下，幕间淡出淡入合计不到 0.7 秒。
 * 每幕开头淡入、结尾淡出，切换内容发生在全透明的那一刻。
 */

export type ChapterId = 'plan' | 'team' | 'image'
export type PreviewScene = 'agent' | 'image'
export type ImageJobState = 'idle' | 'drawing' | 'done'
/** 同客户端 turn_status.rs 的 TurnAction */
export type TurnStatus = 'waitingForModel' | 'runningTool' | 'runningTools' | 'thinking' | 'writing' | 'waitingForYou'
export type PanelKind = 'none' | 'plan' | 'team'
export type PlanState = 'none' | 'pending' | 'approved'
export type TeamPhase = 'review' | 'running' | 'done'
export type TeamMemberState = 'waiting' | 'working' | 'idle'
export type TeamTaskState = 'blocked' | 'open' | 'running' | 'completed'

export interface Chapter {
  id: ChapterId
  start: number
  end: number
  /** 暂停或减少动态效果时停在这一章的哪一帧 */
  staticT: number
}

export const CYCLE_MS = 17650
/** 淡入 / 淡出时长；窗口的透明度过渡（HomeAgentWorkflowPreview 的 duration-300）不能比淡出长 */
export const FADE_IN_MS = 300
export const FADE_OUT_MS = 350

export const CHAPTERS: readonly Chapter[] = [
  { id: 'plan', start: 0, end: 8750, staticT: 3500 },
  { id: 'team', start: 8750, end: 13800, staticT: 11300 },
  { id: 'image', start: 13800, end: CYCLE_MS, staticT: 16600 },
]

/** 第一章的静态帧：计划已交、等你审批，卡片和计划面板都在 */
export const STATIC_T = CHAPTERS[0].staticT

export const PLAN_ROW_COUNT = 3
export const SUBAGENT_COUNT = 2
export const EXEC_ROW_COUNT = 3
export const TEAM_MEMBER_COUNT = 3
export const TEAM_TASK_COUNT = 4
/** 任务 → 负责的成员下标（架构 / 实现 / 测试） */
export const TEAM_TASK_OWNERS: readonly number[] = [0, 1, 2, 0]
/** 任务依赖：T2 等 T1，T3 等 T2，T4 等 T1 */
export const TEAM_TASK_DEPS: ReadonlyArray<readonly number[]> = [[], [0], [1], [0]]

export const MOMENTS = {
  // 第一幕
  pickerOpen: 250,
  pickerClose: 1200,
  send: 1400,
  planRows: [1700, 2150, 2600],
  planReady: 3000,
  approvePress: 3900,
  approved: 4150,
  subagentRows: [4300, 4450],
  subagentsDone: 5500,
  execRows: [5650, 6000, 6350],
  runDone: 7200,
  teamSessionCue: 8100,
  // 第二幕
  teamApprovePress: 9700,
  teamStart: 9900,
  teamRound2: 11150,
  teamDone: 12600,
  imageRowCue: 13150,
  // 第三幕
  typeStart: 14200,
  typeEnd: 14900,
  imageSubmit: 15050,
  imageDone: 16100,
} as const

/** 开始计时时这一轮已经跑了多少秒，让「工作中」的数字看起来像真的任务 */
const WORK_SECONDS_OFFSET = 26

export interface TeamFrame {
  phase: TeamPhase
  /** 「确认并启动」正被按下 */
  approvePressed: boolean
  members: readonly TeamMemberState[]
  /** 每个成员正在处理的任务下标 */
  memberTask: ReadonlyArray<number | null>
  tasks: readonly TeamTaskState[]
  /** 每个任务当前是第几轮（检查没过、返工后加一） */
  rounds: readonly number[]
  messages: number
}

export interface PreviewFrame {
  chapter: ChapterId
  chapterIndex: number
  scene: PreviewScene
  fading: boolean
  /** 侧栏选中的会话 */
  activeSession: 'plan' | 'team' | null
  /** 侧栏「画图」行是否处于选中态 */
  imageRowActive: boolean

  /** 输入框里还是没发出去的草稿 */
  draft: boolean
  pickerOpen: boolean
  mode: 'plan' | 'build'
  sent: boolean
  planRows: number
  subagentRows: number
  subagentsRunning: boolean
  execRows: number
  /** 第一幕那一轮已结束（之后的幕里一直为真） */
  runDone: boolean
  status: TurnStatus | null
  statusTools: number
  turnTokens: number
  workSeconds: number
  /** 顶栏信息按钮上的后台工作脉冲点 */
  backgroundLive: boolean
  plan: PlanState
  /** 输入框上方的「规划完成」卡片 */
  planCard: boolean
  planApprovePressed: boolean
  /** 宽屏时右侧以一列显示的面板 */
  panel: PanelKind
  /** 窄屏时盖在对话上的面板；不为 none 时总与 panel 相同 */
  sheet: PanelKind

  team: TeamFrame | null
  teamReply: 'none' | 'draft' | 'done'

  /** 画图提示词打出来的比例，0~1 */
  typed: number
  submitted: boolean
  imageJob: ImageJobState
  drawSeconds: number
  /** 已经扣过几次费：Agent 这一轮、团队、一张图 */
  charges: number
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

function countReached(t: number, moments: readonly number[]): number {
  return moments.filter((at) => t >= at).length
}

function wrap(tMs: number): number {
  return ((tMs % CYCLE_MS) + CYCLE_MS) % CYCLE_MS
}

export function chapterIndexAt(tMs: number): number {
  const t = wrap(tMs)
  const index = CHAPTERS.findIndex((chapter) => t >= chapter.start && t < chapter.end)
  return index < 0 ? 0 : index
}

/** 当前章走过的比例，0~1，给章节导航的进度条用 */
export function chapterProgressAt(tMs: number): number {
  const t = wrap(tMs)
  const chapter = CHAPTERS[chapterIndexAt(t)]
  return clamp((t - chapter.start) / (chapter.end - chapter.start), 0, 1)
}

function isFading(t: number, chapter: Chapter): boolean {
  return t < chapter.start + FADE_IN_MS || t >= chapter.end - FADE_OUT_MS
}

/** 暂停时把时间挪到最近的不透明帧，免得停在淡入淡出的空白上 */
export function settleT(tMs: number): number {
  const t = wrap(tMs)
  const chapter = CHAPTERS[chapterIndexAt(t)]
  if (t < chapter.start + FADE_IN_MS) return chapter.start + FADE_IN_MS
  if (t >= chapter.end - FADE_OUT_MS) return chapter.end - FADE_OUT_MS - 100
  return t
}

/** 同客户端 waku-protocol usage.rs 的 format_tokens：950 / 1.2k */
export function formatTokens(count: number): string {
  if (count >= 1000) return `${(count / 1000).toFixed(1)}k`
  return String(count)
}

function planStatus(t: number): TurnStatus | null {
  const m = MOMENTS
  if (t < m.send || t >= m.runDone) return null
  if (t < m.planRows[0]) return 'waitingForModel'
  if (t < m.planRows[1]) return 'runningTool'
  if (t < m.planRows[2]) return 'thinking'
  if (t < m.planReady) return 'writing'
  if (t < m.approved) return 'waitingForYou'
  if (t < m.subagentsDone) return 'runningTools'
  if (t < m.execRows[0]) return 'thinking'
  return 'runningTool'
}

function turnTokens(t: number): number {
  if (t < MOMENTS.send) return 0
  const planning = 0.5 * clamp(t - MOMENTS.send, 0, MOMENTS.planReady - MOMENTS.send)
  const running = 0.75 * clamp(t - MOMENTS.approved, 0, MOMENTS.runDone - MOMENTS.approved)
  return Math.round((planning + running) / 100) * 100
}

interface TeamKeyframe {
  at: number
  members: readonly TeamMemberState[]
  memberTask: ReadonlyArray<number | null>
  tasks: readonly TeamTaskState[]
  messages: number
}

// 团队面板的关键帧：成员按依赖领取任务，T2 第一轮没过检查，返工后在 11900 通过
const TEAM_KEYFRAMES: readonly TeamKeyframe[] = [
  { at: CHAPTERS[1].start, members: ['waiting', 'waiting', 'waiting'], memberTask: [null, null, null], tasks: ['open', 'blocked', 'blocked', 'blocked'], messages: 0 },
  { at: MOMENTS.teamStart, members: ['working', 'waiting', 'waiting'], memberTask: [0, null, null], tasks: ['running', 'blocked', 'blocked', 'blocked'], messages: 1 },
  { at: 10500, members: ['idle', 'waiting', 'waiting'], memberTask: [null, null, null], tasks: ['completed', 'open', 'blocked', 'open'], messages: 3 },
  { at: 10700, members: ['working', 'working', 'waiting'], memberTask: [3, 1, null], tasks: ['completed', 'running', 'blocked', 'running'], messages: 4 },
  { at: MOMENTS.teamRound2, members: ['working', 'working', 'waiting'], memberTask: [3, 1, null], tasks: ['completed', 'running', 'blocked', 'running'], messages: 6 },
  { at: 11500, members: ['idle', 'working', 'waiting'], memberTask: [null, 1, null], tasks: ['completed', 'running', 'blocked', 'completed'], messages: 7 },
  { at: 11900, members: ['idle', 'idle', 'waiting'], memberTask: [null, null, null], tasks: ['completed', 'completed', 'open', 'completed'], messages: 9 },
  { at: 12050, members: ['idle', 'idle', 'working'], memberTask: [null, null, 2], tasks: ['completed', 'completed', 'running', 'completed'], messages: 10 },
  { at: MOMENTS.teamDone, members: ['idle', 'idle', 'idle'], memberTask: [null, null, null], tasks: ['completed', 'completed', 'completed', 'completed'], messages: 12 },
]

function teamFrame(t: number): TeamFrame {
  let keyframe = TEAM_KEYFRAMES[0]
  for (const candidate of TEAM_KEYFRAMES) {
    if (t >= candidate.at) keyframe = candidate
  }
  const phase: TeamPhase = t < MOMENTS.teamStart ? 'review' : t < MOMENTS.teamDone ? 'running' : 'done'
  return {
    phase,
    approvePressed: t >= MOMENTS.teamApprovePress && t < MOMENTS.teamStart,
    members: keyframe.members,
    memberTask: keyframe.memberTask,
    tasks: keyframe.tasks,
    rounds: [1, t >= MOMENTS.teamRound2 ? 2 : 1, 1, 1],
    messages: keyframe.messages,
  }
}

export function deriveFrame(tMs: number): PreviewFrame {
  const t = wrap(tMs)
  const m = MOMENTS
  const chapterIndex = chapterIndexAt(t)
  const chapter = CHAPTERS[chapterIndex]
  const inPlan = chapter.id === 'plan'
  const inTeam = chapter.id === 'team'

  const planState: PlanState = !inPlan || t < m.planReady ? 'none' : t < m.approved ? 'pending' : 'approved'
  const status = inPlan ? planStatus(t) : null
  const subagentRows = inPlan ? countReached(t, m.subagentRows) : 0

  let panel: PanelKind = 'none'
  let sheet: PanelKind = 'none'
  if (planState !== 'none') {
    panel = 'plan'
    if (planState === 'pending') sheet = 'plan'
  } else if (inTeam) {
    panel = 'team'
    if (t < m.teamDone) sheet = 'team'
  }

  let activeSession: PreviewFrame['activeSession'] = null
  if (inPlan) activeSession = t >= m.teamSessionCue ? 'team' : 'plan'
  else if (inTeam) activeSession = t >= m.imageRowCue ? null : 'team'

  return {
    chapter: chapter.id,
    chapterIndex,
    scene: chapter.id === 'image' ? 'image' : 'agent',
    fading: isFading(t, chapter),
    activeSession,
    imageRowActive: t >= m.imageRowCue,

    draft: inPlan && t < m.send,
    pickerOpen: inPlan && t >= m.pickerOpen && t < m.pickerClose,
    mode: inPlan && t < m.approved ? 'plan' : 'build',
    sent: !inPlan || t >= m.send,
    planRows: inPlan ? countReached(t, m.planRows) : 0,
    subagentRows,
    subagentsRunning: subagentRows > 0 && t < m.subagentsDone,
    execRows: inPlan ? countReached(t, m.execRows) : 0,
    runDone: t >= m.runDone,
    status,
    statusTools: status === 'runningTools' ? SUBAGENT_COUNT : status === 'runningTool' ? 1 : 0,
    turnTokens: inPlan ? turnTokens(Math.min(t, m.runDone)) : 0,
    workSeconds: WORK_SECONDS_OFFSET + Math.floor((clamp(t, m.send, m.runDone) - m.send) / 1000),
    backgroundLive: subagentRows > 0 && t < m.subagentsDone,
    plan: planState,
    planCard: planState === 'pending',
    planApprovePressed: inPlan && t >= m.approvePress && t < m.approved,
    panel,
    sheet,

    team: inTeam ? teamFrame(t) : null,
    teamReply: !inTeam ? 'none' : t >= m.teamDone ? 'done' : 'draft',

    typed: clamp((t - m.typeStart) / (m.typeEnd - m.typeStart), 0, 1),
    submitted: t >= m.imageSubmit,
    imageJob: t < m.imageSubmit ? 'idle' : t < m.imageDone ? 'drawing' : 'done',
    drawSeconds: Math.max(0, Math.floor((Math.min(t, m.imageDone) - m.imageSubmit) / 1000)),
    charges: (t >= m.runDone ? 1 : 0) + (t >= m.teamDone ? 1 : 0) + (t >= m.imageDone ? 1 : 0),
  }
}
