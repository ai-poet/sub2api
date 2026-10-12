// fork：模型输入/输出模态推断（移植自 TokenRouter utils/modelCapabilities.ts，LGPL-3.0）。
// 模型广场卡片右上角「输入 → 输出」图标的数据源。目录接口没有下发模态元数据，
// 计费模式（图片 / 视频）比模型 ID 更可靠，优先使用；ID 模式只作兜底。

import type { ModelCatalogBillingMode } from '@/api/modelCatalog'

export type ModelModality = 'text' | 'image' | 'audio' | 'video'

export interface ModelCapabilities {
  input: ModelModality[]
  output: ModelModality[]
}

// 标签固定按该顺序渲染，保证不同卡片之间可扫读对比。
export const MODEL_MODALITY_ORDER: ModelModality[] = ['text', 'image', 'audio', 'video']

// 已知生图模型的 ID 特征（命中后只输出图片）。
const IMAGE_OUTPUT_PATTERNS = [/dall-e/, /flux/, /stable-diffusion/, /imagen/, /cogview/, /midjourney/, /mj-/, /gpt-image/, /-image(?:-|$)/]

// 已知视频生成模型的 ID 特征（命中后只输出视频）。
const VIDEO_OUTPUT_PATTERNS = [/veo/, /sora/, /cogvideo/, /-video(?:-|$)/]

// 音频输入输出双通的对话模型（如 gpt-4o-audio、realtime 系列）。
const AUDIO_IO_PATTERNS = [/gpt-4o-audio/, /gpt-audio/, /realtime/]

// 仅音频输入的模型（语音识别类）。
const AUDIO_INPUT_PATTERNS = [/whisper/, /voxtral/, /qwen-audio/, /transcribe/]

// 仅音频输出的模型（语音合成类）。
const AUDIO_OUTPUT_PATTERNS = [/tts/]

// 已知支持图片理解（视觉输入）的模型 ID 特征。
const IMAGE_INPUT_PATTERNS = [
  /claude/,
  /gemini/,
  /gpt-4o/,
  /gpt-4\.1/,
  /gpt-5/,
  /gpt-6/,
  /\bo3\b/,
  /\bo4\b/,
  /grok-4/,
  /qwen-vl/,
  /qwen3-vl/,
  /glm-4v/,
  /glm-4\.5v/,
  /pixtral/,
  /llama-4/,
  /kimi-k2\.5/,
]

// 支持图片编辑（图生图）的生图模型：输入除文字外还有图片。
const IMAGE_EDIT_PATTERNS = [/gpt-image/, /-image(?:-|$)/, /flux-kontext/]

function sortedModalities(modalities: Set<ModelModality>): ModelModality[] {
  return MODEL_MODALITY_ORDER.filter((modality) => modalities.has(modality))
}

export function resolveModelCapabilities(
  modelId: string,
  billingMode?: ModelCatalogBillingMode | string,
): ModelCapabilities {
  const id = modelId.toLowerCase()

  // 大部分模型至少支持文字输入输出，作为默认值；命中更具体的能力时再调整。
  const input = new Set<ModelModality>(['text'])
  const output = new Set<ModelModality>(['text'])

  const imageOutput = billingMode === 'image' || IMAGE_OUTPUT_PATTERNS.some((pattern) => pattern.test(id))
  const videoOutput = billingMode === 'video' || VIDEO_OUTPUT_PATTERNS.some((pattern) => pattern.test(id))
  const audioIO = AUDIO_IO_PATTERNS.some((pattern) => pattern.test(id))
  const audioInput = AUDIO_INPUT_PATTERNS.some((pattern) => pattern.test(id))
  const audioOutput = AUDIO_OUTPUT_PATTERNS.some((pattern) => pattern.test(id))

  // 视频生成优先于图片：grok-imagine-video 这类 ID 同时带 image 和 video 字样。
  if (videoOutput) {
    output.delete('text')
    output.add('video')
  } else if (imageOutput) {
    // 生图模型只输出图片，没有对话输出。
    output.delete('text')
    output.add('image')
  }

  if (audioIO) {
    input.add('audio')
    output.add('audio')
  }
  if (audioInput) {
    // 语音识别类模型输入只有音频。
    input.delete('text')
    input.add('audio')
  }
  if (audioOutput) {
    // 语音合成类模型输出只有音频。
    output.delete('text')
    output.add('audio')
  }

  // 纯生成类模型默认只有文字输入；支持图生图的生图模型叠加图片输入；
  // 音频对话 / 识别模型不按 gpt-4o 视觉规则叠加图片输入。
  const generativeOnly = imageOutput || videoOutput || audioOutput
  if (imageOutput && !videoOutput && IMAGE_EDIT_PATTERNS.some((pattern) => pattern.test(id))) {
    input.add('image')
  } else if (!generativeOnly && !audioIO && !audioInput) {
    if (IMAGE_INPUT_PATTERNS.some((pattern) => pattern.test(id))) {
      input.add('image')
    }
  }

  return {
    input: sortedModalities(input),
    output: sortedModalities(output),
  }
}
