<template>
  <div
    class="agent-workflow-preview"
    data-test="agent-workflow-preview"
    role="img"
    :aria-label="t('home.clientWorkflow.ariaLabel')"
  >
    <div
      class="relative mx-auto w-full max-w-[1100px] overflow-hidden rounded-xl border border-gray-200 bg-[#f7f6f4] shadow-[0_24px_64px_rgba(15,17,20,0.12)] dark:border-white/10 dark:bg-[#1a1c1f] dark:shadow-[0_24px_64px_rgba(0,0,0,0.45)]"
    >
      <div class="flex items-stretch">
        <!-- ===== LEFT SIDEBAR ===== -->
        <aside
          class="hidden w-[230px] shrink-0 flex-col border-r border-black/[0.06] bg-[#efede9] sm:flex dark:border-white/[0.06] dark:bg-[#151719]"
        >
          <!-- Window controls row -->
          <div class="flex items-center gap-1 px-3 pt-3 text-gray-400 dark:text-white/30">
            <Glyph :d="glyphs.panelCollapse" class="h-4 w-4" />
            <span class="flex-1"></span>
            <Icon name="chevronLeft" size="sm" />
            <Icon name="chevronRight" size="sm" class="opacity-40" />
          </div>

          <!-- Primary nav -->
          <nav class="mt-2 space-y-0.5 px-2">
            <div class="flex items-center gap-2 rounded-lg px-2 py-1.5 text-[13px] text-gray-700 dark:text-white/70">
              <Icon name="edit" size="sm" class="text-gray-500 dark:text-white/40" />
              <span>{{ t('home.clientWorkflow.sidebar.newTask') }}</span>
            </div>
            <div class="flex items-center gap-2 rounded-lg px-2 py-1.5 text-[13px] text-gray-700 dark:text-white/70">
              <Icon name="search" size="sm" class="text-gray-500 dark:text-white/40" />
              <span>{{ t('home.clientWorkflow.sidebar.search') }}</span>
            </div>
          </nav>

          <!-- Section: today -->
          <div class="mt-4 flex items-center px-4 text-[11px] font-medium text-gray-400 dark:text-white/30">
            <span>{{ t('home.clientWorkflow.sidebar.today') }}</span>
            <span class="ml-auto flex items-center gap-1.5">
              <Icon name="arrowsUpDown" size="xs" />
              <Icon name="more" size="xs" />
            </span>
          </div>

          <!-- Active task card -->
          <div class="mx-2 mt-1 rounded-lg bg-black/[0.06] px-3 py-2 dark:bg-white/[0.07]">
            <div class="flex items-center justify-between gap-2">
              <span class="truncate text-[13px] font-medium text-gray-900 dark:text-white">
                {{ t('home.clientWorkflow.sidebar.taskTitle') }}
              </span>
              <span
                class="h-3.5 w-3.5 shrink-0 rounded-full border-2 border-orange-500/25 border-t-orange-500 motion-safe:animate-spin"
                aria-hidden="true"
              ></span>
            </div>
            <div class="mt-1 flex items-center gap-1.5 text-[11px] text-gray-500 dark:text-white/40">
              <Glyph :d="glyphs.folder" class="h-3 w-3 shrink-0" />
              <span class="truncate">{{ t('home.clientWorkflow.sidebar.project') }}</span>
              <span class="shrink-0">{{ t('home.clientWorkflow.working', { seconds: frame.seconds }) }}</span>
            </div>
          </div>

          <!-- Footer: account -->
          <div
            class="mt-auto flex items-center gap-2 border-t border-black/[0.06] px-3 py-2.5 text-[12px] text-gray-500 dark:border-white/[0.06] dark:text-white/40"
          >
            <Icon name="cog" size="sm" />
            <span class="truncate">{{ t('home.clientWorkflow.sidebar.email') }}</span>
          </div>
        </aside>

        <!-- ===== MAIN AREA ===== -->
        <div class="flex min-w-0 flex-1 flex-col">
          <!-- Titlebar -->
          <div class="flex items-center justify-between px-4 py-2.5 sm:px-5">
            <span class="text-[13px] font-semibold text-gray-900 dark:text-white">
              {{ t('home.clientWorkflow.sidebar.taskTitle') }}
            </span>
            <Icon name="infoCircle" size="sm" class="text-gray-400 dark:text-white/30" />
          </div>

          <!-- Transcript: tool rows -->
          <div class="flex-1 space-y-1.5 px-4 sm:px-5">
            <div
              v-for="(row, i) in rows"
              :key="i"
              class="flex items-center gap-2 rounded-lg border border-black/[0.05] bg-white px-3 py-[5px] transition-all duration-300 ease-out dark:border-white/[0.06] dark:bg-white/5"
              :class="i < frame.visibleRows ? 'translate-y-0 opacity-100' : 'translate-y-2 opacity-0'"
            >
              <Glyph :d="row.icon" class="h-3.5 w-3.5 shrink-0 text-gray-400 dark:text-white/35" />
              <span class="shrink-0 text-[12px] font-semibold text-gray-800 dark:text-white/80">{{ row.label }}</span>
              <span class="truncate text-[12px] text-gray-500 dark:text-white/45">{{ row.text }}</span>
              <Icon name="chevronRight" size="xs" class="ml-auto shrink-0 text-gray-300 dark:text-white/20" />
            </div>
          </div>

          <!-- Working line -->
          <div class="flex items-center gap-2 px-5 py-2 text-[12px] text-gray-400 dark:text-white/35 sm:px-6">
            <span class="preview-dots" aria-hidden="true"><i></i><i></i><i></i></span>
            <span>{{ t('home.clientWorkflow.working', { seconds: frame.seconds }) }}</span>
          </div>

          <!-- Composer -->
          <div
            class="mx-4 rounded-xl border border-black/[0.08] bg-white p-3 shadow-sm sm:mx-5 dark:border-white/10 dark:bg-white/[0.04]"
          >
            <div class="px-1 text-[13px] text-gray-400 dark:text-white/30">
              {{ t('home.clientWorkflow.composer.placeholder') }}
            </div>
            <div class="mt-3 flex items-center gap-1 text-[11px] text-gray-500 dark:text-white/45">
              <span class="flex items-center gap-1.5 rounded-full px-2 py-1">
                <Glyph :d="glyphs.provider" class="h-3 w-3" />
                <span class="font-medium text-gray-700 dark:text-white/70">{{ t('home.clientWorkflow.composer.model') }}</span>
                <Icon name="chevronDown" size="xs" class="text-gray-300 dark:text-white/20" />
              </span>
              <span class="rounded-full px-2 py-1">{{ t('home.clientWorkflow.composer.effort') }}</span>
              <span class="flex items-center gap-1 rounded-full px-2 py-1">
                <Icon name="lock" size="xs" />
                <span>{{ t('home.clientWorkflow.composer.access') }}</span>
              </span>
              <span class="hidden items-center gap-1 rounded-full px-2 py-1 sm:flex">
                <Glyph :d="glyphs.wrench" class="h-3 w-3" />
                <span>{{ t('home.clientWorkflow.composer.build') }}</span>
              </span>
              <span
                class="ml-auto flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gray-900 dark:bg-white"
                :title="t('home.clientWorkflow.composer.stop')"
              >
                <span class="h-2.5 w-2.5 rounded-[2px] bg-white dark:bg-gray-900"></span>
              </span>
            </div>
          </div>

          <!-- Status bar -->
          <div
            class="mt-3 flex items-center gap-3 border-t border-black/[0.05] px-4 py-2 text-[11px] text-gray-500 sm:px-5 dark:border-white/[0.06] dark:text-white/40"
          >
            <span class="flex min-w-0 items-center gap-1">
              <Glyph :d="glyphs.folder" class="h-3 w-3 shrink-0" />
              <span class="truncate">{{ t('home.clientWorkflow.statusBar.project') }}</span>
            </span>
            <span class="flex shrink-0 items-center gap-1">
              <Glyph :d="glyphs.display" class="h-3 w-3" />
              <span>{{ t('home.clientWorkflow.statusBar.local') }}</span>
            </span>
            <span class="flex shrink-0 items-center gap-1">
              <Glyph :d="glyphs.branch" class="h-3 w-3" />
              <span>{{ t('home.clientWorkflow.statusBar.branch') }}</span>
            </span>
            <span
              class="ml-auto flex shrink-0 items-center gap-1 tabular-nums transition-colors duration-300"
              :class="frame.balanceUpdated ? 'text-emerald-600 dark:text-emerald-400' : ''"
            >
              <Glyph :d="glyphs.wallet" class="h-3 w-3" />
              <span>{{ balanceText }}</span>
            </span>
          </div>
        </div>
      </div>

      <!-- ===== ACCOUNT MENU OVERLAY ===== -->
      <div
        class="absolute bottom-11 left-2 z-20 w-[236px] origin-bottom-left rounded-xl border border-black/10 bg-white p-1.5 shadow-xl transition-all duration-200 ease-out dark:border-white/10 dark:bg-[#232629]"
        :class="frame.menuOpen ? 'scale-100 opacity-100' : 'pointer-events-none scale-95 opacity-0'"
        data-test="preview-account-menu"
      >
        <div class="px-2.5 pb-1 pt-2 text-[12px] font-semibold text-gray-900 dark:text-white">
          {{ t('home.clientWorkflow.menu.balance', { amount: balanceText }) }}
        </div>
        <div class="rounded-lg px-2.5 py-1.5 text-[12px] text-gray-700 dark:text-white/70">
          {{ t('home.clientWorkflow.menu.topUp') }}
        </div>
        <div class="my-1 h-px bg-black/5 dark:bg-white/10"></div>
        <div class="flex items-center rounded-lg px-2.5 py-1.5 text-[12px] text-gray-700 dark:text-white/70">
          <span>{{ t('home.clientWorkflow.menu.claudeGroup') }}</span>
          <span class="ml-auto flex items-center gap-0.5 text-gray-400 dark:text-white/35">
            {{ t('home.clientWorkflow.menu.claudeValue') }}
            <Icon name="chevronRight" size="xs" />
          </span>
        </div>
        <div
          class="flex items-center rounded-lg bg-black/[0.05] px-2.5 py-1.5 text-[12px] text-gray-900 dark:bg-white/[0.08] dark:text-white"
        >
          <span>{{ t('home.clientWorkflow.menu.codexGroup') }}</span>
          <span class="ml-auto flex items-center gap-0.5 text-gray-500 dark:text-white/45">
            {{ frame.saleSelected ? t('home.clientWorkflow.menu.codexValueAfter') : t('home.clientWorkflow.menu.codexValueBefore') }}
            <Icon name="chevronRight" size="xs" />
          </span>
        </div>
        <div class="flex items-center rounded-lg px-2.5 py-1.5 text-[12px] text-gray-700 dark:text-white/70">
          <span>{{ t('home.clientWorkflow.menu.grokGroup') }}</span>
          <span class="ml-auto flex items-center gap-0.5 text-gray-400 dark:text-white/35">
            {{ t('home.clientWorkflow.menu.grokValue') }}
            <Icon name="chevronRight" size="xs" />
          </span>
        </div>
        <div class="my-1 h-px bg-black/5 dark:bg-white/10"></div>
        <div class="rounded-lg px-2.5 py-1.5 text-[12px] text-gray-700 dark:text-white/70">
          {{ t('home.clientWorkflow.menu.modelPlaza') }}
        </div>
        <div class="rounded-lg px-2.5 py-1.5 text-[12px] text-gray-700 dark:text-white/70">
          {{ t('home.clientWorkflow.menu.usage') }}
        </div>
        <div class="my-1 h-px bg-black/5 dark:bg-white/10"></div>
        <div class="rounded-lg px-2.5 py-1.5 text-[12px] text-gray-700 dark:text-white/70">
          {{ t('home.clientWorkflow.menu.logout') }}
        </div>
      </div>

      <!-- ===== SECOND-LEVEL GROUP SUBMENU ===== -->
      <div
        class="absolute bottom-[10rem] left-[244px] z-30 w-[236px] origin-left rounded-xl border border-black/10 bg-white p-1.5 shadow-xl transition-all duration-200 ease-out max-sm:left-auto max-sm:right-2 dark:border-white/10 dark:bg-[#232629]"
        :class="frame.submenuOpen ? 'translate-x-0 opacity-100' : 'pointer-events-none -translate-x-1 opacity-0'"
        data-test="preview-group-submenu"
      >
        <div class="rounded-lg px-2.5 py-1.5 text-[12px] text-gray-700 dark:text-white/70">
          {{ t('home.clientWorkflow.menu.submenu.default') }}
        </div>
        <div class="flex items-center rounded-lg px-2.5 py-1.5 text-[12px] text-gray-700 dark:text-white/70">
          <span>{{ t('home.clientWorkflow.menu.submenu.codexName') }}</span>
          <span class="ml-1.5 text-[10px] text-gray-400 dark:text-white/30">{{ t('home.clientWorkflow.menu.submenu.codexMeta') }}</span>
          <Icon v-if="!frame.saleSelected" name="check" size="xs" class="ml-auto text-gray-700 dark:text-white/80" />
        </div>
        <div
          class="flex items-center rounded-lg px-2.5 py-1.5 text-[12px] text-gray-900 dark:text-white"
          :class="frame.saleFlash ? 'preview-flash' : frame.saleSelected ? 'bg-black/[0.05] dark:bg-white/[0.08]' : ''"
          data-test="preview-sale-row"
        >
          <span>{{ t('home.clientWorkflow.menu.submenu.saleName') }}</span>
          <span class="ml-1.5 text-[10px] text-gray-400 dark:text-white/30">{{ t('home.clientWorkflow.menu.submenu.saleMeta') }}</span>
          <Icon v-if="frame.saleSelected" name="check" size="xs" class="ml-auto text-gray-900 dark:text-white" />
        </div>
        <div class="flex items-center rounded-lg px-2.5 py-1.5 text-[12px] text-gray-700 dark:text-white/70">
          <span>{{ t('home.clientWorkflow.menu.submenu.welfareName') }}</span>
          <span class="ml-1.5 text-[10px] text-gray-400 dark:text-white/30">{{ t('home.clientWorkflow.menu.submenu.welfareMeta') }}</span>
        </div>
      </div>

      <!-- ===== TOAST ===== -->
      <div
        class="absolute bottom-14 left-1/2 z-40 -translate-x-1/2 whitespace-nowrap rounded-full bg-gray-900 px-4 py-2 text-[12px] text-white shadow-lg transition-all duration-300 ease-out max-sm:whitespace-normal dark:bg-white dark:text-gray-900"
        :class="frame.toastVisible ? 'translate-y-0 opacity-100' : 'pointer-events-none translate-y-2 opacity-0'"
        data-test="preview-toast"
      >
        {{ t('home.clientWorkflow.toast') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

/* ============ Inline SVG glyphs (icons Icon.vue does not provide) ============ */

const Glyph = Object.assign(
  (props: { d: string }) =>
    h(
      'svg',
      {
        viewBox: '0 0 24 24',
        fill: 'none',
        stroke: 'currentColor',
        'stroke-width': 1.5,
        'aria-hidden': 'true',
      },
      [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: props.d })],
    ),
  { props: ['d'] },
)

