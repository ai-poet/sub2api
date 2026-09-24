<template>
  <div
    class="agent-workflow-preview"
    data-test="agent-workflow-preview"
    role="img"
    :aria-label="t('home.clientWorkflow.ariaLabel', { siteName: props.siteName })"
  >
    <!-- 按客户端真实界面画的主窗口：client/src/app/{render,sidebar,composer,image_studio_view}.rs -->
    <div
      class="client-window relative mx-auto w-full max-w-[1100px] overflow-hidden rounded-xl border border-black/10 text-left shadow-[0_24px_64px_rgba(15,17,20,0.12)] dark:border-white/10 dark:shadow-[0_24px_64px_rgba(0,0,0,0.45)]"
    >
      <div class="flex h-[520px] items-stretch sm:h-[600px]">
        <!-- ===== 侧栏 ===== -->
        <aside class="sidebar hidden w-[252px] shrink-0 flex-col sm:flex" data-test="preview-sidebar">
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
              class="sidebar-row"
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

            <div class="session" :class="scene === 'agent' ? 'is-active' : ''">
              <div class="flex items-center gap-1.5">
                <span class="min-w-0 flex-1 truncate text-[13.5px] text-[color:var(--cw-text)]">
                  {{ t('home.clientWorkflow.sidebar.taskTitle') }}
                </span>
                <ClientIcon
                  v-if="!frame.runDone"
                  name="loaderCircle"
                  class="cw-spin h-3.5 w-3.5 shrink-0 text-[color:var(--cw-accent)]"
                />
              </div>
              <div class="mt-0.5 flex items-center gap-1.5 text-[12.5px] text-[color:var(--cw-text-tertiary)]">
                <ClientIcon name="folder" class="h-3 w-3 shrink-0" />
                <span class="min-w-0 truncate">{{ t('home.clientWorkflow.footer.project') }}</span>
                <span class="ml-auto shrink-0 text-[color:var(--cw-text-ghost)]">
                  {{ frame.runDone
                    ? t('home.clientWorkflow.sidebar.justNow')
                    : t('home.clientWorkflow.sidebar.working', { seconds: frame.workSeconds }) }}
                </span>
              </div>
            </div>

            <div class="session">
              <div class="truncate text-[13.5px] text-[color:var(--cw-text)]">{{ t('home.clientWorkflow.sidebar.olderTask') }}</div>
              <div class="mt-0.5 flex items-center gap-1.5 text-[12.5px] text-[color:var(--cw-text-tertiary)]">
                <ClientIcon name="folder" class="h-3 w-3 shrink-0" />
                <span class="min-w-0 truncate">{{ t('home.clientWorkflow.footer.project') }}</span>
                <span class="ml-auto shrink-0 text-[color:var(--cw-text-ghost)]">{{ t('home.clientWorkflow.sidebar.olderTime') }}</span>
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
        <div class="main flex min-w-0 flex-1 flex-col">
          <div class="flex h-12 flex-none items-center gap-0.5 pl-4 pr-2 sm:pl-5">
            <span class="min-w-0 truncate text-[13px] font-medium text-[color:var(--cw-text)]">
              {{ scene === 'image' ? t('home.clientWorkflow.image.title') : t('home.clientWorkflow.sidebar.taskTitle') }}
            </span>
            <span class="flex-1"></span>
            <span class="cw-icon-button max-sm:hidden"><ClientIcon name="info" class="h-3.5 w-3.5" /></span>
            <span class="cw-icon-button"><ClientIcon name="bell" class="h-3.5 w-3.5" /></span>
            <span v-for="surface in SURFACES" :key="surface" class="cw-icon-button max-md:hidden">
              <ClientIcon :name="surface" class="h-3.5 w-3.5" />
            </span>
            <template v-if="!isMac">
              <span class="cw-icon-button is-round ml-1.5 max-md:hidden"><ClientIcon name="windowMinimize" class="h-3 w-3" /></span>
              <span class="cw-icon-button is-round max-md:hidden"><ClientIcon name="windowMaximize" class="h-3 w-3" /></span>
              <span class="cw-icon-button is-round max-md:hidden"><ClientIcon name="x" class="h-3 w-3" /></span>
            </template>
          </div>

          <!-- 两个场景常驻，靠透明度切换，窗口高度不跳 -->
          <div
            class="relative min-h-0 flex-1 transition-opacity duration-500"
            :class="frame.fading ? 'opacity-0' : 'opacity-100'"
          >
            <AgentScene
              class="absolute inset-0 transition-opacity duration-300"
              :class="scene === 'agent' ? 'opacity-100' : 'pointer-events-none opacity-0'"
              :frame="frame"
              :balance="balanceText"
              :site-name="props.siteName"
            />
            <ImageScene
              class="absolute inset-0 transition-opacity duration-300"
              :class="scene === 'image' ? 'opacity-100' : 'pointer-events-none opacity-0'"
              :frame="frame"
              :balance="balanceText"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AgentScene from '@/components/home/clientPreview/AgentScene.vue'
