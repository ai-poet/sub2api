import { describe, expect, it } from 'vitest'

import {
  detectPreferredClientPlatform,
  getClientDownloadOptions,
  type ClientDownloadPlatform,
} from '@/utils/clientDownloads'

function createNavigator(platform: string, userAgent = platform) {
  return {
    platform,
    userAgent,
    userAgentData: { platform },
  } as Navigator & { userAgentData?: { platform?: string } }
}

describe('clientDownloads', () => {
  it('detects Windows from navigator platform signals', () => {
    expect(detectPreferredClientPlatform(createNavigator('Windows'))).toBe('windows')
  })

  it('detects macOS from navigator platform signals', () => {
    expect(detectPreferredClientPlatform(createNavigator('macOS'))).toBe('macos')
  })

  it('defaults Linux and unknown platforms to Windows', () => {
    expect(detectPreferredClientPlatform(createNavigator('Linux x86_64'))).toBe('windows')
    expect(detectPreferredClientPlatform(undefined)).toBe('windows')
  })

  it('returns only configured platform links and trims URLs', () => {
    expect(
      getClientDownloadOptions({
        windowsUrl: ' https://downloads.example.com/windows.exe ',
        macosUrl: '',
      }),
    ).toEqual([
      {
        id: 'windows',
        name: 'Windows',
        url: 'https://downloads.example.com/windows.exe',
        type: 'download',
      },
    ])
  })

  it('prioritizes the preferred platform when both links exist', () => {
    const options = getClientDownloadOptions(
      {
        windowsUrl: 'https://downloads.example.com/windows.exe',
        macosUrl: 'curl -fsSL https://example.com/install.sh | bash',
      },
      'macos' as ClientDownloadPlatform,
    )

    expect(options.map((option) => option.id)).toEqual(['macos', 'windows'])
    expect(options[0].type).toBe('command')
    expect(options[1].type).toBe('download')
  })
})