const glyphs = {
  folder:
    'M2.25 12.75V6.75a2.25 2.25 0 012.25-2.25h4.19c.6 0 1.17.24 1.59.66l1.16 1.18c.42.42.99.66 1.59.66h5.72a2.25 2.25 0 012.25 2.25v7.5a2.25 2.25 0 01-2.25 2.25H4.5a2.25 2.25 0 01-2.25-2.25v-4.5z',
  wrench:
    'M21.75 6.75a4.5 4.5 0 01-4.884 4.484c-1.076-.091-2.264.071-2.95.904l-7.152 8.684a2.548 2.548 0 11-3.586-3.586l8.684-7.152c.833-.686.995-1.874.904-2.95a4.5 4.5 0 016.336-4.486l-3.276 3.276a3.004 3.004 0 002.25 2.25l3.276-3.276c.256.565.398 1.192.398 1.852z',
  file:
    'M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z',
  asterisk: 'M12 4.5v15M5.5 8.25l13 7.5M18.5 8.25l-13 7.5',
  panelCollapse:
    'M4.5 4.5h15A1.5 1.5 0 0121 6v12a1.5 1.5 0 01-1.5 1.5h-15A1.5 1.5 0 013 18V6a1.5 1.5 0 011.5-1.5zM9.75 4.5v15',
  display:
    'M9 17.25v1.007a3 3 0 01-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0115 18.257V17.25m6-12V15a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 15V5.25m18 0A2.25 2.25 0 0018.75 3H5.25A2.25 2.25 0 003 5.25',
  branch:
    'M6 5.25a2.25 2.25 0 104.5 0 2.25 2.25 0 00-4.5 0zm2.25 2.25v9m0 0a2.25 2.25 0 102.25 2.25M8.25 16.5a2.25 2.25 0 012.25 2.25m5.25-13.5a2.25 2.25 0 104.5 0 2.25 2.25 0 00-4.5 0zm2.25 2.25a6 6 0 01-6 6',
  wallet:
    'M21 12a2.25 2.25 0 00-2.25-2.25H15a3 3 0 11-6 0H5.25A2.25 2.25 0 003 12m18 0v6a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 18v-6m18 0V9a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 9v3',
  provider: 'M18.75 5.25L5.25 18.75M18.75 12.75l-6 6',
}

