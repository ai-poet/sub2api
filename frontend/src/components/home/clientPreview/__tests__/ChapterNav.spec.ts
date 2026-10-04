import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ChapterNav from '../ChapterNav.vue'

const CHAPTERS = [
  { id: 'plan', label: 'Plan → run' },
  { id: 'team', label: 'Agent team' },
  { id: 'image', label: 'Images' },
]

function mountNav(props: Partial<{ current: number; progress: number; paused: boolean; showProgress: boolean }> = {}) {
  return mount(ChapterNav, {
    props: {
      chapters: CHAPTERS,
      current: 1,
      progress: 0.25,
      paused: false,
      showProgress: true,
      label: 'Demo chapters',
      pauseLabel: 'Pause the demo',
      playLabel: 'Play the demo',
      ...props,
    },
  })
}

describe('ChapterNav', () => {
  it('labels the nav and every chapter, marking the current one', () => {
    const wrapper = mountNav()

    expect(wrapper.find('nav').attributes('aria-label')).toBe('Demo chapters')
    const buttons = CHAPTERS.map((chapter) => wrapper.find(`[data-test="preview-chapter-${chapter.id}"]`))
    expect(buttons.map((button) => button.text())).toEqual(['Plan → run', 'Agent team', 'Images'])
    expect(buttons.map((button) => button.attributes('aria-current'))).toEqual([undefined, 'true', undefined])
  })

  it('fills earlier chapters, the current one by progress, and leaves later ones empty', () => {
    const wrapper = mountNav()

    const fills = wrapper.findAll('[data-test="preview-chapter-fill"]').map((fill) => fill.attributes('style'))
    expect(fills).toEqual(['transform: scaleX(1);', 'transform: scaleX(0.25);', 'transform: scaleX(0);'])
  })

  it('emits the picked chapter and the pause toggle', async () => {
    const wrapper = mountNav()

    await wrapper.find('[data-test="preview-chapter-image"]').trigger('click')
    await wrapper.find('[data-test="preview-chapter-toggle"]').trigger('click')
    expect(wrapper.emitted('select')).toEqual([[2]])
    expect(wrapper.emitted('toggle')).toHaveLength(1)
  })

  it('names the toggle after what it will do', () => {
    expect(mountNav({ paused: false }).find('[data-test="preview-chapter-toggle"]').attributes('aria-label')).toBe(
      'Pause the demo',
    )
    expect(mountNav({ paused: true }).find('[data-test="preview-chapter-toggle"]').attributes('aria-label')).toBe(
      'Play the demo',
    )
  })

  it('drops the progress bars and the toggle when motion is reduced', () => {
    const wrapper = mountNav({ showProgress: false })

    expect(wrapper.find('[data-test="preview-chapter-fill"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="preview-chapter-toggle"]').exists()).toBe(false)
    expect(wrapper.findAll('button')).toHaveLength(3)
  })
})
