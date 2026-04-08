import { pickLocaleText, type Locale } from './locale';

export const PAY_CENTER_METADATA_TITLE = 'Payment Center';
export const PAY_CENTER_METADATA_DESCRIPTION = 'Integrated recharge and subscription subsystem';
export const DEFAULT_PRODUCT_NAME_PREFIX = 'Payment';
export const DEFAULT_PRODUCT_NAME_SUFFIX = 'CNY';

export function getPayCenterBadge(locale: Locale): string {
  return pickLocaleText(locale, '支付中心', 'Payment Center');
}

export function getAdminAccessHint(locale: Locale): string {
  return pickLocaleText(locale, '请从主系统正确进入管理页面', 'Please open the admin page from the main system.');
}

export function getRechargeAccessHint(locale: Locale): string {
  return pickLocaleText(locale, '请从主系统正确进入充值页面', 'Please open the recharge page from the main system.');
}

export function getOrdersAccessHint(locale: Locale): string {
  return pickLocaleText(locale, '请从主系统正确进入订单页面', 'Please open the orders page from the main system.');
}

export function getOrdersSessionExpiredHint(locale: Locale): string {
  return pickLocaleText(locale, '登录态已失效，请从主系统重新进入支付页。', 'Session expired. Please re-enter from the main system.');
}

export function getSystemGroupLabel(locale: Locale): string {
  return pickLocaleText(locale, '主系统分组', 'Main System Group');
}

export function getSystemGroupIdLabel(locale: Locale): string {
  return pickLocaleText(locale, '主系统分组 ID', 'Main System Group ID');
}

export function getSystemGroupStatusLabel(locale: Locale): string {
  return pickLocaleText(locale, '主系统状态', 'Main System Status');
}

export function getSystemGroupInfoLabel(locale: Locale): string {
  return pickLocaleText(locale, '主系统分组信息', 'Main System Group Info');
}

export function getSystemGroupReadonlyHint(locale: Locale): string {
  return pickLocaleText(locale, '（只读，来自主系统）', '(read-only, synced from the main system)');
}

export function getSyncSystemGroupsLabel(locale: Locale): string {
  return pickLocaleText(locale, '同步主系统分组', 'Sync Main System Groups');
}

export function getNoSystemGroupsHint(locale: Locale): string {
  return pickLocaleText(locale, '主系统中没有找到分组', 'No groups found in the main system');
}

export function getFetchSystemGroupsFailed(locale: Locale): string {
  return pickLocaleText(locale, '获取主系统分组列表失败', 'Failed to fetch groups from the main system');
}

export function buildDefaultRechargeSubject(amount: string): string {
  return `${DEFAULT_PRODUCT_NAME_PREFIX} ${amount} ${DEFAULT_PRODUCT_NAME_SUFFIX}`.trim();
}

export function buildDefaultSubscriptionSubject(planName: string): string {
  return `Subscription ${planName}`.trim();
}
