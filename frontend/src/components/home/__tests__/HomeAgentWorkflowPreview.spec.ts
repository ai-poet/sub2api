import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import HomeAgentWorkflowPreview from '../HomeAgentWorkflowPreview.vue'
import { DEMO_IMAGES, preloadDemoImages } from '../clientPreview/demoImages'
import { CHAPTERS, FADE_IN_MS, MOMENTS } from '../clientPreview/timeline'

const translations: Record<string, string> = {
  'home.clientWorkflow.sidebar.newTask': 'New Task',
  'home.clientWorkflow.sidebar.search': 'Search',
  'home.clientWorkflow.sidebar.images': 'Images',
  'home.clientWorkflow.sidebar.modelStatus': 'Model status',
  'home.clientWorkflow.composer.placeholder': 'Do anything…',
  'home.clientWorkflow.composer.access': 'Full access',
  'home.clientWorkflow.composer.plan': 'Plan',
  'home.clientWorkflow.composer.build': 'Build',
  'home.clientWorkflow.transcript.prompt': 'Send people back after sign-in',
  'home.clientWorkflow.picker.builtinAgent': 'Built-in agent',
  'home.clientWorkflow.picker.descriptions.claudeSonnet': 'Balanced speed and intelligence.',
  'home.clientWorkflow.image.placeholder': 'Describe a picture…',
  'home.clientWorkflow.image.estimate': 'About $0.04',
  'home.clientWorkflow.image.generate': 'Draw',
  'home.clientWorkflow.image.gallery.sunset': 'Beach at sunset',
  'home.clientWorkflow.chapters.plan': 'Plan → run',
  'home.clientWorkflow.chapters.team': 'Agent team',
  'home.clientWorkflow.chapters.image': 'Images',
  'home.clientWorkflow.chapters.pause': 'Pause the demo',
  'home.clientWorkflow.chapters.play': 'Play the demo',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => translations[key] || key,
    }),
  }
})

vi.mock('../clientPreview/demoImages', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../clientPreview/demoImages')>()
  return { ...actual, preloadDemoImages: vi.fn() }
})

// ---- 可控的 rAF：advance(ms) 按 50ms 一步推进时间 ----
let rafQueue = new Map<number, FrameRequestCallback>()
let rafId = 0
let now = 0
const requestFrame = vi.fn((callback: FrameRequestCallback) => {
  rafId += 1
  rafQueue.set(rafId, callback)
  return rafId
})

async function advance(ms: number) {
  for (let spent = 0; spent < ms; spent += 50) {
    now += 50
    const callbacks = [...rafQueue.values()]
    rafQueue = new Map()
    callbacks.forEach((callback) => callback(now))
  }
  await nextTick()
}

// ---- 可控的 IntersectionObserver ----
let observers: FakeObserver[] = []
class FakeObserver {
  observe = vi.fn()
  disconnect = vi.fn()
  unobserve = vi.fn()
  constructor(private callback: (entries: Array<{ isIntersecting: boolean }>) => void) {
    observers.push(this)
  }
  report(isIntersecting: boolean) {
    this.callback([{ isIntersecting }])
  }
}

let hidden = false

// 测试环境的 matchMedia 默认对所有查询返回 true，这里按用例显式指定是否减少动态效果
function reduceMotion(matches: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn().mockReturnValue({ matches }),
  })
}

let wrapper: VueWrapper | null = null

function mountPreview() {
  wrapper = mount(HomeAgentWorkflowPreview, { props: { siteName: 'CheapRouter' } })
  return wrapper
}

function windowHtml(view: VueWrapper) {
  return view.find('[data-test="preview-window"]').html()
}

