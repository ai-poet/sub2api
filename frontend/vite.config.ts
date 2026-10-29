import { defineConfig, loadEnv, Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import checker from 'vite-plugin-checker'
import { resolve } from 'path'

// fork：motion-v 及其依赖只服务侧栏动态图标，单独分包懒加载（见 manualChunks）
const FORK_MOTION_ICON_PACKAGES = ['motion-v', 'framer-motion', 'motion-dom', 'motion-utils', 'hey-listen']

function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (character) => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  })[character] || character)
}

function isSafeImageUrl(value: string): boolean {
  const trimmed = value.trim()
  if ((trimmed.startsWith('/') && !trimmed.startsWith('//')) || /^data:image\//i.test(trimmed)) {
    return true
  }
  try {
    const parsed = new URL(trimmed)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:'
  } catch {
    return false
  }
}

function injectBranding(html: string, config: { site_name?: string; site_logo?: string }): string {
  let brandedHtml = html
  const siteName = config.site_name?.trim()
  if (siteName) {
    brandedHtml = brandedHtml.replace(
      /<title>[^<]*<\/title>/i,
      `<title>${escapeHtml(siteName)} - AI API Gateway</title>`,
    )
  }

  const siteLogo = config.site_logo?.trim()
  if (siteLogo && isSafeImageUrl(siteLogo)) {
    brandedHtml = brandedHtml.replace(
      /<link\s+rel=["']icon["'][^>]*>/i,
      `<link rel="icon" href="${escapeHtml(siteLogo)}" />`,
    )
  }
  return brandedHtml
}

/**
 * Vite 插件：开发模式下注入公开配置到 index.html
 * 与生产模式的后端注入行为保持一致，消除闪烁
 */
function injectPublicSettings(backendUrl: string): Plugin {
  return {
    name: 'inject-public-settings',
    apply: 'serve',
    transformIndexHtml: {
      order: 'pre',
      async handler(html) {
        try {
          const response = await fetch(`${backendUrl}/api/v1/settings/public`, {
            signal: AbortSignal.timeout(2000)
          })
          if (response.ok) {
            const data = await response.json()
            if (data.code === 0 && data.data) {
              const script = `<script>window.__APP_CONFIG__=${JSON.stringify(data.data)};</script>`
              return injectBranding(html, data.data).replace('</head>', `${script}\n</head>`)
            }
          }
        } catch (e) {
          console.warn('[vite] 无法获取公开配置，将回退到 API 调用:', (e as Error).message)
        }
        return html
      }
    }
  }
}

export default defineConfig(({ mode }) => {
  // 加载环境变量
  const env = loadEnv(mode, process.cwd(), '')
  const backendUrl = env.VITE_DEV_PROXY_TARGET || 'http://localhost:8080'
  const devPort = Number(env.VITE_DEV_PORT || 3000)

  return {
    plugins: [
      vue(),
      checker({
        vueTsc: true
      }),
      injectPublicSettings(backendUrl)
    ],
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src'),
        // 使用 vue-i18n 运行时版本，避免 CSP unsafe-eval 问题
        'vue-i18n': 'vue-i18n/dist/vue-i18n.runtime.esm-bundler.js'
      }
    },
    define: {
      // 启用 vue-i18n JIT 编译，在 CSP 环境下处理消息插值
      // JIT 编译器生成 AST 对象而非 JS 代码，无需 unsafe-eval
      __INTLIFY_JIT_COMPILATION__: true
    },
    build: {
      outDir: '../backend/internal/web/dist',
      emptyOutDir: true,
      rollupOptions: {
        output: {
          /**
           * 手动分包配置
           * 分离第三方库并按功能合并应用代码，避免循环依赖
           */
          manualChunks(id: string) {
            const normalizedId = id.split('\\').join('/')

            // fork：侧栏动态图标连同 motion-v 单独分包，由 SidebarNavIcon 懒加载，不进首屏
            if (
              FORK_MOTION_ICON_PACKAGES.some((pkg) => normalizedId.includes(`/node_modules/${pkg}/`)) ||
              normalizedId.includes('/src/components/motion-icons/')
            ) {
              return 'feature-motion-icons'
            }

            // fork：认证页背景动画（components/auth/fork/AuthBackground.vue 动态导入）：播放器和两份动画数据各自分包，
            // 只有打开登录 / 注册页时才下载，并且只下载当前主题用到的那份
            if (normalizedId.includes('/node_modules/lottie-web/')) {
              return 'feature-auth-lottie'
            }
            if (normalizedId.includes('/src/assets/lottie/auth-bg-light.json')) {
              return 'feature-auth-bg-light'
            }
            if (normalizedId.includes('/src/assets/lottie/auth-bg-dark.json')) {
              return 'feature-auth-bg-dark'
            }

            // fork：排行榜页面单独分包，路由懒加载时才下载
            if (
              normalizedId.includes('/src/views/user/LeaderboardView.vue') ||
              normalizedId.includes('/src/components/leaderboard/')
            ) {
              return 'feature-leaderboard'
            }

            // fork：画布工作台与 Fabric 只在打开 /canvas 时下载，不进首屏
            if (
              normalizedId.includes('/node_modules/fabric/') ||
              normalizedId.includes('/src/components/canvas/') ||
              normalizedId.includes('/src/views/user/CanvasStudioView.vue') ||
              normalizedId.includes('/src/utils/canvasComposite.ts') ||
              normalizedId.includes('/src/utils/canvasLocalStore.ts')
            ) {
              return 'feature-canvas-studio'
            }

            if (id.includes('node_modules')) {
              // Vue 核心库
              if (
                id.includes('/vue/') ||
                id.includes('/vue-router/') ||
                id.includes('/pinia/') ||
                id.includes('/@vue/')
              ) {
                return 'vendor-vue'
              }

              // UI 工具库（较大，单独分离）
              if (id.includes('/@vueuse/') || id.includes('/xlsx/')) {
                return 'vendor-ui'
              }

              // 图表库
              if (id.includes('/chart.js/') || id.includes('/vue-chartjs/')) {
                return 'vendor-chart'
              }

              // 国际化
              if (id.includes('/vue-i18n/') || id.includes('/@intlify/')) {
                return 'vendor-i18n'
              }

              // 其他小型第三方库合并
              return 'vendor-misc'
            }

            if (
              normalizedId.includes('/src/components/admin/ErrorPassthroughRulesModal.vue') ||
              normalizedId.includes('/src/components/admin/TLSFingerprintProfilesModal.vue')
            ) {
              return 'feature-admin-config-dialogs'
            }

            if (
              normalizedId.includes('/src/composables/useOpenAIOAuth.ts') ||
              normalizedId.includes('/src/composables/useGeminiOAuth.ts') ||
              normalizedId.includes('/src/composables/useAntigravityOAuth.ts') ||
              normalizedId.includes('/src/composables/useAccountOAuth.ts')
            ) {
              return 'feature-admin-oauth'
            }

            // 图表库
            if (id.includes('/chart.js/') || id.includes('/vue-chartjs/')) {
              return 'vendor-chart'
            }

            // 国际化
            if (id.includes('/vue-i18n/') || id.includes('/@intlify/')) {
              return 'vendor-i18n'
            }


            // 其他小型第三方库合并
            return 'vendor-misc'
          }
        }
      }
    },
    server: {
      host: '0.0.0.0',
      port: devPort,
      proxy: {
        '/api': {
          target: backendUrl,
          changeOrigin: true
        },
        '/v1': {
          target: backendUrl,
          changeOrigin: true
        },
        '/setup': {
          target: backendUrl,
          changeOrigin: true
        }
      }
    }
  }
})
