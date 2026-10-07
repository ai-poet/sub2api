<template>
  <div
    ref="root"
    class="agent-workflow-preview"
    data-test="agent-workflow-preview"
    @focusin="onFocusIn"
    @focusout="onFocusOut"
  >
    <!-- 按客户端真实界面画的主窗口：client/src/app/{render,sidebar,composer,plan_review,team_panel,image_studio_view}.rs。
         窗口整体是一张图（role="img"），里面不放可聚焦的元素；章节导航在它外面。 -->
    <div
      class="client-window relative mx-auto w-full max-w-[1100px] overflow-hidden rounded-xl border border-black/10 text-left shadow-[0_24px_64px_rgba(15,17,20,0.12)] dark:border-white/10 dark:shadow-[0_24px_64px_rgba(0,0,0,0.45)]"
      role="img"
      :aria-label="t('home.clientWorkflow.ariaLabel', { siteName: props.siteName })"
      data-test="preview-window"
      @pointermove="onPointerMove"
      @pointerleave="onPointerLeave"
    >
      <div class="flex h-[520px] items-stretch sm:h-[560px]">
        <!-- ===== 侧栏 ===== -->
        <aside class="cw-sidebar hidden w-[252px] shrink-0 flex-col sm:flex" data-test="preview-sidebar">
          <div class="flex h-12 flex-none items-center gap-0.5 px-2">
            <span v-if="isMac" class="ml-1.5 mr-3 flex items-center gap-2" aria-hidden="true">
              <i class="h-3 w-3 rounded-full bg-[#FF5F57]"></i>
              <i class="h-3 w-3 rounded-full bg-[#FEBC2E]"></i>
              <i class="h-3 w-3 rounded-full bg-[#28C840]"></i>
            </span>
            <span class="cw-icon-button"><ClientIcon name="panelLeft" class="h-3.5 w-3.5" /></span>
            <span class="cw-icon-button"><ClientIcon name="arrowLeft" class="h-3.5 w-3.5" /></span>
            <span class="cw-icon-button opacity-50"><ClientIcon name="arrowRight" class="h-3.5 w-3.5" /></span>
          </div>

          <nav class="flex flex-col gap-px px-2.5">
            <span
              v-for="action in sidebarActions"
              :key="action.key"
              class="cw-nav-row"
              :class="action.active ? 'is-active' : ''"
              :data-test="`preview-sidebar-${action.key}`"
            >
              <span class="flex h-5 w-5 items-center justify-center">
                <ClientIcon :name="action.icon" class="h-3.5 w-3.5" />
              </span>
              <span>{{ action.label }}</span>
            </span>
          </nav>

          <!-- 任务列表（sidebar_toolbar.rs / sidebar_sections.rs / task_rows.rs）：
               工具栏切换「按项目 / 时间线」，项目带展开箭头，任务单行、状态在标题左边 -->
          <div class="mt-2.5 flex flex-col px-2.5" data-test="preview-task-list">
            <div class="flex h-9 items-center gap-1 pl-0.5" data-test="preview-sidebar-toolbar">
              <span class="flex h-7 min-w-0 items-center gap-0.5 rounded-full bg-[color:var(--cw-overlay-strong)] p-0.5">
                <span class="cw-segment is-active" data-test="preview-view-project">{{ t('home.clientWorkflow.sidebar.viewByProject') }}</span>
                <span class="cw-segment">{{ t('home.clientWorkflow.sidebar.viewTimeline') }}</span>
              </span>
              <span class="cw-tool-button"><ClientIcon name="chevronsDownUp" class="h-3.5 w-3.5" /></span>
              <span class="ml-auto flex items-center gap-0.5">
                <span class="cw-tool-button"><ClientIcon name="listFilter" class="h-3.5 w-3.5" /></span>
                <span class="cw-tool-button"><ClientIcon name="archive" class="h-3.5 w-3.5" /></span>
              </span>
            </div>

            <template v-for="row in taskRows" :key="row.key">
              <div v-if="row.kind === 'section'" class="cw-section-header">
                <span class="truncate">{{ row.label }}</span>
                <span
                  v-if="row.addProject"
                  class="ml-auto flex h-[22px] w-5 items-center justify-center text-[color:var(--cw-text-secondary)]"
                >
                  <ClientIcon name="folderNew" class="h-3.5 w-3.5" />
                </span>
              </div>
              <div v-else-if="row.kind === 'project'" class="cw-project-row" data-test="preview-project-row">
                <span class="flex w-3.5 shrink-0 justify-center text-[color:var(--cw-text-tertiary)]">
                  <ClientIcon name="chevronDown" class="h-3 w-3" />
                </span>
                <span class="ml-0.5 flex w-4 shrink-0 justify-center text-[color:var(--cw-text-tertiary)]">
                  <ClientIcon name="folderOpen" class="h-3.5 w-3.5" />
                </span>
                <span class="ml-1.5 min-w-0 truncate text-[13.5px] text-[color:var(--cw-text-secondary)]">{{ row.label }}</span>
              </div>
              <div v-else-if="row.kind === 'spacer'" class="h-2.5"></div>
              <div
                v-else
                class="cw-task"
                :class="[row.active ? 'is-active' : '', row.nested ? 'is-nested' : '']"
                :data-test="`preview-session-${row.key}`"
              >
                <span class="flex h-4 w-4 shrink-0 items-center justify-center">
                  <ClientIcon
                    v-if="row.running"
                    name="loaderCircle"
                    class="cw-spin h-3 w-3 text-[color:var(--cw-accent)]"
                  />
                  <i
                    v-else-if="row.unread"
                    class="cw-unread-dot"
                    :data-test="`preview-unread-${row.key}`"
                  ></i>
                </span>
                <span
                  class="min-w-0 flex-1 truncate text-[13.5px] text-[color:var(--cw-text)]"
                  :class="row.unread ? 'font-medium' : ''"
                >{{ row.title }}</span>
                <span
                  class="shrink-0 text-[12px]"
                  :class="row.running ? 'text-[color:var(--cw-text-tertiary)]' : 'text-[color:var(--cw-text-ghost)]'"
                >{{ row.time }}</span>
              </div>
            </template>
          </div>

          <div class="mt-auto flex h-10 flex-none items-center gap-1.5 px-2.5">
            <span class="cw-icon-button"><ClientIcon name="settings" class="h-3.5 w-3.5" /></span>
            <span class="flex h-[26px] min-w-0 items-center truncate rounded-md px-2 text-[12px] text-[color:var(--cw-text-secondary)]">
              {{ t('home.clientWorkflow.sidebar.email') }}
            </span>
          </div>
        </aside>

        <!-- ===== 主栏 ===== -->
        <div class="cw-main flex min-w-0 flex-1 flex-col">
          <div class="flex h-12 flex-none items-center gap-0.5 pl-4 pr-2 sm:pl-5">
            <span class="min-w-0 truncate text-[13px] font-medium text-[color:var(--cw-text)]" data-test="preview-title">
              {{ title }}
            </span>
            <span class="flex-1"></span>
            <!-- 信息按钮：有子智能体在后台跑时带一个脉冲点 -->
            <span class="cw-icon-button relative max-sm:hidden" data-test="preview-info-button">
              <ClientIcon name="info" class="h-3.5 w-3.5" />
              <i
                v-if="frame.backgroundLive"
                class="cw-pulse absolute right-1 top-1 h-1.5 w-1.5 rounded-full bg-[color:var(--cw-accent)]"
                data-test="preview-background-dot"
              ></i>
            </span>
            <span class="cw-icon-button"><ClientIcon name="bell" class="h-3.5 w-3.5" /></span>
            <span v-for="surface in SURFACES" :key="surface" class="cw-icon-button max-md:hidden">
              <ClientIcon :name="surface" class="h-3.5 w-3.5" />
            </span>
            <!-- 有团队的任务多一个「团队」按钮，交过计划的任务多一个「计划」按钮（surface_bar.rs 的顺序） -->
            <span
              v-if="frame.panel === 'team'"
              class="cw-icon-button bg-[color:var(--cw-overlay)] text-[color:var(--cw-accent)]"
              data-test="preview-surface-team"
            >
              <ClientIcon name="users" class="h-3.5 w-3.5" />
            </span>
            <span
              v-if="frame.panel === 'plan'"
              class="cw-icon-button relative bg-[color:var(--cw-overlay)] text-[color:var(--cw-accent)]"
              data-test="preview-surface-plan"
            >
              <ClientIcon name="list" class="h-3.5 w-3.5" />
            </span>
            <template v-if="!isMac">
              <span class="cw-icon-button is-round ml-1.5 max-md:hidden"><ClientIcon name="windowMinimize" class="h-3 w-3" /></span>
              <span class="cw-icon-button is-round max-md:hidden"><ClientIcon name="windowMaximize" class="h-3 w-3" /></span>
              <span class="cw-icon-button is-round max-md:hidden"><ClientIcon name="x" class="h-3 w-3" /></span>
            </template>
          </div>

          <!-- 场景常驻、靠透明度切换，右侧面板在同一层里；窗口高度不跳 -->
          <div
            class="relative flex min-h-0 flex-1 transition-opacity duration-300 motion-reduce:transition-none"
            :class="frame.fading ? 'opacity-0' : 'opacity-100'"
          >
            <div class="relative min-w-0 flex-1">
              <AgentScene
                class="absolute inset-0 transition-opacity duration-300 motion-reduce:transition-none"
                :class="frame.scene === 'agent' ? 'opacity-100' : 'pointer-events-none opacity-0'"
                :frame="frame"
                :balance="balanceText"
                :site-name="props.siteName"
              />
              <ImageScene
                class="absolute inset-0 transition-opacity duration-300 motion-reduce:transition-none"
                :class="frame.scene === 'image' ? 'opacity-100' : 'pointer-events-none opacity-0'"
                :frame="frame"
                :balance="balanceText"
              />
            </div>
            <RightPanel :frame="frame" />
          </div>
        </div>
      </div>
    </div>

    <ChapterNav
      :chapters="chapterItems"
      :current="currentChapter"
      :progress="progress"
      :paused="interactivePaused"
      :show-progress="!prefersReducedMotion"
      :label="t('home.clientWorkflow.chapters.label')"
      :pause-label="t('home.clientWorkflow.chapters.pause')"
      :play-label="t('home.clientWorkflow.chapters.play')"
      @select="jumpTo"
      @toggle="togglePaused"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AgentScene from '@/components/home/clientPreview/AgentScene.vue'
