import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import SettingsView from '../SettingsView.vue'

const {
  getSettings,
  updateSettings,
  getAllGroups,
  getAdminApiKey,
  getOverloadCooldownSettings,
  getStreamTimeoutSettings,
  getRectifierSettings,
  getBetaPolicySettings,
  fetchPublicSettings,
  fetchAdminSettings,
  showError,
  showSuccess
} = vi.hoisted(() => ({
  getSettings: vi.fn(),
  updateSettings: vi.fn(),
  getAllGroups: vi.fn(),
  getAdminApiKey: vi.fn(),
  getOverloadCooldownSettings: vi.fn(),
  getStreamTimeoutSettings: vi.fn(),
  getRectifierSettings: vi.fn(),
  getBetaPolicySettings: vi.fn(),
  fetchPublicSettings: vi.fn(),
  fetchAdminSettings: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api', () => ({
  adminAPI: {
    settings: {
      getSettings,
      updateSettings,
      testSmtpConnection: vi.fn(),
      sendTestEmail: vi.fn(),
      getAdminApiKey,
      regenerateAdminApiKey: vi.fn(),
      deleteAdminApiKey: vi.fn(),
      getOverloadCooldownSettings,
      updateOverloadCooldownSettings: vi.fn(),
      getStreamTimeoutSettings,
      updateStreamTimeoutSettings: vi.fn(),
      getRectifierSettings,
      updateRectifierSettings: vi.fn(),
      getBetaPolicySettings,
      updateBetaPolicySettings: vi.fn()
    },
    groups: {
      getAll: getAllGroups
    }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    fetchPublicSettings,
    showError,
    showSuccess
  })
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    fetch: fetchAdminSettings
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function createSettings(overrides: Record<string, unknown> = {}) {
  return {
    registration_enabled: true,
    email_verify_enabled: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: true,
    password_reset_enabled: false,
    frontend_url: '',
    invitation_code_enabled: false,
    totp_enabled: false,
    totp_encryption_key_configured: false,
    default_balance: 0,
    default_concurrency: 1,
    default_subscriptions: [],
    site_name: 'Sub2API',
    site_logo: '',
    site_subtitle: '',
    api_base_url: '',
    contact_info: '',
    doc_url: '',
    home_content: '',
    hide_ccs_import_button: false,
    purchase_subscription_enabled: false,
    purchase_subscription_url: '',
    purchase_subscription_open_mode: 'iframe',
    client_download_windows_url: 'https://downloads.example.com/windows.exe',
    client_download_macos_url: 'https://downloads.example.com/macos.dmg',
    backend_mode_enabled: false,
    group_status_enabled: false,
    custom_menu_items: [],
    custom_endpoints: [],
    smtp_host: '',
    smtp_port: 587,
    smtp_username: '',
    smtp_password_configured: false,
    smtp_from_email: '',
    smtp_from_name: '',
    smtp_use_tls: true,
    turnstile_enabled: false,
    turnstile_site_key: '',
    turnstile_secret_key_configured: false,
    linuxdo_connect_enabled: false,
    linuxdo_connect_client_id: '',
    linuxdo_connect_client_secret_configured: false,
    linuxdo_connect_redirect_url: '',
    github_oauth_enabled: false,
    github_oauth_client_id: '',
    github_oauth_client_secret_configured: false,
    github_oauth_redirect_url: '',
    enable_model_fallback: false,
    fallback_model_anthropic: '',
    fallback_model_openai: '',
    fallback_model_gemini: '',
    fallback_model_antigravity: '',
    enable_identity_patch: false,
    identity_patch_prompt: '',
    ops_monitoring_enabled: true,
    ops_realtime_monitoring_enabled: true,
    ops_query_mode_default: 'auto',
    ops_metrics_interval_seconds: 60,
    min_claude_code_version: '',
    max_claude_code_version: '',
    allow_ungrouped_key_scheduling: false,
    enable_fingerprint_unification: true,
    enable_metadata_passthrough: false,
    enable_cch_signing: false,
    ...overrides
  }
}

function mountSettingsView() {
  return mount(SettingsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
        Select: true,
        GroupBadge: true,
        GroupOptionItem: true,
        Toggle: true,
        ImageUpload: true,
        BackupSettings: true
      }
    }
  })
}

describe('admin SettingsView client downloads', () => {
  beforeEach(() => {
    getSettings.mockReset()
    updateSettings.mockReset()
    getAllGroups.mockReset()
    getAdminApiKey.mockReset()
    getOverloadCooldownSettings.mockReset()
    getStreamTimeoutSettings.mockReset()
    getRectifierSettings.mockReset()
    getBetaPolicySettings.mockReset()
    fetchPublicSettings.mockReset()
    fetchAdminSettings.mockReset()
    showError.mockReset()
    showSuccess.mockReset()

    const settings = createSettings()
    getSettings.mockResolvedValue(settings)
    updateSettings.mockImplementation(async (payload) => createSettings(payload))
    getAllGroups.mockResolvedValue([])
    getAdminApiKey.mockResolvedValue({ exists: false, masked_key: '' })
    getOverloadCooldownSettings.mockResolvedValue({ enabled: true, cooldown_minutes: 10 })
    getStreamTimeoutSettings.mockResolvedValue({
      enabled: true,
      action: 'temp_unsched',
      temp_unsched_minutes: 5,
      threshold_count: 3,
      threshold_window_minutes: 10
    })
    getRectifierSettings.mockResolvedValue({
      enabled: true,
      thinking_signature_enabled: true,
      thinking_budget_enabled: true,
      apikey_signature_enabled: false,
      apikey_signature_patterns: []
    })
    getBetaPolicySettings.mockResolvedValue({ rules: [] })
  })

  it('loads client download URLs and submits updated values', async () => {
    const wrapper = mountSettingsView()
    await flushPromises()

    const clientTab = wrapper
      .findAll('button.settings-tab')
      .find((button) => button.text().includes('admin.settings.tabs.client'))
    expect(clientTab).toBeTruthy()
    await clientTab!.trigger('click')

    const windowsInput = wrapper.find<HTMLInputElement>('[data-test="client-download-windows-url"]')
    const macosInput = wrapper.find<HTMLInputElement>('[data-test="client-download-macos-url"]')

    expect(windowsInput.element.value).toBe('https://downloads.example.com/windows.exe')
    expect(macosInput.element.value).toBe('https://downloads.example.com/macos.dmg')

    await windowsInput.setValue('https://cdn.example.com/CheapRouter-Setup.exe')
    await macosInput.setValue('https://cdn.example.com/CheapRouter.dmg')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        client_download_windows_url: 'https://cdn.example.com/CheapRouter-Setup.exe',
        client_download_macos_url: 'https://cdn.example.com/CheapRouter.dmg'
      })
    )
    expect(fetchPublicSettings).toHaveBeenCalledWith(true)
    expect(fetchAdminSettings).toHaveBeenCalledWith(true)
  })
})
