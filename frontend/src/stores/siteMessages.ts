import { defineStore } from 'pinia'
import { ref } from 'vue'
import siteMessagesAPI, { type SiteMessage } from '@/api/siteMessages'
import { useAuthStore } from './auth'

// 站内信（fork 本地功能）：顶栏角标 + 收件箱列表 + 默认自动弹窗。
//
// 每分钟轮询未读数（页面隐藏时跳过）。登录后第一次拉到未读、或之后未读数上涨时，
// 拉最多 POPUP_BATCH 条未读按时间从旧到新排进弹窗队列；同一会话里弹过的 id 记在
// sessionStorage，不会重复弹。弹窗点「知道了」即标为已读，所以下次登录也不会再弹。
const POLL_INTERVAL_MS = 60 * 1000
const POPUP_BATCH = 10
const INBOX_PAGE_SIZE = 20
export const SITE_MESSAGE_POPPED_KEY_PREFIX = 'sub2api_site_msg_popped:'

function poppedKey(userId: number | undefined): string {
  return `${SITE_MESSAGE_POPPED_KEY_PREFIX}${userId ?? 'anon'}`
}

function readPopped(userId: number | undefined): Set<number> {
  try {
    const raw = sessionStorage.getItem(poppedKey(userId))
    const ids = raw ? (JSON.parse(raw) as unknown) : []
    return new Set(Array.isArray(ids) ? ids.filter((v): v is number => typeof v === 'number') : [])
  } catch {
    return new Set()
  }
}

function writePopped(userId: number | undefined, ids: Set<number>) {
  try {
    // 只留最近的一批，避免长会话里无限增长
    sessionStorage.setItem(poppedKey(userId), JSON.stringify([...ids].slice(-200)))
  } catch {
    // sessionStorage 不可用时退化为只在内存里去重
  }
}

function clearPoppedKeys() {
  try {
    for (let i = sessionStorage.length - 1; i >= 0; i--) {
      const key = sessionStorage.key(i)
      if (key && key.startsWith(SITE_MESSAGE_POPPED_KEY_PREFIX)) sessionStorage.removeItem(key)
    }
  } catch {
    // ignore
  }
}