import ChapterNav from '@/components/home/clientPreview/ChapterNav.vue'
import ClientIcon from '@/components/home/clientPreview/ClientIcon.vue'
import ImageScene from '@/components/home/clientPreview/ImageScene.vue'
import RightPanel from '@/components/home/clientPreview/RightPanel.vue'
import { preloadDemoImages } from '@/components/home/clientPreview/demoImages'
import type { ClientIconName } from '@/components/home/clientPreview/icons'
import { CHAPTERS } from '@/components/home/clientPreview/timeline'
import { usePreviewClock } from '@/components/home/clientPreview/usePreviewClock'
import '@/components/home/clientPreview/theme.css'
import { detectPreferredClientPlatform } from '@/utils/clientDownloads'

const props = withDefaults(defineProps<{
  siteName: string
}>(), {
  siteName: 'CheapRouter',
})

const { t } = useI18n()

// 客户端在 macOS 上把红绿灯放在侧栏顶部，在 Windows 上把窗口按钮放在标题栏右端
const isMac = detectPreferredClientPlatform() === 'macos'

// 标题栏右侧的四个工具面板：终端 / 文件 / 浏览器 / 审阅
const SURFACES: ClientIconName[] = ['terminal', 'folder', 'globe', 'fileDiff']

// 演示用余额：Agent 这一轮、团队、一张图各扣一次
const BALANCE = 36.52
const CHARGES = [0.18, 0.46, 0.04]

