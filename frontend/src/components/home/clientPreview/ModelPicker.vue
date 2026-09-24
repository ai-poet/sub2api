<template>
  <!-- 客户端输入框上方的模型选择器：左栏是各家 Agent，右边是当前 Agent 的模型 -->
  <div
    class="cw-picker absolute bottom-full left-0 z-20 mb-2 flex h-[300px] w-full max-w-[560px] origin-bottom-left overflow-hidden rounded-[13px] shadow-[0_18px_48px_rgba(0,0,0,0.18)] transition-all duration-200 ease-out sm:h-[360px]"
    :class="open ? 'scale-100 opacity-100' : 'pointer-events-none scale-[0.97] opacity-0'"
    data-test="preview-model-picker"
  >
    <!-- 左栏：收藏 + 各家 Agent，只显示图标 -->
    <div class="cw-rail relative flex w-[46px] shrink-0 flex-col items-center gap-px overflow-hidden py-1.5">
      <span class="cw-rail-item text-[color:var(--cw-text-tertiary)]">
        <ClientIcon name="star" class="h-4 w-4" />
      </span>
      <span class="my-1 h-px w-6 bg-[color:var(--cw-border-strong)]"></span>
      <span
        v-for="(agent, index) in agents"
        :key="agent.id"
        class="cw-rail-item"
        :class="index === activeIndex ? 'is-active' : ''"
        :title="agent.name"
        :style="{ color: agent.color }"
        data-test="preview-agent-rail-item"
      >
        <ClientIcon v-if="agent.icon" :name="agent.icon" class="h-4 w-4" />
        <PlatformIcon v-else-if="agent.platform" :platform="agent.platform" size="md" />
      </span>
    </div>

    <!-- 停留的 Agent 名称浮层 -->
    <span
      class="cw-tooltip pointer-events-none absolute left-[52px] z-10 -translate-y-1/2 whitespace-nowrap rounded-md px-2 py-1 text-[11.5px] font-medium shadow-[0_6px_16px_rgba(0,0,0,0.2)] transition-all duration-200"
      :style="{ top: `${tooltipTop}px` }"
      data-test="preview-agent-tooltip"
    >{{ agents[activeIndex].name }}</span>

    <div class="flex min-w-0 flex-1 flex-col bg-[color:var(--cw-surface)]">
      <div class="m-2 flex h-[34px] items-center gap-2 rounded-[9px] border border-[color:var(--cw-border)] px-2.5 text-[13px] text-[color:var(--cw-text-ghost)]">
        <ClientIcon name="search" class="h-3.5 w-3.5" />
        <span>{{ t('home.clientWorkflow.picker.search') }}</span>
      </div>

      <div class="flex min-h-0 flex-1 gap-1 px-1.5 pb-1.5">
        <!-- 只有内置 Agent 有厂商列 -->
        <ul v-if="activeAgent.id === 'builtin'" class="hidden w-[124px] shrink-0 flex-col gap-px sm:flex">
          <li
            v-for="vendor in VENDORS"
            :key="vendor.name"
            class="flex h-8 items-center gap-2 rounded-[7px] px-2 text-[12.5px]"
            :class="vendor.selected
              ? 'bg-[color:var(--cw-overlay-strong)] text-[color:var(--cw-text)]'
              : 'text-[color:var(--cw-text-secondary)]'"
          >
            <PlatformIcon :platform="vendor.platform" size="sm" />
            <span class="truncate">{{ vendor.name }}</span>
          </li>
        </ul>

        <ul class="flex min-w-0 flex-1 flex-col gap-px">
          <li
            v-for="(model, index) in activeModels.models"
            :key="model"
            class="flex h-[52px] items-center gap-2 rounded-[9px] px-3"
            :class="activeAgent.id === 'builtin' && index === 0 ? 'bg-[color:var(--cw-overlay-strong)]' : ''"
          >
            <div class="min-w-0 flex-1">
              <div class="truncate text-[13px] font-semibold text-[color:var(--cw-text)]">{{ model }}</div>
              <div class="mt-0.5 flex items-center gap-1 truncate text-[12px] text-[color:var(--cw-text-tertiary)]">
                <PlatformIcon :platform="activeModels.vendor" size="xs" />
                <span class="truncate">{{ siteName }} · {{ activeModels.vendor }}</span>
              </div>
            </div>
            <ClientIcon name="star" class="h-3.5 w-3.5 shrink-0 text-[color:var(--cw-text-ghost)]" />
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { GroupPlatform } from '@/types'
import ClientIcon from './ClientIcon.vue'
import type { ClientIconName } from './icons'

