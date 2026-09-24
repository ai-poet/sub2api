import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import HomeAgentWorkflowPreview from '../HomeAgentWorkflowPreview.vue'

const translations: Record<string, string> = {
  'home.clientWorkflow.sidebar.newTask': 'New Task',
  'home.clientWorkflow.sidebar.search': 'Search',
  'home.clientWorkflow.sidebar.images': 'Images',
  'home.clientWorkflow.sidebar.taskTitle': 'Return to the page after login',
  'home.clientWorkflow.composer.placeholder': 'Do anything…',
  'home.clientWorkflow.composer.access': 'Full access',
  'home.clientWorkflow.picker.builtinAgent': 'Built-in agent',
  'home.clientWorkflow.image.placeholder': 'Describe a picture…',
  'home.clientWorkflow.image.estimate': 'About $0.04',
  'home.clientWorkflow.image.generate': 'Draw',
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

function mountPreview() {
  return mount(HomeAgentWorkflowPreview, { props: { siteName: 'CheapRouter' } })
}

// 测试环境的 matchMedia 默认对所有查询返回 true，这里按用例显式指定是否减少动态效果
function reduceMotion(matches: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn().mockReturnValue({ matches }),
  })
}

describe('HomeAgentWorkflowPreview', () => {
  beforeEach(() => {
    reduceMotion(false)
  })

  it('draws the client sidebar with the Images entry', () => {
    const wrapper = mountPreview()

    expect(wrapper.find('[data-test="preview-sidebar-new-task"]').text()).toBe('New Task')
    expect(wrapper.find('[data-test="preview-sidebar-search"]').text()).toBe('Search')
    expect(wrapper.find('[data-test="preview-sidebar-images"]').text()).toBe('Images')
  })

  it('offers every agent in the model picker, built-in agent first', () => {
    const wrapper = mountPreview()

    const names = wrapper
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
  })

  it('keeps both the agent scene and the image studio in the window', () => {
    const wrapper = mountPreview()

    const agent = wrapper.find('[data-test="preview-agent-scene"]')
    expect(agent.text()).toContain('Do anything…')
    expect(agent.text()).toContain('gpt-5.6-sol')
    expect(agent.text()).toContain('Full access')

    const studio = wrapper.find('[data-test="preview-image-studio"]')
    expect(studio.text()).toContain('Describe a picture…')
    expect(studio.text()).toContain('gpt-image-2')
    expect(studio.text()).toContain('About $0.04')
    expect(studio.text()).toContain('Draw')
  })

  it('starts the animation from an empty turn with the picker closed', () => {
    const wrapper = mountPreview()

    expect(wrapper.find('[data-test="preview-model-picker"]').classes()).toContain('opacity-0')
    expect(wrapper.find('[data-test="preview-tool-rows"]').findAll('.cw-row-in')).toHaveLength(0)
    expect(wrapper.find('[data-test="preview-balance"]').text()).toContain('$36.52')
    expect(wrapper.find('[data-test="preview-agent-tooltip"]').text()).toBe('Built-in agent')
  })

  it('shows the whole working turn with the picker closed when motion is reduced', () => {
    reduceMotion(true)
    const wrapper = mountPreview()

    expect(wrapper.find('[data-test="preview-model-picker"]').classes()).toContain('opacity-0')
    expect(wrapper.find('[data-test="preview-tool-rows"]').findAll('.cw-row-in')).toHaveLength(6)
  })
})
