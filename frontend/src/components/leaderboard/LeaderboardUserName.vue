<template>
  <div class="min-w-0" :class="center ? 'flex flex-col items-center text-center' : ''">
    <RouterLink
      v-if="usageRoute"
      :to="usageRoute"
      class="block max-w-full truncate hover:text-primary-600 hover:underline dark:hover:text-primary-400"
      :class="nameClass"
      :title="t('leaderboard.users.viewUsage')"
      data-testid="leaderboard-user-link"
    >
      {{ name }}
    </RouterLink>
    <component :is="tag" v-else class="max-w-full truncate" :class="nameClass">{{ name }}</component>
    <p
      v-if="detail"
      class="mt-0.5 max-w-full truncate text-xs font-normal text-gray-500 dark:text-gray-400"
      :title="detail"
      data-testid="leaderboard-user-detail"
    >
      {{ detail }}
    </p>
  </div>
</template>

<script setup lang="ts">
// fork：用户榜的名字。普通用户看到脱敏名字；管理员 / 运维管理员（后端下发了 user_id）看到真实名字，
// 名字链到该用户的调用日志，下面一行是 #ID 与邮箱。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LeaderboardUserItem } from '@/api/leaderboard'

const props = withDefaults(
  defineProps<{
    item: LeaderboardUserItem
    nameClass?: string
    tag?: string
    center?: boolean
  }>(),
  { nameClass: '', tag: 'p', center: false }
)

const { t } = useI18n()

const name = computed(() => props.item.display_name || t('leaderboard.users.anonymous'))

const usageRoute = computed(() =>
  typeof props.item.user_id === 'number' ? { path: '/admin/usage', query: { user_id: String(props.item.user_id) } } : null
)

const detail = computed(() => {
  if (typeof props.item.user_id !== 'number') return ''
  const parts = [`#${props.item.user_id}`]
  // 显示名已经是邮箱时不再重复
  if (props.item.email && props.item.email !== props.item.display_name) parts.push(props.item.email)
  return parts.join(' · ')
})
</script>