import ClientIcon from '@/components/home/clientPreview/ClientIcon.vue'
import ImageScene from '@/components/home/clientPreview/ImageScene.vue'
import type { ClientIconName } from '@/components/home/clientPreview/icons'
import { CYCLE_MS, STATIC_T, deriveFrame, type PreviewFrame } from '@/components/home/clientPreview/timeline'
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

// 演示用余额：Agent 这一轮扣一次，出一张图再扣一次
const BALANCE = 36.52
const CHARGES = [0.18, 0.04]

const prefersReducedMotion =
  typeof window !== 'undefined' &&
  typeof window.matchMedia === 'function' &&
  window.matchMedia('(prefers-reduced-motion: reduce)').matches

const frame = ref<PreviewFrame>(deriveFrame(prefersReducedMotion ? STATIC_T : 0))
const scene = computed(() => frame.value.scene)

const balanceText = computed(() => {
  const spent = CHARGES.slice(0, frame.value.charges).reduce((sum, value) => sum + value, 0)
  return `$${(BALANCE - spent).toFixed(2)}`
})

const sidebarActions = computed<Array<{ key: string; icon: ClientIconName; label: string; active: boolean }>>(() => [
  { key: 'new-task', icon: 'compose', label: t('home.clientWorkflow.sidebar.newTask'), active: false },
  { key: 'search', icon: 'search', label: t('home.clientWorkflow.sidebar.search'), active: false },
  { key: 'images', icon: 'image', label: t('home.clientWorkflow.sidebar.images'), active: frame.value.imageRowActive },
])

let rafId = 0
let startStamp = 0
let lastTick = -1000

function loop(now: number) {
  if (startStamp === 0) startStamp = now
  const tMs = (now - startStamp) % CYCLE_MS
  // 100ms 粒度足够驱动所有状态切换，避免每帧重建 frame 对象
  if (Math.abs(tMs - lastTick) >= 100 || tMs < lastTick) {
    lastTick = tMs
    frame.value = deriveFrame(tMs)
  }
  rafId = requestAnimationFrame(loop)
}

onMounted(() => {
  if (prefersReducedMotion) return
  if (typeof requestAnimationFrame !== 'function') return
  rafId = requestAnimationFrame(loop)
})

onBeforeUnmount(() => {
  if (rafId) cancelAnimationFrame(rafId)
})
</script>

<style scoped>
.sidebar {
  background: var(--cw-sidebar);
}

.main {
  background: var(--cw-surface);
}

@media (min-width: 640px) {
  .main {
    border-left: 1px solid var(--cw-sidebar-border);
  }
}

.sidebar-row {
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

.sidebar-row.is-active,
.session.is-active {
  background: var(--cw-sidebar-item);
}

.session {
  border-radius: 7px;
  padding: 7px 8px;
  transition: background-color 0.2s ease;
}
</style>
