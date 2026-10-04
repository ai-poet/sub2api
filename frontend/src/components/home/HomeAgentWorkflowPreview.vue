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
      @pointerenter="onPointerEnter"
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

          <div class="mt-2.5 flex flex-col gap-px px-2.5">
            <div class="flex h-7 items-center rounded-md px-2 text-[13px] font-medium text-[color:var(--cw-text-secondary)]">
              <span>{{ t('home.clientWorkflow.sidebar.today') }}</span>
              <span class="ml-auto flex items-center text-[color:var(--cw-text-tertiary)]">
                <span class="flex h-5 w-5 items-center justify-center"><ClientIcon name="listFilter" class="h-3 w-3" /></span>
                <span class="flex h-5 w-5 items-center justify-center"><ClientIcon name="folderNew" class="h-3 w-3" /></span>
              </span>
            </div>

            <div
              v-for="session in sessions"
              :key="session.key"
              class="cw-session"
              :class="session.active ? 'is-active' : ''"
              :data-test="`preview-session-${session.key}`"
            >
              <div class="flex items-center gap-1.5">
                <span class="min-w-0 flex-1 truncate text-[13.5px] text-[color:var(--cw-text)]">{{ session.title }}</span>
                <ClientIcon
                  v-if="session.running"
                  name="loaderCircle"
                  class="cw-spin h-3.5 w-3.5 shrink-0 text-[color:var(--cw-accent)]"
                />
              </div>
              <div class="mt-0.5 flex items-center gap-1.5 text-[12.5px] text-[color:var(--cw-text-tertiary)]">
                <ClientIcon name="folder" class="h-3 w-3 shrink-0" />
                <span class="min-w-0 truncate">{{ t('home.clientWorkflow.footer.project') }}</span>
                <span class="ml-auto shrink-0 text-[color:var(--cw-text-ghost)]">{{ session.time }}</span>
              </div>
            </div>
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
            class="relative flex min-h-0 flex-1 transition-opacity duration-500 motion-reduce:transition-none"
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

const sessions = computed(() => {
  const current = frame.value
  const planRunning = current.chapter === 'plan' && current.sent && !current.runDone
  const teamRunning = current.team?.phase === 'running'
  const teamTouched = current.chapter !== 'plan'
  return [
    {
      key: 'plan',
      title: t('home.clientWorkflow.sidebar.taskTitle'),
      active: current.activeSession === 'plan',
      running: planRunning,
      time: planRunning
        ? t('home.clientWorkflow.sidebar.working', { seconds: current.workSeconds })
        : t('home.clientWorkflow.sidebar.justNow'),
    },
    {
      key: 'team',
      title: t('home.clientWorkflow.sidebar.teamTaskTitle'),
      active: current.activeSession === 'team',
      running: teamRunning,
      time: teamTouched ? t('home.clientWorkflow.sidebar.justNow') : t('home.clientWorkflow.sidebar.teamTime'),
    },
    {
      key: 'older',
      title: t('home.clientWorkflow.sidebar.olderTask'),
      active: false,
      running: false,
      time: t('home.clientWorkflow.sidebar.olderTime'),
    },
  ]
})

const chapterItems = computed(() =>
  CHAPTERS.map((chapter) => ({ id: chapter.id, label: t(`home.clientWorkflow.chapters.${chapter.id}`) })),
)

// 鼠标停在窗口上时暂停，方便看清；触屏的 pointerenter 不算
function onPointerEnter(event: PointerEvent) {
  if (event.pointerType === 'touch') return
  setPaused('hover', true)
}

function onPointerLeave() {
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
.cw-session.is-active {
  background: var(--cw-sidebar-item);
}

.cw-session {
  border-radius: 7px;
  padding: 7px 8px;
  transition: background-color 0.2s ease;
}

@media (prefers-reduced-motion: reduce) {
  .cw-nav-row,
  .cw-session {
    transition: none;
  }
}
</style>
