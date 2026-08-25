const OAUTH_AFFILIATE_CODE_KEY = 'oauth_aff_code'
const AFFILIATE_REFERRAL_CODE_KEY = 'affiliate_referral_code'
const AFFILIATE_REFERRAL_TTL_MS = 30 * 24 * 60 * 60 * 1000

interface StoredAffiliateReferralCode {
  code: string
  expiresAt: number
}

export function normalizeOAuthAffiliateCode(value?: unknown): string {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' ? raw.trim() : ''
}

export function pickOAuthAffiliateCode(...values: unknown[]): string {
  for (const value of values) {
    const code = normalizeOAuthAffiliateCode(value)
    if (code) {
      return code
    }
  }
  return ''
}

export function storeAffiliateReferralCode(value?: unknown, now = Date.now()): void {
  if (typeof window === 'undefined') {
    return
  }
  const code = normalizeOAuthAffiliateCode(value)
  if (!code) {
    return
  }
  try {
    const payload: StoredAffiliateReferralCode = {
      code,
      expiresAt: now + AFFILIATE_REFERRAL_TTL_MS
    }
    window.localStorage.setItem(AFFILIATE_REFERRAL_CODE_KEY, JSON.stringify(payload))
  } catch {
    // 忽略浏览器存储异常。
  }
}

export function loadAffiliateReferralCode(now = Date.now()): string {
  if (typeof window === 'undefined') {
    return ''
  }
  try {
    const raw = window.localStorage.getItem(AFFILIATE_REFERRAL_CODE_KEY)
    if (!raw) {
      return ''
    }
    const parsed = JSON.parse(raw) as Partial<StoredAffiliateReferralCode>
    const code = normalizeOAuthAffiliateCode(parsed.code)
    const expiresAt = Number(parsed.expiresAt) || 0
    if (!code || expiresAt <= now) {
      clearAffiliateReferralCode()
      return ''
    }
    return code
  } catch {
    clearAffiliateReferralCode()
    return ''
  }
}

export function clearAffiliateReferralCode(): void {
  if (typeof window === 'undefined') {
    return
  }
  try {
    window.localStorage.removeItem(AFFILIATE_REFERRAL_CODE_KEY)
  } catch {
    // 忽略浏览器存储异常。
  }
}

export function resolveAffiliateReferralCode(...values: unknown[]): string {
  const code = pickOAuthAffiliateCode(...values)
  if (code) {
    storeAffiliateReferralCode(code)
    return code
  }
  return loadAffiliateReferralCode()
}

// 邀请链接存在两种参数形态：上游返利体系的 ?aff= / ?aff_code=，以及本仓库推荐码体系
// 生成的 /register?ref=（见 ReferralView 与后端 GetReferralInfo）。注册页与各 OAuth
// 入口必须统一经由此函数解析，漏掉任何一种参数都会导致邀请关系无法建立。
export function resolveAffiliateReferralCodeFromQuery(
  query: Record<string, unknown>,
  preferredCode?: unknown
): string {
  return resolveAffiliateReferralCode(preferredCode, query.aff, query.aff_code, query.ref)
}

export function storeOAuthAffiliateCode(value?: unknown): void {
  if (typeof window === 'undefined') {
    return
  }
  const code = normalizeOAuthAffiliateCode(value)
  try {
    if (code) {
      window.sessionStorage.setItem(OAUTH_AFFILIATE_CODE_KEY, code)
    } else {
      window.sessionStorage.removeItem(OAUTH_AFFILIATE_CODE_KEY)
    }
  } catch {
    // 忽略浏览器存储异常。
  }
}

export function loadOAuthAffiliateCode(): string {
  if (typeof window === 'undefined') {
    return ''
  }
  try {
    return normalizeOAuthAffiliateCode(window.sessionStorage.getItem(OAUTH_AFFILIATE_CODE_KEY))
  } catch {
    return ''
  }
}

export function clearOAuthAffiliateCode(): void {
  if (typeof window === 'undefined') {
    return
  }
  try {
    window.sessionStorage.removeItem(OAUTH_AFFILIATE_CODE_KEY)
  } catch {
    // 忽略浏览器存储异常。
  }
}

export function clearAllAffiliateReferralCodes(): void {
  clearOAuthAffiliateCode()
  clearAffiliateReferralCode()
}

export function oauthAffiliatePayload(value?: unknown): { aff_code?: string } {
  const code = normalizeOAuthAffiliateCode(value)
  return code ? { aff_code: code } : {}
}