const prefersReducedMotion =
  typeof window !== 'undefined' &&
  typeof window.matchMedia === 'function' &&
  window.matchMedia('(prefers-reduced-motion: reduce)').matches

const { frame, progress, interactivePaused, currentChapter, setPaused, togglePaused, jumpTo, start, dispose } =
  usePreviewClock(prefersReducedMotion)

const root = ref<HTMLElement | null>(null)

const balanceText = computed(() => {
  const spent = CHARGES.slice(0, frame.value.charges).reduce((sum, value) => sum + value, 0)
  return `$${(BALANCE - spent).toFixed(2)}`
})

const title = computed(() => {
  if (frame.value.scene === 'image') return t('home.clientWorkflow.image.title')
  if (frame.value.chapter === 'team') return t('home.clientWorkflow.sidebar.teamTaskTitle')
  return t('home.clientWorkflow.sidebar.taskTitle')
})

const sidebarActions = computed<Array<{ key: string; icon: ClientIconName; label: string; active: boolean }>>(() => [
  { key: 'new-task', icon: 'compose', label: t('home.clientWorkflow.sidebar.newTask'), active: false },
  { key: 'search', icon: 'search', label: t('home.clientWorkflow.sidebar.search'), active: false },
  { key: 'images', icon: 'image', label: t('home.clientWorkflow.sidebar.images'), active: frame.value.imageRowActive },
  { key: 'model-status', icon: 'server', label: t('home.clientWorkflow.sidebar.modelStatus'), active: false },
])

