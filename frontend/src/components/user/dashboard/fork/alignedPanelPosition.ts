import { getFloatingPanelPosition, type FloatingPanelOptions, type FloatingPanelPosition } from '@/utils/floatingPanel'

// fork：在上游 getFloatingPanelPosition 上补 TokenRouter 的 align / pinLeftOnMobile 两个选项
// （移植自 TokenRouter utils/floatingPanel.ts）。宽度和上下翻转仍由上游函数计算，这里只重算左缘。

export interface AlignedPanelOptions extends FloatingPanelOptions {
  // 面板左缘对齐触发器左缘；默认 'right'，右缘对齐触发器右缘。
  align?: 'left' | 'right'
  // 窄屏（视口宽低于 mobileBreakpoint）把面板钉到视口左缘；传 false 保持与触发器对齐。
  pinLeftOnMobile?: boolean
}

export const getAlignedPanelPosition = (
  triggerRect: Pick<DOMRect, 'top' | 'right' | 'bottom' | 'left'>,
  viewportWidth: number,
  viewportHeight: number,
  options: AlignedPanelOptions = {},
): FloatingPanelPosition => {
  const position = getFloatingPanelPosition(triggerRect, viewportWidth, viewportHeight, options)
  const viewportPadding = options.viewportPadding ?? 16
  const mobileBreakpoint = options.mobileBreakpoint ?? 768
  // 左对齐与右对齐都夹取在视口内。
  const rawLeft = options.align === 'left' ? triggerRect.left : triggerRect.right - position.width
  const pinLeft = viewportWidth < mobileBreakpoint && (options.pinLeftOnMobile ?? true)
  const left = pinLeft
    ? viewportPadding
    : Math.max(viewportPadding, Math.min(rawLeft, viewportWidth - position.width - viewportPadding))
  return { ...position, left }
}