/* ============ Transcript rows ============ */

type RowKind = 'read' | 'tool' | 'think'

const ROW_DEFS: Array<{ kind: RowKind; key: string }> = [
  { kind: 'read', key: 'r1' },
  { kind: 'tool', key: 'r2' },
  { kind: 'read', key: 'r3' },
  { kind: 'think', key: 'r4' },
  { kind: 'read', key: 'r5' },
  { kind: 'read', key: 'r6' },
  { kind: 'read', key: 'r7' },
  { kind: 'read', key: 'r8' },
  { kind: 'read', key: 'r9' },
  { kind: 'tool', key: 'r10' },
  { kind: 'read', key: 'r11' },
  { kind: 'read', key: 'r12' },
]

const KIND_LABEL_KEY: Record<RowKind, string> = {
  read: 'home.clientWorkflow.labels.read',
  tool: 'home.clientWorkflow.labels.tool',
  think: 'home.clientWorkflow.labels.thinking',
}

const KIND_GLYPH: Record<RowKind, string> = {
  read: glyphs.file,
  tool: glyphs.wrench,
  think: glyphs.asterisk,
}

const rows = computed(() =>
  ROW_DEFS.map((def) => ({
    label: t(KIND_LABEL_KEY[def.kind]),
    text: t(`home.clientWorkflow.rows.${def.key}`),
    icon: KIND_GLYPH[def.kind],
  })),
)

