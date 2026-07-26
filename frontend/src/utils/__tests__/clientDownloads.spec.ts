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
      },
    ])
  })

  it('prioritizes the preferred platform when both links exist', () => {
    const options = getClientDownloadOptions(
      {
        windowsUrl: 'https://downloads.example.com/windows.exe',
        macosUrl: 'https://downloads.example.com/macos.dmg',
      },
      'macos' as ClientDownloadPlatform,
    )

    expect(options.map((option) => option.id)).toEqual(['macos', 'windows'])
  })
})
