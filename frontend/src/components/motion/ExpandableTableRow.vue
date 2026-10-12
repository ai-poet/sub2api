<template>
  <tr v-if="present">
    <td :colspan="colspan" class="border-0 p-0">
      <Collapse :open="open" appear unmount-on-hide @after-leave="finishLeave">
        <slot />
      </Collapse>
    </td>
  </tr>
</template>

<script setup lang="ts">
// fork：表格里可展开的明细行（移植自 TokenRouter components/common/ExpandableTableRow.vue）。
// 表格是 tr / td 结构，收起动画结束后再移除整行，空行和它的分隔线不会留在表格里。
import { ref, watch } from 'vue'
import Collapse from './Collapse.vue'

const props = defineProps<{ open: boolean; colspan: number }>()
const present = ref(props.open)
watch(() => props.open, (open) => { if (open) present.value = true }, { flush: 'sync' })

function finishLeave() {
  if (!props.open) present.value = false
}
</script>
