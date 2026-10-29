/**
 * canvasComposite.ts 把画布内容合成为可提交的 PNG，并生成 OpenAI edits 需要的遮罩。
 *
 * 画布的最终产物是两个字节流：
 * - 合成图：用户看到的图层叠加结果，作为 images/edits 的 image 部分；
 * - 遮罩图：只有笔刷覆盖区域为黑色（表示要重绘），其余为白色。
 *
 * 遮罩遵循 OpenAI images/edits 的约定：透明（alpha=0）的区域是要重绘的部分。
 * 我们用 alpha 通道表达遮罩，而不是黑白像素，这样和上游的实际读法一致。
 */

/** 提交给上游的最大图片边长。超过时按比例缩小，避免触发 multipart 单片 50MB 限制。 */
export const MAX_CANVAS_EXPORT_EDGE = 2048

/** 合成的图片 MIME 类型。 */
export const CANVAS_EXPORT_MIME = 'image/png'

/** 遮罩的 MIME 类型。 */
export const CANVAS_MASK_MIME = 'image/png'

/**
 * CompositeOptions 描述一次合成所需的输入。
 */
export interface CompositeOptions {
  /** 合成图的宽度（像素）。 */
  width: number
  /** 合成图的高度（像素）。 */
  height: number
  /** 遮罩画布的宽度（像素）。需要与合成图同尺寸。 */
  maskWidth: number
  /** 遮罩画布的高度（像素）。需要与合成图同尺寸。 */
  maskHeight: number
  /** 导出时的背景色，CSS 颜色值。 */
  background: string
}

/**
 * CompositeResult 是一次合成的产物。
 */
export interface CompositeResult {
  /** 合成后的图片字节。 */
  blob: Blob
  /** 遮罩字节。没有笔刷痕迹时为 null，表示不需要遮罩。 */
  mask: Blob | null
  /** 合成图的实际宽度。 */
  width: number
  /** 合成图的实际高度。 */
  height: number
}

/**
 * fitEdge 把边长约束在 maxEdge 以内，保持宽高比。
 * 已经小于上限时原样返回。
 */
export function fitEdge(width: number, height: number, maxEdge: number): { width: number; height: number } {
  if (width <= 0 || height <= 0) {
    return { width: maxEdge, height: maxEdge }
  }
  const longest = Math.max(width, height)
  if (longest <= maxEdge) {
    return { width: Math.round(width), height: Math.round(height) }
  }
  const scale = maxEdge / longest
  return { width: Math.max(1, Math.round(width * scale)), height: Math.max(1, Math.round(height * scale)) }
}

/**
 * canvasToBlob 把一个画布导出为 PNG Blob。
 * 导出失败时返回 null 而不是抛错，调用方据此跳过该部分。
 */
export function canvasToBlob(canvas: HTMLCanvasElement): Promise<Blob | null> {
  return new Promise((resolve) => {
    try {
      canvas.toBlob((blob) => resolve(blob), CANVAS_EXPORT_MIME)
    } catch {
      resolve(null)
    }
  })
}

/**
 * buildMaskBlob 从遮罩画布生成 OpenAI 约定的遮罩图片。
 *
 * 输入画布里，笔刷覆盖处画的是不透明黑色，其余是透明。
 * 这里把「透明」翻转为「要重绘」：不透明处变成全透明（alpha=0），
 * 透明处变成全白不透明（alpha=255）。这样即使上游按 alpha 读，
 * 未涂抹区域也不会被误判为重绘目标。
 *
 * 画布完全空白（没有任何笔刷像素）时返回 null，让调用方省略 mask 字段。
 */
export async function buildMaskBlob(maskCanvas: HTMLCanvasElement): Promise<Blob | null> {
  const width = maskCanvas.width
  const height = maskCanvas.height
  if (width <= 0 || height <= 0) {
    return null
  }

  let source: ImageData
  try {
    source = maskCanvas.getContext('2d', { willReadFrequently: true })?.getImageData(0, 0, width, height) ?? (() => {
      throw new Error('missing 2d context')
    })()
  } catch {
    return null
  }

  const pixels = source.data
  let painted = 0
  for (let i = 3; i < pixels.length; i += 4) {
    if (pixels[i] > 0) {
      painted++
    }
  }
  // 没有任何笔刷像素，就不需要遮罩：否则空遮罩会被上游当成「整张都要重绘」。
  if (painted === 0) {
    return null
  }

  const output = new ImageData(width, height)
  const out = output.data
  for (let i = 0; i < pixels.length; i += 4) {
    // 有笔刷像素 → alpha=0（要重绘）；无笔刷像素 → alpha=255（保留原图）。
    const alpha = pixels[i + 3] > 0 ? 0 : 255
    out[i] = 255
    out[i + 1] = 255
    out[i + 2] = 255
    out[i + 3] = alpha
  }

  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    return null
  }
  ctx.putImageData(output, 0, 0)
  return canvasToBlob(canvas)
}

/**
 * buildComposite 把源画布和遮罩画布合成为一次提交的产物。
 *
 * 合成图按 fitEdge 约束边长后绘制；遮罩画布用同样的尺寸缩放，
 * 保证两个图的上游尺寸一致（否则遮罩对不上）。
 */
export async function buildComposite(
  sourceCanvas: HTMLCanvasElement,
  maskCanvas: HTMLCanvasElement | null,
  options: CompositeOptions,
): Promise<CompositeResult | null> {
  const target = fitEdge(options.width, options.height, MAX_CANVAS_EXPORT_EDGE)

  const canvas = document.createElement('canvas')
  canvas.width = target.width
  canvas.height = target.height
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    return null
  }
  ctx.fillStyle = options.background
  ctx.fillRect(0, 0, target.width, target.height)
  ctx.drawImage(sourceCanvas, 0, 0, target.width, target.height)

  const blob = await canvasToBlob(canvas)
  if (!blob) {
    return null
  }

  let mask: Blob | null = null
  if (maskCanvas) {
    const scaled = document.createElement('canvas')
    scaled.width = target.width
    scaled.height = target.height
    const scaledCtx = scaled.getContext('2d')
    if (scaledCtx) {
      scaledCtx.drawImage(maskCanvas, 0, 0, target.width, target.height)
      mask = await buildMaskBlob(scaled)
    }
  }

  return { blob, mask, width: target.width, height: target.height }
}
