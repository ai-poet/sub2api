// fork：侧栏用到的动态图标（从 TokenRouter components/icons/registry.ts 精简而来）。
// 语义名称在此统一映射，调用方不直接依赖上游文件名。新增图标先选同语义的上游图形，
// 再复制对应的 artwork 文件并在这里登记，来源与许可见 README.md。
import BadgeCheckIcon from './artwork/badge-check'
import BellIcon from './artwork/bell'
import BookOpenIcon from './artwork/book-open'
import BoxIcon from './artwork/box'
import ChartColumnIncreasingIcon from './artwork/chart-column-increasing'
import CircleUserIcon from './artwork/circle-user'
import CreditCardIcon from './artwork/credit-card'
import FolderIcon from './artwork/folder'
import GiftIcon from './artwork/gift'
import GlobeIcon from './artwork/globe'
import HandshakeIcon from './artwork/handshake'
import HistoryIcon from './artwork/history'
import ImagesIcon from './artwork/images'
import KeyIcon from './artwork/key'
import LayersIcon from './artwork/layers'
import LayoutDashboardIcon from './artwork/layout-dashboard'
import MessageCircleIcon from './artwork/message-circle'
import MonitorIcon from './artwork/monitor'
import MoonIcon from './artwork/moon'
import ReceiptIcon from './artwork/receipt'
import ServerIcon from './artwork/server'
import SettingsIcon from './artwork/settings'
import ShieldIcon from './artwork/shield'
import ShieldCheckIcon from './artwork/shield-check'
import SunIcon from './artwork/sun'
import TableIcon from './artwork/table'
import TicketIcon from './artwork/ticket'
import TrophyIcon from './artwork/trophy'
import UsersIcon from './artwork/users'
import WalletIcon from './artwork/wallet'
import type { IconDefinition } from './types'

export const icons = {
  dashboard: LayoutDashboardIcon,
  key: KeyIcon,
  batchImage: ImagesIcon,
  chart: ChartColumnIncreasingIcon,
  layers: LayersIcon,
  creditCard: CreditCardIcon,
  recharge: WalletIcon,
  gift: GiftIcon,
  book: BookOpenIcon,
  chat: MessageCircleIcon,
  modelCatalog: TableIcon,
  ranking: TrophyIcon,
  server: ServerIcon,
  affiliate: HandshakeIcon,
  userCircle: CircleUserIcon,
  monitor: MonitorIcon,
  shieldCheck: ShieldCheckIcon,
  users: UsersIcon,
  badge: BadgeCheckIcon,
  folder: FolderIcon,
  globe: GlobeIcon,
  cube: BoxIcon,
  bell: BellIcon,
  shield: ShieldIcon,
  ticket: TicketIcon,
  order: ReceiptIcon,
  history: HistoryIcon,
  cog: SettingsIcon,
  sun: SunIcon,
  moon: MoonIcon,
} as const satisfies Record<string, IconDefinition>

export type AnimatedIconName = keyof typeof icons
