import { getCurrentInstance } from 'vue'
import type { ContentTranslationLang } from '@/api/contentTranslations'
import {
  documentContentLang,
  normalizeContentLang,
  useContentTranslationsStore
} from '@/stores/contentTranslations'

// 内容自动翻译（fork 本地功能）的显示层入口。
//
// tx(text) 返回当前语言下的译文，没有译文时返回原文。只能用在「渲染出来的文字」上：
// 不要把结果写回数据对象，也不要拿它做搜索、过滤、比较、revision 等逻辑。
// 管理后台（/admin 下的路由）一律显示原文。
//
// 路由取自应用的 $route（vue-router 安装时挂到 globalProperties 上，且是响应式的），
// 而不是 useRoute()：很多组件测试会部分 mock vue-router，直接 import 它的导出会在那些测试里抛错。
// 没有路由上下文（纯 TS 模块、未装路由的测试）时退回 window.location.pathname。
//
// 语言同理取 vue-i18n 注入的 $i18n.locale（响应式，等同 @/i18n 的 getLocale()），不直接 import
// `@/i18n`：它在模块求值时调 createI18n，会让只 mock 了 useI18n 的组件测试在 import 阶段失败。
// 拿不到时退回 <html lang>（i18n 初始化 / 切换语言时会同步写入）。

export function isAdminContentPath(path: string | null | undefined): boolean {
  return typeof path === 'string' && (path === '/admin' || path.startsWith('/admin/'))
}

function windowPath(): string {
  try {
    return typeof window !== 'undefined' ? window.location.pathname : ''
  } catch {
    return ''
  }
}

function resolveStore(): ReturnType<typeof useContentTranslationsStore> | null {
  // 没有激活的 Pinia（例如未装 Pinia 的组件测试）时退化为原样显示。
  // 不用 getActivePinia()：它在组件里无默认值地 inject，没装 Pinia 时会打 Vue 警告。
  try {
    return useContentTranslationsStore()
  } catch {
    return null
  }
}

function translateWith(
  store: ReturnType<typeof useContentTranslationsStore> | null,
  text: string | null | undefined,
  path: string,
  lang: () => ContentTranslationLang
): string {
  if (text === null || text === undefined) return ''
  if (!store || isAdminContentPath(path)) return text
  return store.translate(text, lang())
}

export function useContentTranslation() {
  const instance = getCurrentInstance()
  const globals = instance?.appContext.config.globalProperties as
    | { $route?: { path?: unknown }; $i18n?: { locale?: unknown } }
    | undefined
  const store = resolveStore()

  function currentPath(): string {
    const route = globals?.$route
    if (route && typeof route.path === 'string') return route.path
    return windowPath()
  }

  function currentLang(): ContentTranslationLang {
    return normalizeContentLang(globals?.$i18n?.locale) ?? documentContentLang()
  }

  /** 显示用：返回当前语言下的译文或原文；/admin 路由下总是原文 */
  function tx(text?: string | null): string {
    return translateWith(store, text, currentPath(), currentLang)
  }

  return { tx }
}

/**
 * 非组件场景（如 router/title.ts）用的同一套规则：直接读 store。
 * path 传当前（或即将进入的）路由路径，不传时读 window.location.pathname；
 * lang 不传时读 <html lang>。
 */
export function translateContent(text?: string | null, path?: string, lang?: string): string {
  return translateWith(
    resolveStore(),
    text,
    typeof path === 'string' ? path : windowPath(),
    () => normalizeContentLang(lang) ?? documentContentLang()
  )
}