describe('HomeAgentWorkflowPreview', () => {
  const originalObserver = globalThis.IntersectionObserver

  beforeEach(() => {
    reduceMotion(false)
    rafQueue = new Map()
    rafId = 0
    now = 0
    observers = []
    hidden = false
    requestFrame.mockClear()
    vi.stubGlobal('requestAnimationFrame', requestFrame)
    vi.stubGlobal('cancelAnimationFrame', (id: number) => rafQueue.delete(id))
    globalThis.IntersectionObserver = FakeObserver as unknown as typeof IntersectionObserver
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => hidden })
    vi.mocked(preloadDemoImages).mockClear()
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
    globalThis.IntersectionObserver = originalObserver
  })

  it('draws the client sidebar with the Images and Model status entries', () => {
    const view = mountPreview()

    expect(view.find('[data-test="preview-sidebar-new-task"]').text()).toBe('New Task')
    expect(view.find('[data-test="preview-sidebar-search"]').text()).toBe('Search')
    expect(view.find('[data-test="preview-sidebar-images"]').text()).toBe('Images')
    expect(view.find('[data-test="preview-sidebar-model-status"]').text()).toBe('Model status')
  })

  it('lists tasks the way the client does: by project, one line each', () => {
    const view = mountPreview()

    // The toolbar opens on the project view.
    expect(view.find('[data-test="preview-view-project"]').classes()).toContain('is-active')
    // The project's tasks sit inside its group, a project-less one under 「任务」.
    expect(view.find('[data-test="preview-project-row"]').exists()).toBe(true)
    expect(view.find('[data-test="preview-session-plan"]').classes()).toContain('is-nested')
    expect(view.find('[data-test="preview-session-team"]').classes()).toContain('is-nested')
    expect(view.find('[data-test="preview-session-older"]').classes()).not.toContain('is-nested')
    // A reply finished elsewhere carries the unread dot.
    expect(view.find('[data-test="preview-unread-older"]').exists()).toBe(true)
    expect(view.find('[data-test="preview-unread-plan"]').exists()).toBe(false)
  })

  it('offers every agent in the model picker and describes each model', () => {
    const view = mountPreview()

    const names = view
      .findAll('[data-test="preview-agent-rail-item"]')
      .map((item) => item.attributes('title'))
    expect(names).toEqual([
      'Built-in agent',
      'Amp',
      'Claude Code',
      'Codex CLI',
      'Cursor CLI',
      'DeepSeek Harness',
      'Fx',
      'OpenCode',
      'Grok Build',
      'Kimi Code',
      'Oh My Pi',
      'Pi',
    ])

    const models = view.findAll('[data-test="preview-picker-model"]')
    expect(models.map((row) => row.find('.font-semibold').text())).toEqual([
      'Claude Sonnet 5.5',
      'Claude Opus 5.5',
      'Claude Fable 5.1',
    ])
    expect(models[0].text()).toContain('Balanced speed and intelligence.')
  })

  it('keeps both the agent scene and the image studio in the window', () => {
    const view = mountPreview()

    const agent = view.find('[data-test="preview-agent-scene"]')
    expect(agent.find('[data-test="preview-model-chip"]').text()).toContain('Claude Sonnet 5.5')
    expect(agent.text()).toContain('Full access')

    const studio = view.find('[data-test="preview-image-studio"]')
    expect(studio.text()).toContain('Describe a picture…')
    expect(studio.text()).toContain('gpt-image-2')
    expect(studio.text()).toContain('About $0.04')
    expect(studio.text()).toContain('Draw')
  })

  it('keeps the chapter buttons outside the window image', () => {
    const view = mountPreview()

    expect(view.attributes('role')).toBeUndefined()
    expect(view.find('[data-test="preview-window"]').attributes('role')).toBe('img')
    expect(view.find('[data-test="preview-window"] [data-test="preview-chapter-nav"]').exists()).toBe(false)
    expect(view.find('[data-test="preview-chapter-nav"]').findAll('button')).toHaveLength(4)
  })

  it('starts on a draft in plan mode with the picker closed', () => {
    const view = mountPreview()

    expect(view.find('[data-test="preview-composer-draft"]').text()).toContain('Send people back after sign-in')
    expect(view.find('[data-test="preview-mode-chip"]').text()).toBe('Plan')
    expect(view.find('[data-test="preview-model-picker"]').classes()).toContain('opacity-0')
    expect(view.find('[data-test="preview-balance"]').text()).toContain('$36.52')
    expect(view.find('[data-test="preview-chapter-plan"]').attributes('aria-current')).toBe('true')
    expect(vi.mocked(preloadDemoImages)).toHaveBeenCalledTimes(1)
  })

  it('opens on the new-task welcome screen with its activity overview until the prompt is sent', async () => {
    const view = mountPreview()

    expect(view.find('[data-test="preview-welcome"]').exists()).toBe(true)
    expect(view.find('[data-test="preview-overview"]').exists()).toBe(true)
    // 26 weeks of days, the last week only up to today.
    expect(view.findAll('[data-test="preview-heat-grid"] .cw-heat-cell').length).toBe(26 * 7)
    expect(view.find('[data-test="preview-plan-transcript"]').exists()).toBe(false)

    await advance(MOMENTS.send + 100)
    expect(view.find('[data-test="preview-welcome"]').exists()).toBe(false)
    expect(view.find('[data-test="preview-plan-transcript"]').exists()).toBe(true)
  })

  it('hands over the plan in a card and the Plan panel, then runs sub-agents after approval', async () => {
    const view = mountPreview()

    await advance(MOMENTS.planReady + 100)
    expect(view.find('[data-test="preview-plan-card"]').exists()).toBe(true)
    expect(view.find('[data-test="preview-plan-panel"]').attributes('data-status')).toBe('pending')
    expect(view.find('[data-test="preview-surface-plan"]').exists()).toBe(true)

    await advance(MOMENTS.subagentRows[1] - MOMENTS.planReady)
    expect(view.find('[data-test="preview-plan-card"]').exists()).toBe(false)
    expect(view.find('[data-test="preview-plan-panel"]').attributes('data-status')).toBe('approved')
    expect(view.findAll('[data-test="preview-subagent-row"]')).toHaveLength(2)
    expect(view.find('[data-test="preview-background-dot"]').exists()).toBe(true)
    expect(view.find('[data-test="preview-mode-chip"]').text()).toBe('Build')

    await advance(MOMENTS.runDone - MOMENTS.subagentRows[1])
    expect(view.find('[data-test="preview-turn-done"]').exists()).toBe(true)
    expect(view.find('[data-test="preview-balance"]').text()).toContain('$36.34')
  })

  it('jumps to the team chapter and runs the team', async () => {
    const view = mountPreview()

    await view.find('[data-test="preview-chapter-team"]').trigger('click')
    expect(view.find('[data-test="preview-chapter-team"]').attributes('aria-current')).toBe('true')
    expect(view.find('[data-test="preview-chapter-plan"]').attributes('aria-current')).toBeUndefined()
    expect(view.find('[data-test="preview-team-panel"]').attributes('data-phase')).toBe('review')
    expect(view.find('[data-test="preview-surface-team"]').exists()).toBe(true)
    expect(view.findAll('[data-test="preview-team-task"]')).toHaveLength(4)

    await advance(MOMENTS.teamStart - CHAPTERS[1].start)
    expect(view.find('[data-test="preview-team-panel"]').attributes('data-phase')).toBe('running')

    await advance(MOMENTS.teamDone - MOMENTS.teamStart)
    expect(view.find('[data-test="preview-team-panel"]').attributes('data-phase')).toBe('done')
    expect(view.find('[data-test="preview-team-done-reply"]').exists()).toBe(true)
  })

  it('pauses while the pointer moves over the window and carries on from there', async () => {
    const view = mountPreview()
    await advance(3000)

    await view.find('[data-test="preview-window"]').trigger('pointermove', { pointerType: 'mouse', movementX: 4 })
    await nextTick()
    const held = windowHtml(view)
    await advance(3000)
    expect(windowHtml(view)).toBe(held)

    await view.find('[data-test="preview-window"]').trigger('pointerleave')
    await nextTick()
    await advance(1500)
    expect(windowHtml(view)).not.toBe(held)
  })

  it('keeps playing when the page scrolls the window under a still pointer', async () => {
    const view = mountPreview()
    await advance(1000)

    // What a browser reports when content scrolls under a pointer that did not move.
    await view.find('[data-test="preview-window"]').trigger('pointermove', { pointerType: 'mouse', movementX: 0, movementY: 0 })
    await nextTick()
    const before = windowHtml(view)
    await advance(2000)
    expect(windowHtml(view)).not.toBe(before)
  })

  it('lets go of a hover pause once the page scrolls', async () => {
    const view = mountPreview()
    await advance(3000)

    await view.find('[data-test="preview-window"]').trigger('pointermove', { pointerType: 'mouse', movementY: 2 })
    await nextTick()
    const held = windowHtml(view)
    await advance(1500)
    expect(windowHtml(view)).toBe(held)

    window.dispatchEvent(new Event('scroll'))
    await nextTick()
    await advance(1500)
    expect(windowHtml(view)).not.toBe(held)
  })

  it('ignores touch pointers for pausing', async () => {
    const view = mountPreview()
    await advance(1000)

    await view.find('[data-test="preview-window"]').trigger('pointermove', { pointerType: 'touch', movementX: 4 })
    await nextTick()
    const before = windowHtml(view)
    await advance(2000)
    expect(windowHtml(view)).not.toBe(before)
  })

  it('pauses offscreen and resumes when scrolled back', async () => {
    const view = mountPreview()
    await advance(2500)

    observers[0].report(false)
    await nextTick()
    const held = windowHtml(view)
    await advance(3000)
    expect(windowHtml(view)).toBe(held)

    observers[0].report(true)
    await nextTick()
    await advance(1500)
    expect(windowHtml(view)).not.toBe(held)
  })

  it('pauses while the tab is hidden', async () => {
    const view = mountPreview()
    await advance(2500)

    hidden = true
    document.dispatchEvent(new Event('visibilitychange'))
    await nextTick()
    const held = windowHtml(view)
    await advance(3000)
    expect(windowHtml(view)).toBe(held)
  })

  it('pauses and plays from the chapter toggle', async () => {
    const view = mountPreview()
    await advance(2500)

    const toggle = view.find('[data-test="preview-chapter-toggle"]')
    expect(toggle.attributes('aria-label')).toBe('Pause the demo')
    await toggle.trigger('click')
    expect(toggle.attributes('aria-label')).toBe('Play the demo')
    const held = windowHtml(view)
    await advance(3000)
    expect(windowHtml(view)).toBe(held)

    await toggle.trigger('click')
    await nextTick()
    await advance(1500)
    expect(toggle.attributes('aria-label')).toBe('Pause the demo')
    expect(windowHtml(view)).not.toBe(held)
  })

  it('pauses on keyboard focus and shows a chapter at rest when picked', async () => {
    const original = Element.prototype.matches
    vi.spyOn(Element.prototype, 'matches').mockImplementation(function (this: Element, selector: string) {
      return selector === ':focus-visible' ? true : original.call(this, selector)
    })
    const view = mountPreview()

    const imageChapter = view.find('[data-test="preview-chapter-image"]')
    await imageChapter.trigger('focusin')
    await nextTick()
    expect(view.find('[data-test="preview-chapter-toggle"]').attributes('aria-label')).toBe('Play the demo')

    await imageChapter.trigger('click')
    expect(view.find('[data-test="preview-image-done"]').exists()).toBe(true)
  })

  it('draws no loop with reduced motion and switches still frames by chapter', async () => {
    reduceMotion(true)
    const view = mountPreview()

    expect(requestFrame).not.toHaveBeenCalled()
    expect(view.find('[data-test="preview-plan-panel"]').attributes('data-status')).toBe('pending')
    expect(view.find('[data-test="preview-chapter-toggle"]').exists()).toBe(false)
    expect(view.find('[data-test="preview-chapter-fill"]').exists()).toBe(false)

    await view.find('[data-test="preview-chapter-team"]').trigger('click')
    expect(view.find('[data-test="preview-team-panel"]').attributes('data-phase')).toBe('running')
    expect(view.find('[data-test="preview-team-round"]').exists()).toBe(true)
    expect(view.find('[data-test="preview-chapter-team"]').attributes('aria-current')).toBe('true')

    await view.find('[data-test="preview-chapter-image"]').trigger('click')
    expect(view.find('[data-test="preview-image-done"]').exists()).toBe(true)
    expect(requestFrame).not.toHaveBeenCalled()
  })

  it('shows real photos in the gallery and a plain tile when one fails to load', async () => {
    const view = mountPreview()

    const images = view.findAll('[data-test="preview-gallery-card"] img')
    expect(images).toHaveLength(3)
    expect(images[0].attributes('src')).toBe(DEMO_IMAGES.sunset)
    expect(images[0].attributes('alt')).toBe('Beach at sunset')
    for (const image of images) expect(image.attributes('src')).toMatch(/^https:\/\//)

    await images[0].trigger('error')
    expect(view.findAll('[data-test="preview-picture-fallback"]')).toHaveLength(1)

    await view.find('[data-test="preview-chapter-image"]').trigger('click')
    // 跳章后第一帧只记时间不推进，多走 200ms 余量
    await advance(MOMENTS.imageDone - (CHAPTERS[2].start + FADE_IN_MS) + 200)
    expect(view.find('[data-test="preview-image-done"] img').attributes('src')).toBe(DEMO_IMAGES.cat)
  })
})
