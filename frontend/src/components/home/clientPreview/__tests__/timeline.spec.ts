import { describe, expect, it } from 'vitest'
import { CYCLE_MS, STATIC_T, TOOL_ROW_COUNT, deriveFrame } from '../timeline'

describe('client preview timeline', () => {
  it('starts on the agent scene with nothing shown yet', () => {
    const frame = deriveFrame(0)
    expect(frame.scene).toBe('agent')
    expect(frame.visibleRows).toBe(0)
    expect(frame.pickerOpen).toBe(false)
    expect(frame.runDone).toBe(false)
    expect(frame.charges).toBe(0)
  })

  it('reveals the tool rows one by one', () => {
    expect(deriveFrame(800).visibleRows).toBe(1)
    expect(deriveFrame(3000).visibleRows).toBe(TOOL_ROW_COUNT)
  })

  it('opens the model picker and walks the agent rail', () => {
    expect(deriveFrame(4500)).toMatchObject({ pickerOpen: true, pickerIndex: 0 })
    expect(deriveFrame(5200)).toMatchObject({ pickerOpen: true, pickerIndex: 2 })
    expect(deriveFrame(6000)).toMatchObject({ pickerOpen: true, pickerIndex: 3 })
    expect(deriveFrame(7000)).toMatchObject({ pickerOpen: true, pickerIndex: 0 })
    expect(deriveFrame(7500).pickerOpen).toBe(false)
  })

  it('settles the turn and charges once', () => {
    const frame = deriveFrame(8500)
    expect(frame.runDone).toBe(true)
    expect(frame.charges).toBe(1)
    expect(frame.workSeconds).toBe(deriveFrame(12000).workSeconds)
  })

  it('switches to the image studio, types, draws and charges again', () => {
    expect(deriveFrame(9300)).toMatchObject({ scene: 'agent', imageRowActive: true })

    const typing = deriveFrame(10600)
    expect(typing.scene).toBe('image')
    expect(typing.typed).toBeGreaterThan(0)
    expect(typing.typed).toBeLessThan(1)
    expect(typing.imageJob).toBe('idle')

    const drawing = deriveFrame(13000)
    expect(drawing).toMatchObject({ imageJob: 'drawing', submitted: true, charges: 1 })
    expect(drawing.drawSeconds).toBe(1)

    expect(deriveFrame(15000)).toMatchObject({ imageJob: 'done', charges: 2, fading: false })
  })

  it('fades out at the end and loops back to the start', () => {
    expect(deriveFrame(17200).fading).toBe(true)
    expect(deriveFrame(CYCLE_MS + 100)).toEqual(deriveFrame(100))
  })

  it('uses a static frame with every row shown and the picker open on the built-in agent', () => {
    expect(deriveFrame(STATIC_T)).toMatchObject({
      scene: 'agent',
      visibleRows: TOOL_ROW_COUNT,
      pickerOpen: true,
      pickerIndex: 0,
      runDone: false,
    })
  })
})