/* ============ Animation timeline ============
 * ~12s loop, GPU-friendly (translate/opacity only):
 * 1. tool rows fade in staggered while the spinner spins and seconds tick
 * 2. pause
 * 3. account menu pops in, submenu slides out, the check moves to Codex Sale
 * 4. menu closes, toast confirms the switch, status-bar balance updates
 * 5. fade back to start
 * prefers-reduced-motion: renders the static "hero" frame instead.
 */

const CYCLE_MS = 12000
const ROW_COUNT = ROW_DEFS.length
const ROW_START = 500
const ROW_STAGGER = 250
const MENU_AT = 4800
const SUBMENU_AT = 5500
const SELECT_AT = 6600
const FLASH_UNTIL = 7400
const MENU_CLOSE = 7800
const TOAST_AT = 8100
const BALANCE_AT = 8300
const TOAST_UNTIL = 10900
const RESET_AT = 11400
// 静态帧：菜单 + 二级分组面板展开、已切到 Codex Sale（reduced motion 使用）
const STATIC_T = 7500

interface PreviewFrame {
  visibleRows: number
  seconds: number
  menuOpen: boolean
  submenuOpen: boolean
  saleSelected: boolean
  saleFlash: boolean
  toastVisible: boolean
  balanceUpdated: boolean
}

