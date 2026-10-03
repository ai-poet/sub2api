/**
 * 站内信（fork 本地功能）：分类 / 来源的展示辅助。
 *
 * 标签一律通过静态字面量 t('...') 取——i18n 完整性检查只扫描字面量键，
 * 拼接键（t(`siteMessages.categories.${c}`)）会漏检。
 */

type TranslateFn = (key: string, ...args: any[]) => string

export function siteMessageCategoryLabel(t: TranslateFn, category: string): string {
  switch (category) {
    case 'security':
      return t('siteMessages.categories.security')
    case 'admin':
      return t('siteMessages.categories.admin')
    case 'system':
      return t('siteMessages.categories.system')
    default:
      return category
  }
}

export function siteMessageFromLabel(t: TranslateFn, from: string): string {
  switch (from) {
    case 'staff':
      return t('siteMessages.from.staff')
    case 'system':
      return t('siteMessages.from.system')
    default:
      return from
  }
}
