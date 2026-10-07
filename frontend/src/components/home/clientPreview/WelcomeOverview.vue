<template>
  <!-- 新任务的空白页：品牌标、「想在 X 中构建什么？」和活动概览（client/src/app/{sidebar,home_overview}.rs） -->
  <div class="flex h-full flex-col items-center justify-center pb-4" data-test="preview-welcome">
    <HomeBrandMark :site-name="siteName" class="h-[26px] w-[26px] rounded-[6px] text-[13px]" />
    <!-- 行内排版：各语言自带的空格（「想在 X 中构建什么？」/「What should we build in X?」）原样保留 -->
    <div class="mt-2.5 whitespace-pre text-center text-[18px] font-medium text-[color:var(--cw-text)] sm:text-[20px]">
      <span>{{ t('home.clientWorkflow.overview.headlineLead') }}</span>
      <span class="inline-flex items-center gap-0.5">{{ t('home.clientWorkflow.footer.project') }}<ClientIcon name="chevronDown" class="h-3.5 w-3.5 text-[color:var(--cw-text-tertiary)]" /></span>
      <span>{{ t('home.clientWorkflow.overview.headlineTail') }}</span>
    </div>

    <div class="cw-overview mt-4 w-[520px] max-w-full" data-test="preview-overview">
      <div class="flex items-center justify-between">
        <span class="flex items-center gap-0.5">
          <span class="cw-pill is-selected">{{ t('home.clientWorkflow.overview.tabOverview') }}</span>
          <span class="cw-pill">{{ t('home.clientWorkflow.overview.tabModels') }}</span>
        </span>
        <span class="flex items-center gap-0.5">
          <span class="cw-pill is-selected">{{ t('home.clientWorkflow.overview.rangeAll') }}</span>
          <span class="cw-pill">{{ t('home.clientWorkflow.overview.range30d') }}</span>
          <span class="cw-pill">{{ t('home.clientWorkflow.overview.range7d') }}</span>
        </span>
      </div>

      <div class="grid grid-cols-3 gap-1.5">
        <div v-for="tile in tiles" :key="tile.key" class="cw-tile">
          <div class="truncate text-[11.5px] leading-[15px] text-[color:var(--cw-text-tertiary)]">{{ tile.label }}</div>
          <div
            class="mt-0.5 truncate text-[13.5px] leading-[18px] text-[color:var(--cw-text)]"
            :class="tile.strong ? 'font-semibold' : 'font-medium'"
          >{{ tile.value }}</div>
        </div>
      </div>

      <div class="cw-heat flex flex-col gap-1.5">
        <div class="flex justify-center overflow-hidden" data-test="preview-heat-grid">
          <div class="flex gap-[2px]">
            <div v-for="(week, column) in HEAT_WEEKS" :key="column" class="flex flex-col gap-[2px]">
              <i
                v-for="(level, row) in week"
                :key="row"
                class="cw-heat-cell"
                :class="level === null ? 'is-future' : `is-level-${level}`"
              ></i>
            </div>
          </div>
        </div>
        <div class="flex items-center justify-between text-[11.5px] text-[color:var(--cw-text-tertiary)]">
          <span class="truncate">{{ t('home.clientWorkflow.overview.heatCaption') }}</span>
          <span class="flex shrink-0 items-center gap-[3px]">
            <span class="mr-[3px]">{{ t('home.clientWorkflow.overview.less') }}</span>
            <i v-for="level in 5" :key="level" class="cw-heat-cell is-legend" :class="`is-level-${level - 1}`"></i>
            <span class="ml-[3px]">{{ t('home.clientWorkflow.overview.more') }}</span>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeBrandMark from '@/components/home/HomeBrandMark.vue'
import ClientIcon from './ClientIcon.vue'

defineProps<{
  siteName: string
}>()

const { t } = useI18n()

// 演示用的半年活跃度：26 周 × 7 天，越近越忙，最后一周只到今天（同客户端的 26 周热力格）
const WEEKS = 26
const TODAY_WEEKDAY = 3

function buildHeatWeeks(): Array<Array<number | null>> {
  let seed = 7
  const next = () => {
    seed = (seed * 1103515245 + 12345) % 2147483648
    return seed / 2147483648
  }
  return Array.from({ length: WEEKS }, (_, week) =>
    Array.from({ length: 7 }, (_, day) => {
      if (week === WEEKS - 1 && day > TODAY_WEEKDAY) return null
      const busy = 0.25 + (0.65 * week) / (WEEKS - 1)
      const roll = next()
      if (roll > busy + 0.15) return 0
      return Math.min(4, 1 + Math.floor(next() * 4 * busy + (day < 5 ? 0.6 : 0)))
    }),
  )
}

const HEAT_WEEKS = buildHeatWeeks()

const tiles = computed(() => [
  { key: 'sessions', label: t('home.clientWorkflow.overview.sessions'), value: '128', strong: true },
  { key: 'messages', label: t('home.clientWorkflow.overview.messages'), value: '3,642', strong: true },
  { key: 'activeDays', label: t('home.clientWorkflow.overview.activeDays'), value: '57', strong: true },
  { key: 'peakHour', label: t('home.clientWorkflow.overview.peakHour'), value: t('home.clientWorkflow.overview.peakHourValue'), strong: true },
  { key: 'favoriteModel', label: t('home.clientWorkflow.overview.favoriteModel'), value: 'Claude Sonnet 5.5', strong: false },
  { key: 'longestStreak', label: t('home.clientWorkflow.overview.longestStreak'), value: t('home.clientWorkflow.overview.streakValue'), strong: true },
])
</script>

<style scoped>
.cw-overview {
  display: flex;
  flex-direction: column;
  gap: 10px;
  border: 1px solid var(--cw-border);
  border-radius: 12px;
  background: var(--cw-surface);
  padding: 10px;
}

.cw-pill {
  display: flex;
  height: 22px;
  align-items: center;
  border-radius: 6px;
  padding: 0 8px;
  font-size: 12px;
  color: var(--cw-text-tertiary);
}

.cw-pill.is-selected {
  background: var(--cw-overlay-strong);
  color: var(--cw-text);
}

.cw-tile {
  min-width: 0;
  border-radius: 8px;
  background: var(--cw-overlay);
  padding: 5px 10px;
}

/* 热力格：0 级是正文色 7%，1–4 级是用量色由浅到深（home_overview.rs 的 heat_color） */
.cw-heat-cell {
  display: block;
  height: 9px;
  width: 9px;
  border-radius: 2px;
}

.cw-heat-cell.is-legend {
  height: 8px;
  width: 8px;
}

.cw-heat-cell.is-level-0 {
  background: color-mix(in srgb, var(--cw-text) 7%, transparent);
}

.cw-heat-cell.is-level-1 {
  background: color-mix(in srgb, var(--cw-gauge) 30%, transparent);
}

.cw-heat-cell.is-level-2 {
  background: color-mix(in srgb, var(--cw-gauge) 50%, transparent);
}

.cw-heat-cell.is-level-3 {
  background: color-mix(in srgb, var(--cw-gauge) 75%, transparent);
}

.cw-heat-cell.is-level-4 {
  background: var(--cw-gauge);
}

/* 主栏太窄时（手机）只留数字，热力格放不下 */
@container cw-main (max-width: 420px) {
  .cw-heat {
    display: none;
  }
}
</style>
