<template>
  <section class="card p-5" data-fingerprint-benchmarks>
    <div class="flex items-start gap-3">
      <Icon name="infoCircle" size="md" class="mt-0.5 shrink-0 text-gray-400" />
      <div>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">
          {{ t('modelStatus.benchmarks.title') }}
        </h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('modelStatus.benchmarks.description') }}
        </p>
      </div>
    </div>

    <div class="mt-4 grid gap-4 md:grid-cols-2">
      <div
        v-for="family in FINGERPRINT_BENCHMARKS"
        :key="family.family"
        :data-benchmark-family="family.family"
        class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800"
      >
        <div class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
          <PlatformIcon :platform="family.family === 'gpt' ? 'openai' : 'anthropic'" size="md" />
          {{ t(`modelStatus.benchmarks.families.${family.family}`) }}
        </div>
        <ul class="mt-3 space-y-4">
          <li v-for="source in family.sources" :key="source.key" :data-benchmark-source="source.key">
            <div class="flex flex-wrap items-center gap-2">
              <span class="badge badge-gray">{{ t(`modelStatus.benchmarks.methods.${source.method}`) }}</span>
              <span class="text-sm font-medium text-gray-900 dark:text-white">
                {{ source.models.map((model) => astraModelLabel(model)).join(' / ') }}
              </span>
            </div>
            <p class="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">
              {{ t(`modelStatus.benchmarks.sources.${source.key}`) }}
            </p>
            <div class="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
              <a
                v-if="source.repo"
                :href="source.repo"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-1 text-primary-600 hover:underline dark:text-primary-400"
                data-benchmark-repo
              >
                <Icon name="externalLink" size="xs" />
                {{ t('modelStatus.benchmarks.repo') }}: {{ repoName(source.repo) }}
              </a>
              <a
                v-if="source.site"
                :href="source.site"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-1 text-primary-600 hover:underline dark:text-primary-400"
              >
                <Icon name="externalLink" size="xs" />
                {{ t('modelStatus.benchmarks.site') }}
              </a>
              <span v-if="source.license" class="text-gray-400 dark:text-gray-500">
                {{ t('modelStatus.benchmarks.license') }}: {{ source.license }}
              </span>
              <span v-if="!source.repo" class="text-gray-400 dark:text-gray-500">
                {{ t('modelStatus.benchmarks.selfImplemented') }}
              </span>
            </div>
          </li>
        </ul>
      </div>
    </div>

    <p class="mt-4 text-xs leading-5 text-gray-500 dark:text-gray-400">
      {{ t('modelStatus.benchmarks.disclaimer') }}
    </p>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { FINGERPRINT_BENCHMARKS, astraModelLabel } from '@/utils/groupStatus'

const { t } = useI18n()

// https://github.com/owner/repo → owner/repo，让开源地址本身直接可读
function repoName(url: string): string {
  return url.replace(/^https?:\/\/github\.com\//, '').replace(/\/$/, '')
}
</script>