function deriveFrame(tMs: number): PreviewFrame {
  const resetting = tMs >= RESET_AT
  return {
    visibleRows:
      resetting || tMs < ROW_START
        ? 0
        : Math.min(ROW_COUNT, Math.floor((tMs - ROW_START) / ROW_STAGGER) + 1),
    seconds: 47 + Math.floor(tMs / 1000),
    menuOpen: tMs >= MENU_AT && tMs < MENU_CLOSE,
    submenuOpen: tMs >= SUBMENU_AT && tMs < MENU_CLOSE,
    saleSelected: tMs >= SELECT_AT && !resetting,
    saleFlash: tMs >= SELECT_AT && tMs < FLASH_UNTIL,
    toastVisible: tMs >= TOAST_AT && tMs < TOAST_UNTIL,
    balanceUpdated: tMs >= BALANCE_AT && !resetting,
  }
}

const prefersReducedMotion =
  typeof window !== 'undefined' &&
  typeof window.matchMedia === 'function' &&
  window.matchMedia('(prefers-reduced-motion: reduce)').matches

const frame = ref<PreviewFrame>(deriveFrame(prefersReducedMotion ? STATIC_T : 0))

const balanceText = computed(() =>
  frame.value.balanceUpdated
    ? t('home.clientWorkflow.balanceAfter')
    : t('home.clientWorkflow.balanceBefore'),
)

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
.preview-dots {
  display: inline-flex;
  gap: 3px;
  align-items: center;
}

.preview-dots i {
  width: 3px;
  height: 3px;
  border-radius: 9999px;
  background: currentColor;
  opacity: 0.3;
}

@media (prefers-reduced-motion: no-preference) {
  .preview-dots i {
    animation: preview-dot-pulse 1.2s ease-in-out infinite;
  }

  .preview-dots i:nth-child(2) {
    animation-delay: 0.2s;
  }

  .preview-dots i:nth-child(3) {
    animation-delay: 0.4s;
  }
}

@keyframes preview-dot-pulse {
  0%,
  80%,
  100% {
    opacity: 0.25;
  }
  40% {
    opacity: 0.9;
  }
}

.preview-flash {
  animation: preview-row-flash 0.8s ease-out 1;
  background-color: rgba(0, 0, 0, 0.05);
}

:global(.dark) .preview-flash {
  background-color: rgba(255, 255, 255, 0.08);
}

@keyframes preview-row-flash {
  0% {
    background-color: rgba(249, 115, 22, 0.22);
  }
  100% {
    background-color: rgba(0, 0, 0, 0.05);
  }
}
</style>
