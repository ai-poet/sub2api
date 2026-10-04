import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, reactive, type App } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'

const state = vi.hoisted(() => ({
  lookup: vi.fn()
}))

vi.mock('@/api/contentTranslations', () => ({
  lookupContentTranslations: state.lookup
}))

import {
  isAdminContentPath,
  translateContent,
  useContentTranslation
} from '@/composables/useContentTranslation'
import { CONTENT_TRANSLATION_DEBOUNCE_MS } from '@/stores/contentTranslations'

const Probe = defineComponent({
  props: { text: { type: String, required: true } },
  setup(props) {
    const { tx } = useContentTranslation()
    return () => h('span', { 'data-test': 'probe' }, tx(props.text))
  }
})

function makeRouter(): Router {
  const Empty = defineComponent({ render: () => null })
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/keys', component: Empty },
      { path: '/admin/settings', component: Empty }
    ]
  })
}

async function mountAt(path: string, pinia: Pinia, router: Router, extraPlugins: Array<{ install(app: App): void }> = []) {
  await router.push(path)
  await router.isReady()
  return mount(Probe, {
    props: { text: '分组描述' },
    global: { plugins: [pinia, router, ...extraPlugins] }
  })
}

/** 推进假时钟，并等动态 import 的查询模块就绪、查询结果落地 */
async function settle() {
  await vi.advanceTimersByTimeAsync(CONTENT_TRANSLATION_DEBOUNCE_MS)
  await vi.dynamicImportSettled()
  await vi.advanceTimersByTimeAsync(0)
  await flushPromises()
}

describe('useContentTranslation (fork)', () => {
  let pinia: Pinia

  beforeEach(() => {
    vi.useFakeTimers()
    document.documentElement.setAttribute('lang', 'en')
    pinia = createPinia()
    setActivePinia(pinia)
    state.lookup.mockReset()
    state.lookup.mockImplementation((lang: string, texts: string[]) =>
      Promise.resolve({
        lang,
        translations: Object.fromEntries(texts.map((text) => [text, `EN(${text})`])),
        pending: false
      })
    )
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders the translation on user routes once it arrives', async () => {
    const wrapper = await mountAt('/keys', pinia, makeRouter())
    expect(wrapper.get('[data-test="probe"]').text()).toBe('分组描述')

    await settle()

    expect(state.lookup).toHaveBeenCalledWith('en', ['分组描述'])
    expect(wrapper.get('[data-test="probe"]').text()).toBe('EN(分组描述)')
  })

  it('always shows the original text on /admin routes and never looks it up', async () => {
    const wrapper = await mountAt('/admin/settings', pinia, makeRouter())

    await settle()

    expect(wrapper.get('[data-test="probe"]').text()).toBe('分组描述')
    expect(state.lookup).not.toHaveBeenCalled()
  })

  it('switches back to the original when navigating into the admin console', async () => {
    const router = makeRouter()
    const wrapper = await mountAt('/keys', pinia, router)
    await settle()
    expect(wrapper.get('[data-test="probe"]').text()).toBe('EN(分组描述)')

    await router.push('/admin/settings')
    await flushPromises()
    expect(wrapper.get('[data-test="probe"]').text()).toBe('分组描述')
  })

  it('returns the original text when no Pinia is active', () => {
    setActivePinia(undefined as unknown as Pinia)
    const wrapper = mount(Probe, { props: { text: '分组描述' } })
    expect(wrapper.get('[data-test="probe"]').text()).toBe('分组描述')
    expect(translateContent('分组描述', '/keys')).toBe('分组描述')
  })

  it('translateContent applies the same admin rule outside components', async () => {
    expect(translateContent('自定义菜单', '/custom/x')).toBe('自定义菜单')
    await settle()
    expect(translateContent('自定义菜单', '/custom/x')).toBe('EN(自定义菜单)')
    expect(translateContent('自定义菜单', '/admin/users')).toBe('自定义菜单')
    expect(translateContent(null)).toBe('')
  })

  it('follows the vue-i18n locale ($i18n.locale) reactively', async () => {
    const i18nGlobals = reactive({ locale: 'zh' })
    const fakeI18n = {
      install(app: App) {
        app.config.globalProperties.$i18n = i18nGlobals as never
      }
    }
    const wrapper = mount(Probe, {
      props: { text: 'English group' },
      global: { plugins: [pinia, fakeI18n] }
    })
    await settle()
    expect(state.lookup).toHaveBeenLastCalledWith('zh', ['English group'])
    expect(wrapper.get('[data-test="probe"]').text()).toBe('EN(English group)')

    i18nGlobals.locale = 'en'
    await flushPromises()
    // 英文界面下纯英文文本不查询，直接显示原文
    expect(wrapper.get('[data-test="probe"]').text()).toBe('English group')
  })

  it('matches only the admin console prefix', () => {
    expect(isAdminContentPath('/admin')).toBe(true)
    expect(isAdminContentPath('/admin/ops')).toBe(true)
    expect(isAdminContentPath('/administrator')).toBe(false)
    expect(isAdminContentPath('/keys')).toBe(false)
    expect(isAdminContentPath(undefined)).toBe(false)
  })
})