const props = defineProps<{
  open: boolean
  /** 左栏停留的 Agent 下标 */
  activeIndex: number
  siteName: string
}>()

const { t } = useI18n()

interface AgentEntry {
  id: string
  name: string
  icon?: ClientIconName
  platform?: GroupPlatform
  color: string
}

const MARK = 'var(--cw-mark)'

// 顺序和配色同客户端 ProviderKind::ALL 与各家品牌色
const agents = computed<AgentEntry[]>(() => [
  { id: 'builtin', name: t('home.clientWorkflow.picker.builtinAgent'), icon: 'providerWaku', color: 'var(--cw-accent)' },
  { id: 'amp', name: 'Amp', icon: 'providerAmp', color: '#F34E3F' },
  { id: 'claude', name: 'Claude Code', platform: 'anthropic', color: '#D97757' },
  { id: 'codex', name: 'Codex CLI', platform: 'openai', color: MARK },
  { id: 'cursor', name: 'Cursor CLI', icon: 'providerCursor', color: MARK },
  { id: 'deepseek', name: 'DeepSeek Harness', platform: 'deepseek', color: '#4D6BFE' },
  { id: 'fx', name: 'Fx', icon: 'providerFx', color: MARK },
  { id: 'opencode', name: 'OpenCode', platform: 'opencode_go', color: MARK },
  { id: 'grok', name: 'Grok Build', platform: 'grok', color: MARK },
  { id: 'kimi', name: 'Kimi Code', platform: 'kimi', color: MARK },
  { id: 'ohmypi', name: 'Oh My Pi', icon: 'providerOhMyPi', color: MARK },
  { id: 'pi', name: 'Pi', icon: 'providerPi', color: MARK },
])

const VENDORS: Array<{ name: string; platform: GroupPlatform; selected?: boolean }> = [
  { name: 'Anthropic', platform: 'anthropic' },
  { name: 'OpenAI', platform: 'openai', selected: true },
  { name: 'Google', platform: 'gemini' },
  { name: 'xAI', platform: 'grok' },
  { name: 'DeepSeek', platform: 'deepseek' },
  { name: '智谱 GLM', platform: 'zhipu' },
  { name: 'Kimi', platform: 'kimi' },
  { name: 'MiniMax', platform: 'minimax' },
]

const MODELS: Record<string, { vendor: GroupPlatform; models: string[] }> = {
  builtin: { vendor: 'openai', models: ['gpt-5.6-sol', 'gpt-5.5', 'gpt-5.4', 'gpt-5.3-codex'] },
  claude: { vendor: 'anthropic', models: ['claude-opus-4-7', 'claude-sonnet-4-6', 'claude-haiku-4-5'] },
  codex: { vendor: 'openai', models: ['gpt-5.5', 'gpt-5.4', 'gpt-5.3-codex'] },
}

const activeAgent = computed(() => agents.value[props.activeIndex] ?? agents.value[0])
const activeModels = computed(() => MODELS[activeAgent.value.id] ?? MODELS.builtin)

// 左栏：上内边距 6 + 收藏 30 + 分隔 9（含两侧 1px 间距共 11），之后每项 30 + 1 间距；取图标中线
const tooltipTop = computed(() => 6 + 30 + 11 + props.activeIndex * 31 + 15)
</script>

<style scoped>
.cw-picker {
  border: 1px solid var(--cw-border-strong);
  background: var(--cw-raised);
}

.cw-rail {
  background: var(--cw-canvas);
}

.cw-rail-item {
  display: flex;
  height: 30px;
  width: 34px;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 7px;
  transition: background-color 0.2s ease;
}

.cw-rail-item.is-active {
  background: var(--cw-overlay-strong);
}

.cw-tooltip {
  background: var(--cw-inverse);
  color: var(--cw-on-inverse);
}

/* 指向左栏图标的小三角 */
.cw-tooltip::before {
  content: '';
  position: absolute;
  top: 50%;
  left: -4px;
  width: 8px;
  height: 8px;
  background: inherit;
  transform: translateY(-50%) rotate(45deg);
}
</style>
