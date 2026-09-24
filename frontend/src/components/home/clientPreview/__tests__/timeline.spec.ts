import { describe, expect, it } from 'vitest'
import { CYCLE_MS, STATIC_T, TOOL_ROW_COUNT, deriveFrame } from '../timeline'

describe('client preview timeline', () => {
  it('starts on the agent scene with nothing shown yet, fading in', () => {
    const frame = deriveFrame(0)
    expect(frame.scene).toBe('agent')
    expect(frame.visibleRows).toBe(0)
    expect(frame.pickerOpen).toBe(false)
    expect(frame.runDone).toBe(false)
    expect(frame.charges).toBe(0)
    expect(frame.fading).toBe(true)
    expect(deriveFrame(500).fading).toBe(false)
  })

  it('reveals the tool rows one by one', () => {
    expect(deriveFrame(900).visibleRows).toBe(1)
    expect(deriveFrame(2000).visibleRows).toBe(2)
    expect(deriveFrame(5400).visibleRows).toBe(TOOL_ROW_COUNT)
  })

  it('opens the model picker and walks the agent rail', () => {
    expect(deriveFrame(6500)).toMatchObject({ pickerOpen: true, pickerIndex: 0 })
    expect(deriveFrame(7500)).toMatchObject({ pickerOpen: true, pickerIndex: 2 })
    expect(deriveFrame(8500)).toMatchObject({ pickerOpen: true, pickerIndex: 3 })
    expect(deriveFrame(9500)).toMatchObject({ pickerOpen: true, pickerIndex: 0 })
    expect(deriveFrame(10100).pickerOpen).toBe(false)
  })

  it('settles the turn and charges once', () => {
    const frame = deriveFrame(11000)
    expect(frame.runDone).toBe(true)
    expect(frame.charges).toBe(1)
    expect(frame.workSeconds).toBe(deriveFrame(13000).workSeconds)
  })

  it('spends most of the loop on the agent and only a short stretch drawing', () => {
    expect(deriveFrame(13500).scene).toBe('agent')
    expect(deriveFrame(14100).scene).toBe('image')
  })

  it('switches to the image studio, types, draws and charges again', () => {
    expect(deriveFrame(13700)).toMatchObject({ scene: 'agent', imageRowActive: true })

    const typing = deriveFrame(14700)
    expect(typing.scene).toBe('image')
    expect(typing.typed).toBeGreaterThan(0)
    expect(typing.typed).toBeLessThan(1)
    expect(typing.imageJob).toBe('idle')

    const drawing = deriveFrame(16500)
    expect(drawing).toMatchObject({ imageJob: 'drawing', submitted: true, charges: 1 })
    expect(drawing.drawSeconds).toBe(1)

    expect(deriveFrame(17500)).toMatchObject({ imageJob: 'done', charges: 2, fading: false })
  })

  it('fades out at the end and loops back to the start', () => {
    expect(deriveFrame(18500).fading).toBe(true)
    expect(deriveFrame(CYCLE_MS + 100)).toEqual(deriveFrame(100))
  })

  it('uses a still frame with the whole working turn and the picker closed', () => {
    expect(deriveFrame(STATIC_T)).toMatchObject({
      scene: 'agent',
      visibleRows: TOOL_ROW_COUNT,
      pickerOpen: false,
      runDone: false,
      fading: false,
    })
  })
})
