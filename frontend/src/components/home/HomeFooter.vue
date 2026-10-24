<template>
  <footer class="relative z-10 overflow-hidden bg-primary-600 text-white dark:bg-primary-900">
    <div class="mx-auto max-w-[1200px] px-4 pt-16 md:px-6 md:pt-20">
      <!-- fork：第四列「联系客服」 -->
      <div class="grid gap-12 md:grid-cols-[1.4fr_1fr_1fr_1fr]">
        <div>
          <div class="flex items-center gap-2.5">
            <HomeBrandMark
              :site-name="siteName"
              :site-logo="siteLogo"
              tone="light"
              class="h-9 w-9 rounded-lg text-base"
            />
            <span class="text-xl font-extrabold tracking-[-0.03em]">{{ siteName }}</span>
          </div>
          <p class="mt-4 max-w-[22rem] text-sm leading-6 text-white/75">
            {{ hasClientDownloads ? t('home.landing.footer.clientTagline') : t('home.landing.footer.apiTagline') }}
          </p>
        </div>

        <nav>
          <p class="text-xs font-bold uppercase tracking-[0.2em]">{{ t('home.landing.footer.product') }}</p>
          <ul class="mt-5 space-y-3 text-sm font-semibold uppercase tracking-[0.06em] text-white/80">
            <li v-if="hasClientDownloads">
              <a href="#download" :class="linkClass" @click.prevent="goToSection('download')">
                {{ t('home.landing.footer.download') }}
              </a>
            </li>
            <li>
              <!-- fork：价格直接打开模型广场 -->
              <router-link to="/models" data-test="footer-pricing" :class="linkClass">
                {{ t('home.landing.footer.pricing') }}
              </router-link>
            </li>
            <li v-if="hasClientDownloads">
              <router-link to="/changelog" data-test="footer-changelog" :class="linkClass">
                {{ t('home.landing.footer.changelog') }}
              </router-link>
            </li>
            <li v-if="docUrl">
              <a :href="docUrl" target="_blank" rel="noopener noreferrer" :class="linkClass">
                {{ t('home.landing.footer.docs') }}
              </a>
            </li>
          </ul>
        </nav>

        <nav>
          <p class="text-xs font-bold uppercase tracking-[0.2em]">{{ t('home.landing.footer.support') }}</p>
          <ul class="mt-5 space-y-3 text-sm font-semibold uppercase tracking-[0.06em] text-white/80">
            <li>
              <router-link :to="consolePath" :class="linkClass">{{ t('home.landing.footer.console') }}</router-link>
            </li>
            <li>
              <router-link to="/privacy" :class="linkClass">{{ t('home.footer.privacy') }}</router-link>
            </li>
            <li>
              <router-link to="/terms" :class="linkClass">{{ t('home.footer.terms') }}</router-link>
            </li>
            <!-- fork：使用规范 -->
            <li>
              <router-link to="/acceptable-use" data-test="footer-aup" :class="linkClass">
                {{ t('home.footer.aup') }}
              </router-link>
            </li>
          </ul>
        </nav>

        <!-- fork：页脚公示客服联系方式（在线工单 + 后台客服联系方式 + 交流群二维码） -->
        <div data-test="footer-contact-column">
          <p class="text-xs font-bold uppercase tracking-[0.2em]">{{ t('home.footer.contact') }}</p>
          <ul class="mt-5 space-y-3 text-sm font-semibold uppercase tracking-[0.06em] text-white/80">
            <li>
              <router-link to="/tickets" data-test="footer-support" :class="linkClass">
                {{ t('home.footer.onlineSupport') }}
              </router-link>
            </li>
            <li v-if="contactInfo" class="break-all normal-case tracking-normal" data-test="footer-contact">
              {{ t('home.legal.contactLabel') }}<SupportContactLink :value="contactInfo" :class="linkClass" />
            </li>
          </ul>
          <SupportCommunityQRCode class="mt-5 text-white/75" />
        </div>
      </div>

      <div class="mt-14 border-t border-white/25 pt-6 text-xs font-semibold uppercase tracking-[0.14em] text-white/70">
        &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
      </div>
    </div>

    <!-- 页脚底部的超大字标，只作装饰 -->
    <div aria-hidden="true" class="mt-6 select-none overflow-hidden">
      <p
        class="mx-auto max-w-[1200px] translate-y-[22%] whitespace-nowrap px-4 font-black leading-[0.85] tracking-[-0.06em] text-white/15 md:px-6"
        :style="{ fontSize: wordmarkFontSize }"
      >
        {{ siteName }}
      </p>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeBrandMark from '@/components/home/HomeBrandMark.vue'
import { useHomeSectionNav } from '@/components/home/useHomeSectionNav'
import SupportContactLink from '@/components/common/SupportContactLink.vue'
import SupportCommunityQRCode from '@/components/legal/SupportCommunityQRCode.vue'
import { useAppStore } from '@/stores/app'

const props = withDefaults(
  defineProps<{
    siteName: string
    siteLogo?: string
    docUrl: string
    currentYear: number
    hasClientDownloads?: boolean
    consolePath?: string
  }>(),
  {
    siteLogo: '',
    hasClientDownloads: false,
    consolePath: '/login',
  },
)

const { t } = useI18n()
const { goToSection } = useHomeSectionNav()
const appStore = useAppStore()

// 管理后台「客服联系方式」；没配置时页脚靠在线客服（工单）和交流群二维码公示联系方式
const contactInfo = computed(() => appStore.contactInfo.trim())

const linkClass = 'transition hover:text-white hover:underline hover:underline-offset-4'

// 超大字标按站点名长度定字号，让整个名字刚好铺满内容区宽度（1200px 减去两侧留白）。
// 中日韩字符约占 1em，其余字符在这个字重和字距下约占 0.58em。
const wordmarkFontSize = computed(() => {
  const ems = [...props.siteName].reduce(
    (sum, char) => sum + (/[\u2E80-\u9FFF\uAC00-\uD7AF\uFF00-\uFFEF]/.test(char) ? 1 : 0.58),
    0,
  )
  const width = Math.max(ems, 3)
  return `min(${Math.floor(1152 / width)}px, calc((100vw - 2rem) / ${width.toFixed(2)}))`
})
</script>