export const useSiteMessagesStore = defineStore('siteMessages', () => {
  const unreadCount = ref(0)

  // 收件箱列表（顶栏弹窗里用）
  const items = ref<SiteMessage[]>([])
  const total = ref(0)
  const page = ref(1)
  const unreadOnly = ref(false)
  const listLoading = ref(false)
  /** 由弹窗的「查看全部」或顶栏图标打开 */
  const inboxOpen = ref(false)

  // 自动弹窗
  const popupQueue = ref<SiteMessage[]>([])
  const currentPopup = ref<SiteMessage | null>(null)

  let timer: number | null = null
  let lastCount = -1
  let fetching = false
  let popupChecking = false
  let generation = 0
  let listSeq = 0
  let popped = new Set<number>()
  let poppedUser: number | undefined

  function currentUserId(): number | undefined {
    return useAuthStore().user?.id
  }

  function ensurePoppedLoaded() {
    const uid = currentUserId()
    if (poppedUser !== uid) {
      poppedUser = uid
      popped = readPopped(uid)
    }
  }

  async function fetchUnreadCount() {
    const authStore = useAuthStore()
    if (!authStore.isAuthenticated) {
      unreadCount.value = 0
      return
    }
    if (fetching) return
    fetching = true
    const gen = generation
    try {
      const res = await siteMessagesAPI.unreadCount()
      if (gen !== generation) return
      const count = Number(res?.count ?? 0) || 0
      const grew = count > 0 && count > lastCount
      unreadCount.value = count
      lastCount = count
      if (grew) void checkPopups()
    } catch (err) {
      console.error('Failed to fetch site message unread count:', err)
    } finally {
      fetching = false
    }
  }

  /** 拉未读并把本会话还没弹过的排进队列（从旧到新）。 */
  async function checkPopups() {
    if (popupChecking) return
    popupChecking = true
    const gen = generation
    try {
      ensurePoppedLoaded()
      const res = await siteMessagesAPI.list(1, POPUP_BATCH, { unreadOnly: true })
      if (gen !== generation) return
      const queued = new Set(popupQueue.value.map((m) => m.id))
      if (currentPopup.value) queued.add(currentPopup.value.id)
      const fresh = (res?.items ?? [])
        .filter((m) => !m.read_at && !popped.has(m.id) && !queued.has(m.id))
        .reverse()
      if (fresh.length === 0) return
      popupQueue.value.push(...fresh)
      if (!currentPopup.value) showNextPopup()
    } catch (err) {
      console.error('Failed to load site messages for popup:', err)
    } finally {
      popupChecking = false
    }
  }

  function showNextPopup() {
    const next = popupQueue.value.shift() ?? null
    currentPopup.value = next
    if (next) {
      ensurePoppedLoaded()
      popped.add(next.id)
      writePopped(poppedUser, popped)
    }
  }

  function applyRead(id: number) {
    const now = new Date().toISOString()
    let changed = false
    for (const list of [items.value, popupQueue.value]) {
      const hit = list.find((m) => m.id === id)
      if (hit && !hit.read_at) {
        hit.read_at = now
        changed = true
      }
    }
    if (currentPopup.value?.id === id && !currentPopup.value.read_at) {
      currentPopup.value.read_at = now
      changed = true
    }
    return changed
  }

  /** 标记一条已读（乐观更新，失败时下一次轮询会纠正角标）。 */
  async function markRead(id: number) {
    const target =
      items.value.find((m) => m.id === id) ??
      (currentPopup.value?.id === id ? currentPopup.value : undefined) ??
      popupQueue.value.find((m) => m.id === id)
    const wasUnread = target ? !target.read_at : true
    applyRead(id)
    if (wasUnread && unreadCount.value > 0) {
      unreadCount.value -= 1
      lastCount = unreadCount.value
    }
    if (unreadOnly.value && items.value.some((m) => m.id === id)) {
      // 「未读」列表里读过的条目移出去，总数同步减一，免得「加载更多」的分页错位
      items.value = items.value.filter((m) => m.id !== id)
      total.value = Math.max(0, total.value - 1)
    }
    try {
      await siteMessagesAPI.markRead(id)
    } catch (err) {
      console.error('Failed to mark site message as read:', err)
      void fetchUnreadCount()
    }
  }

  async function markAllRead() {
    await siteMessagesAPI.markAllRead()
    const now = new Date().toISOString()
    for (const m of items.value) if (!m.read_at) m.read_at = now
    if (unreadOnly.value) {
      items.value = []
      total.value = 0
    }
    popupQueue.value = []
    currentPopup.value = null
    unreadCount.value = 0
    lastCount = 0
  }

  /** 弹窗「知道了」：标为已读并显示下一条。 */
  async function acknowledgePopup() {
    const current = currentPopup.value
    if (!current) return
    currentPopup.value = null
    void markRead(current.id)
    if (popupQueue.value.length > 0) {
      window.setTimeout(() => showNextPopup(), 200)
    }
  }

  /** 弹窗「查看全部」：关掉弹窗（不标已读）并打开收件箱。 */
  function openInboxFromPopup() {
    popupQueue.value = []
    currentPopup.value = null
    inboxOpen.value = true
  }

  /**
   * 加载收件箱列表。reset=true 时重新从第一页加载（切换全部 / 未读、打开收件箱），
   * 后发的请求覆盖先发的；reset=false 是「加载更多」，加载中时忽略重复点击。
   */
  async function loadPage(reset = false) {
    if (!reset && listLoading.value) return
    const seq = ++listSeq
    const gen = generation
    const nextPage = reset ? 1 : page.value + 1
    listLoading.value = true
    try {
      const res = await siteMessagesAPI.list(nextPage, INBOX_PAGE_SIZE, { unreadOnly: unreadOnly.value })
      if (gen !== generation || seq !== listSeq) return
      const incoming = res?.items ?? []
      items.value = reset ? incoming : [...items.value, ...incoming.filter((m) => !items.value.some((x) => x.id === m.id))]
      total.value = Number(res?.total ?? items.value.length)
      page.value = nextPage
    } finally {
      if (gen === generation && seq === listSeq) listLoading.value = false
    }
  }

  function setUnreadOnly(value: boolean) {
    if (unreadOnly.value === value) return
    unreadOnly.value = value
    return loadPage(true)
  }

  /** 启动轮询（幂等）：立即取一次，之后每分钟刷新；页面不可见时跳过。 */
  function start() {
    if (timer !== null) return
    void fetchUnreadCount()
    timer = window.setInterval(() => {
      if (typeof document !== 'undefined' && document.hidden) return
      void fetchUnreadCount()
    }, POLL_INTERVAL_MS)
  }

  function stop() {
    if (timer !== null) {
      window.clearInterval(timer)
      timer = null
    }
  }

  function reset() {
    stop()
    generation++
    unreadCount.value = 0
    items.value = []
    total.value = 0
    page.value = 1
    unreadOnly.value = false
    listLoading.value = false
    inboxOpen.value = false
    popupQueue.value = []
    currentPopup.value = null
    lastCount = -1
    fetching = false
    popupChecking = false
    popped = new Set()
    poppedUser = undefined
    clearPoppedKeys()
  }

  return {
    unreadCount,
    items,
    total,
    page,
    unreadOnly,
    listLoading,
    inboxOpen,
    popupQueue,
    currentPopup,
    fetchUnreadCount,
    checkPopups,
    showNextPopup,
    markRead,
    markAllRead,
    acknowledgePopup,
    openInboxFromPopup,
    loadPage,
    setUnreadOnly,
    start,
    stop,
    reset
  }
})
