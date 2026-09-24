<template>
  <figure
    data-test="api-terminal"
    class="mx-auto w-full max-w-[880px] overflow-hidden rounded-2xl border border-black/10 bg-[#111214] text-left shadow-[0_30px_80px_rgba(15,17,20,0.18)] dark:border-white/10 dark:shadow-[0_30px_80px_rgba(0,0,0,0.55)]"
  >
    <div class="flex items-center gap-2 border-b border-white/10 px-4 py-3">
      <span class="h-3 w-3 rounded-full bg-[#ff5f57]"></span>
      <span class="h-3 w-3 rounded-full bg-[#febc2e]"></span>
      <span class="h-3 w-3 rounded-full bg-[#28c840]"></span>
      <span class="ml-3 text-xs font-medium text-white/45">{{ t('home.landing.hero.terminal.title') }}</span>
    </div>
    <div class="overflow-x-auto px-5 py-5 font-mono text-[13px] leading-7 text-white/85 md:px-7 md:py-6 md:text-sm">
      <div v-for="(line, index) in lines" :key="index" class="whitespace-pre">
        <template v-if="line.length === 0">&nbsp;</template>
        <span v-for="(token, tokenIndex) in line" :key="tokenIndex" :class="tokenClass[token.kind]">{{ token.text }}</span>
      </div>
    </div>
    <figcaption class="border-t border-white/10 px-5 py-3 text-xs text-white/50 md:px-7">
      {{ t('home.landing.hero.terminal.caption') }}
    </figcaption>
  </figure>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

// 没有客户端时首屏的主视觉：两段环境变量，说明只换 Base URL 就能接入。
// 地址规则与 IntegrationGuidePanel 一致：Anthropic 用站点根地址，OpenAI 兼容用 /v1。
const props = defineProps<{
  baseUrl: string
}>()

const { t } = useI18n()

type TokenKind = 'comment' | 'keyword' | 'plain' | 'string'
interface Token {
  kind: TokenKind
  text: string
}

const tokenClass: Record<TokenKind, string> = {
  comment: 'text-white/40',
  keyword: 'text-primary-300',
  plain: 'text-white/85',
  string: 'text-amber-200',
}

const PLACEHOLDER_KEY = '"sk-••••••••"'

const rootUrl = computed(() => props.baseUrl.replace(/\/v1\/?$/, '').replace(/\/+$/, ''))

function envLine(name: string, value: string): Token[] {
  return [
    { kind: 'keyword', text: 'export ' },
    { kind: 'plain', text: `${name}=` },
    { kind: 'string', text: value },
  ]
}

const lines = computed<Token[][]>(() => [
  [{ kind: 'comment', text: t('home.landing.hero.terminal.claudeComment') }],
  envLine('ANTHROPIC_BASE_URL', `"${rootUrl.value}"`),
  envLine('ANTHROPIC_AUTH_TOKEN', PLACEHOLDER_KEY),
  [],
  [{ kind: 'comment', text: t('home.landing.hero.terminal.openaiComment') }],
  envLine('OPENAI_BASE_URL', `"${rootUrl.value}/v1"`),
  envLine('OPENAI_API_KEY', PLACEHOLDER_KEY),
])
</script>