type TaskRow =
  | { kind: 'section'; key: string; label: string; addProject?: boolean }
  | { kind: 'project'; key: string; label: string }
  | { kind: 'spacer'; key: string }
  | {
      kind: 'task'
      key: string
      title: string
      time: string
      active: boolean
      running: boolean
      unread: boolean
      nested: boolean
    }

// 按项目视图：「项目」下是这个项目和它的任务（缩进到文件夹下），「任务」下是不属于项目的任务。
// 运行中显示转圈和已运行时长，别处跑完还没看的任务带一个未读点。
const taskRows = computed<TaskRow[]>(() => {
  const current = frame.value
  const planRunning = current.chapter === 'plan' && current.sent && !current.runDone
  const teamRunning = current.team?.phase === 'running'
  const teamTouched = current.chapter !== 'plan'
  return [
    { kind: 'section', key: 'projects', label: t('home.clientWorkflow.sidebar.projects'), addProject: true },
    { kind: 'project', key: 'project', label: t('home.clientWorkflow.footer.project') },
    {
      kind: 'task',
      key: 'plan',
      title: t('home.clientWorkflow.sidebar.taskTitle'),
      active: current.activeSession === 'plan',
      running: planRunning,
      unread: false,
      nested: true,
      time: planRunning
        ? t('home.clientWorkflow.sidebar.elapsed', { seconds: current.workSeconds })
        : t('home.clientWorkflow.sidebar.justNow'),
    },
    {
      kind: 'task',
      key: 'team',
      title: t('home.clientWorkflow.sidebar.teamTaskTitle'),
      active: current.activeSession === 'team',
      running: teamRunning,
      unread: false,
      nested: true,
      time: teamTouched ? t('home.clientWorkflow.sidebar.justNow') : t('home.clientWorkflow.sidebar.teamTime'),
    },
    { kind: 'spacer', key: 'spacer' },
    { kind: 'section', key: 'tasks', label: t('home.clientWorkflow.sidebar.tasks') },
    {
      kind: 'task',
      key: 'older',
      title: t('home.clientWorkflow.sidebar.olderTask'),
      active: false,
      running: false,
      unread: true,
      nested: false,
      time: t('home.clientWorkflow.sidebar.olderTime'),
    },
  ]
})

const chapterItems = computed(() =>
  CHAPTERS.map((chapter) => ({ id: chapter.id, label: t(`home.clientWorkflow.chapters.${chapter.id}`) })),
)

// 鼠标在窗口上移动时暂停，方便看清；触屏不算。
// 只认真的移动：滚动页面时窗口滑到静止的鼠标下面，浏览器补发的事件没有位移，
// 以前按 pointerenter 暂停，演示就会一直停着，直到鼠标挪出去（比如挪到进度条上）才接着播。
function onPointerMove(event: PointerEvent) {
  if (event.pointerType === 'touch') return
  if (!event.movementX && !event.movementY) return
  setPaused('hover', true)
}

function onPointerLeave() {
  setPaused('hover', false)
}

// 页面在滚动，就不是停下来看演示：松开悬停暂停，窗口滑走后不会还停着
function onScroll() {
  setPaused('hover', false)
}

