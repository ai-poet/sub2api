<template>
  <BaseDialog :show="show" :title="t('admin.userGroupRates.addDialog.title')" width="normal" @close="handleClose">
    <div class="space-y-5">
      <!-- 用户 -->
      <div>
        <label class="input-label">{{ t('admin.userGroupRates.addDialog.user') }}</label>
        <div
          v-if="selectedUser"
          class="flex items-center justify-between rounded-lg border border-gray-200 px-3 py-2 dark:border-dark-600"
          data-test="ugr-add-selected-user"
        >
          <div class="min-w-0">
            <div class="truncate font-medium text-gray-900 dark:text-white">{{ selectedUser.email }}</div>
            <div class="truncate text-xs text-gray-500 dark:text-gray-400">
              #{{ selectedUser.id }}<template v-if="selectedUser.username"> · {{ selectedUser.username }}</template>
            </div>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" @click="clearUser">
            {{ t('admin.userGroupRates.addDialog.changeUser') }}
          </button>
        </div>
        <div v-else class="relative">
          <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            v-model="userKeyword"
            type="text"
            class="input pl-10"
            :placeholder="t('admin.userGroupRates.addDialog.userSearch')"
            data-test="ugr-add-user-search"
            @input="handleUserSearch"
          />
          <div
            v-if="userKeyword.trim()"
            class="absolute z-10 mt-1 max-h-56 w-full overflow-auto rounded-lg border border-gray-200 bg-white shadow-lg dark:border-dark-700 dark:bg-dark-800"
          >
            <div v-if="userLoading" class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</div>
            <div v-else-if="userResults.length === 0" class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.userGroupRates.addDialog.noUsers') }}
            </div>
            <button
              v-for="user in userResults"
              :key="user.id"
              type="button"
              class="w-full px-4 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700"
              data-test="ugr-add-user-option"
              @click="selectUser(user)"
            >
              <span class="font-medium text-gray-900 dark:text-white">{{ user.email }}</span>
              <span class="ml-2 text-xs text-gray-500 dark:text-gray-400">#{{ user.id }}<template v-if="user.username"> · {{ user.username }}</template></span>
            </button>
          </div>
        </div>
      </div>

      <!-- 分组 -->
      <div>
        <label class="input-label">{{ t('admin.userGroupRates.addDialog.groups') }}</label>
        <p class="mb-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.userGroupRates.addDialog.groupsHint') }}</p>
        <div class="max-h-64 space-y-1 overflow-auto rounded-lg border border-gray-200 p-2 dark:border-dark-600">
          <label
            v-for="group in eligibleGroups"
            :key="group.id"
            class="flex cursor-pointer items-center gap-3 rounded-md px-2 py-1.5 hover:bg-gray-50 dark:hover:bg-dark-700"
          >
            <input
              v-model="selectedGroupIds"
              type="checkbox"
              :value="group.id"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
              data-test="ugr-add-group"
            />
            <GroupBadge :name="group.name" :platform="group.platform" :rate-multiplier="group.rate_multiplier" />
            <span v-if="existingRate(group.id) !== null" class="ml-auto text-xs text-amber-600 dark:text-amber-400">
              {{ t('admin.userGroupRates.addDialog.existing', { rate: formatRate(existingRate(group.id)) }) }}
            </span>
          </label>
        </div>
      </div>

      <!-- 倍率 -->
      <div>
        <label class="input-label">{{ t('admin.userGroupRates.addDialog.rate') }}</label>
        <input
          v-model.number="rate"
          type="number"
          step="0.0001"
          min="0.0001"
          class="input w-full sm:w-48"
          :placeholder="t('admin.userGroupRates.addDialog.ratePlaceholder')"
          data-test="ugr-add-rate"
        />
      </div>

      <p v-if="validationMessage" class="text-sm text-red-600 dark:text-red-400">{{ validationMessage }}</p>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="submitting" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="submitting"
          data-test="ugr-add-submit"
          @click="submit"
        >
          {{ t('admin.userGroupRates.addDialog.submit') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
// fork：专属倍率用户管理——给某个用户新增（或覆盖）一个或多个分组的专属倍率。
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AdminUser } from '@/types'
import type { UserGroupRateBatchItem, UserGroupRateGroupOption } from '@/api/admin/userGroupRates'
import { formatRate, isValidRate, roundRate } from '@/utils/userGroupRates'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  show: boolean
  groups: UserGroupRateGroupOption[]
  submitting?: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [items: UserGroupRateBatchItem[]]
}>()

const { t } = useI18n()

const userKeyword = ref('')
const userResults = ref<AdminUser[]>([])
const userLoading = ref(false)
const selectedUser = ref<AdminUser | null>(null)
const selectedGroupIds = ref<number[]>([])
const rate = ref<number | ''>('')
const validationMessage = ref('')

// 与用户管理的分组弹窗一致：只给启用中的按量分组设专属倍率
const eligibleGroups = computed(() =>
  props.groups.filter((g) => g.subscription_type === 'standard' && g.status === 'active')
)

function existingRate(groupId: number): number | null {
  const rates = selectedUser.value?.group_rates
  const value = rates?.[groupId]
  return typeof value === 'number' ? value : null
}

watch(
  () => props.show,
  (open) => {
    if (!open) return
    userKeyword.value = ''
    userResults.value = []
    selectedUser.value = null
    selectedGroupIds.value = []
    rate.value = ''
    validationMessage.value = ''
  }
)

let searchTimer: ReturnType<typeof setTimeout> | undefined
let searchSeq = 0

function handleUserSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  const keyword = userKeyword.value.trim()
  if (!keyword) {
    userResults.value = []
    return
  }
  userLoading.value = true
  searchTimer = setTimeout(async () => {
    const seq = ++searchSeq
    try {
      const data = await adminAPI.users.list(1, 10, { search: keyword })
      if (seq === searchSeq) userResults.value = data.items
    } catch {
      if (seq === searchSeq) userResults.value = []
    } finally {
      if (seq === searchSeq) userLoading.value = false
    }
  }, 300)
}

function selectUser(user: AdminUser) {
  selectedUser.value = user
  userKeyword.value = ''
  userResults.value = []
  validationMessage.value = ''
}

function clearUser() {
  selectedUser.value = null
}

function submit() {
  if (!selectedUser.value) {
    validationMessage.value = t('admin.userGroupRates.addDialog.selectUser')
    return
  }
  if (selectedGroupIds.value.length === 0) {
    validationMessage.value = t('admin.userGroupRates.addDialog.selectGroups')
    return
  }
  if (!isValidRate(rate.value)) {
    validationMessage.value = t('admin.userGroupRates.invalidRate')
    return
  }
  validationMessage.value = ''
  const value = roundRate(rate.value)
  const userId = selectedUser.value.id
  emit(
    'submit',
    selectedGroupIds.value.map((groupId) => ({ user_id: userId, group_id: groupId, rate_multiplier: value }))
  )
}

function handleClose() {
  if (props.submitting) return
  emit('close')
}

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>
