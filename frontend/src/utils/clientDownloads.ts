export type ClientDownloadPlatform = 'windows' | 'macos'

export interface ClientDownloadOption {
  id: ClientDownloadPlatform
  name: string
  url: string
}

interface ClientDownloadUrls {
  windowsUrl?: string
  macosUrl?: string
}

const platformRank: Record<ClientDownloadPlatform, number> = {
  windows: 0,
  macos: 1,
}

export function detectPreferredClientPlatform(
  nav: (Navigator & { userAgentData?: { platform?: string } }) | undefined =
    typeof navigator === 'undefined'
      ? undefined
      : (navigator as Navigator & { userAgentData?: { platform?: string } }),
): ClientDownloadPlatform {
  if (!nav) {
    return 'windows'
  }

  const userAgentDataPlatform = nav.userAgentData?.platform || ''
  const platform = userAgentDataPlatform || nav.platform || ''
  const userAgent = nav.userAgent || ''
  const signal = `${platform} ${userAgent}`.toLowerCase()

  if (signal.includes('win')) {
    return 'windows'
  }
  if (signal.includes('mac') || signal.includes('darwin')) {
    return 'macos'
  }
  return 'windows'
}

export function getClientDownloadOptions(
  urls: ClientDownloadUrls,
  preferredPlatform: ClientDownloadPlatform = detectPreferredClientPlatform(),
): ClientDownloadOption[] {
  const windowsUrl = urls.windowsUrl?.trim() || ''
  const macosUrl = urls.macosUrl?.trim() || ''
  const options: ClientDownloadOption[] = []

  if (windowsUrl) {
    options.push({
      id: 'windows',
      name: 'Windows',
      url: windowsUrl,
    })
  }

  if (macosUrl) {
    options.push({
      id: 'macos',
      name: 'macOS',
      url: macosUrl,
    })
  }

  return options.sort((a, b) => {
    if (a.id === preferredPlatform && b.id !== preferredPlatform) return -1
    if (b.id === preferredPlatform && a.id !== preferredPlatform) return 1
    return platformRank[a.id] - platformRank[b.id]
  })
}