// 只有键盘聚焦（:focus-visible）才暂停；鼠标点章节按钮不该让演示停下
function onFocusIn(event: FocusEvent) {
  const target = event.target as Element | null
  let keyboard = true
  try {
    keyboard = target?.matches(':focus-visible') ?? true
  } catch {
    keyboard = true
  }
  if (keyboard) setPaused('focus', true)
}

function onFocusOut(event: FocusEvent) {
  const next = event.relatedTarget as Node | null
  if (!next || !root.value?.contains(next)) setPaused('focus', false)
}

let observer: IntersectionObserver | null = null

function onVisibilityChange() {
  setPaused('hidden', document.hidden)
}

onMounted(() => {
  preloadDemoImages()
  if (typeof document !== 'undefined') {
    setPaused('hidden', document.hidden)
    document.addEventListener('visibilitychange', onVisibilityChange)
  }
  if (typeof window !== 'undefined') window.addEventListener('scroll', onScroll, { passive: true })
  // 滚出视口就停，滚回来从停下的地方接着播
  if (typeof IntersectionObserver === 'function' && root.value) {
    observer = new IntersectionObserver((entries) => {
      const entry = entries[entries.length - 1]
      if (entry) setPaused('offscreen', !entry.isIntersecting)
    })
    observer.observe(root.value)
  }
  start()
})

onBeforeUnmount(() => {
  dispose()
  observer?.disconnect()
  observer = null
  if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', onVisibilityChange)
  if (typeof window !== 'undefined') window.removeEventListener('scroll', onScroll)
})
</script>

<style scoped>
.cw-sidebar {
  background: var(--cw-sidebar);
}

/* 主栏是容器：右侧面板按主栏宽度决定是一列还是盖在对话上 */
.cw-main {
  container-name: cw-main;
  container-type: inline-size;
  background: var(--cw-surface);
}

@media (min-width: 640px) {
  .cw-main {
    border-left: 1px solid var(--cw-sidebar-border);
  }
}

.cw-nav-row {
  display: flex;
  height: 32px;
  align-items: center;
  gap: 10px;
  border-radius: 7px;
  padding: 0 6px;
  font-size: 13px;
  color: var(--cw-text-secondary);
  transition: background-color 0.2s ease;
}

.cw-nav-row.is-active,
.cw-task.is-active {
  background: var(--cw-sidebar-item);
}

/* 尺寸照客户端：sidebar_rows.rs 的行高与缩进 */
.cw-segment {
  display: flex;
  height: 24px;
  min-width: 0;
  align-items: center;
  border: 1px solid transparent;
  border-radius: 9999px;
  padding: 0 9px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  color: var(--cw-text-tertiary);
}

.cw-segment.is-active {
  border-color: var(--cw-border);
  background: var(--cw-composer);
  color: var(--cw-text);
}

.cw-tool-button {
  display: flex;
  height: 24px;
  width: 24px;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  color: var(--cw-text-secondary);
}

.cw-section-header {
  display: flex;
  height: 28px;
  margin-bottom: 2px;
  align-items: center;
  padding: 0 4px 0 10px;
  font-size: 12.5px;
  font-weight: 500;
  color: var(--cw-text-tertiary);
}

.cw-project-row {
  display: flex;
  height: 32px;
  margin-bottom: 2px;
  align-items: center;
  border-radius: 7px;
  padding: 0 4px;
}

.cw-task {
  display: flex;
  height: 32px;
  margin-bottom: 2px;
  align-items: center;
  gap: 6px;
  border-radius: 7px;
  padding: 0 4px 0 10px;
  transition: background-color 0.2s ease;
}

/* 项目里的任务：状态位对齐文件夹，标题对齐项目名 */
.cw-task.is-nested {
  padding-left: 20px;
}

.cw-unread-dot {
  display: block;
  height: 6px;
  width: 6px;
  border-radius: 9999px;
  background: #0ea5e9;
}

:global(.dark) .cw-unread-dot {
  background: #38bdf8;
}

@media (prefers-reduced-motion: reduce) {
  .cw-nav-row,
  .cw-task {
    transition: none;
  }
}
</style>
