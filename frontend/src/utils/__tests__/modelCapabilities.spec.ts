import { describe, expect, it } from 'vitest'

import { resolveModelCapabilities } from '../modelCapabilities'

describe('resolveModelCapabilities', () => {
  it('treats vision chat models as text + image in, text out', () => {
    expect(resolveModelCapabilities('claude-sonnet-4-6', 'token')).toEqual({ input: ['text', 'image'], output: ['text'] })
    expect(resolveModelCapabilities('gpt-5.5', 'token')).toEqual({ input: ['text', 'image'], output: ['text'] })
  })

  it('keeps plain text models text only', () => {
    expect(resolveModelCapabilities('deepseek-v3.2', 'token')).toEqual({ input: ['text'], output: ['text'] })
  })

  it('uses the billing mode before the model id for media models', () => {
    // 计费模式是图片时，即使 ID 没有特征也只输出图片
    expect(resolveModelCapabilities('mystery-model', 'image').output).toEqual(['image'])
    expect(resolveModelCapabilities('grok-imagine-video', 'video').output).toEqual(['video'])
  })

  it('adds image input for image editing models', () => {
    expect(resolveModelCapabilities('gpt-image-2', 'image')).toEqual({ input: ['text', 'image'], output: ['image'] })
  })

  it('recognises speech models', () => {
    expect(resolveModelCapabilities('whisper-1')).toEqual({ input: ['audio'], output: ['text'] })
    expect(resolveModelCapabilities('tts-1')).toEqual({ input: ['text'], output: ['audio'] })
    expect(resolveModelCapabilities('gpt-4o-audio-preview')).toEqual({ input: ['text', 'audio'], output: ['text', 'audio'] })
  })
})
