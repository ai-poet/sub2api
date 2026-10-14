// fork：fork 侧栏入口的静态图标（动态图标加载前和加载失败时显示，见 SidebarNavIcon.vue）。
import { defineComponent, h } from 'vue'
import Icon from '@/components/icons/Icon.vue'

// 专属倍率用户管理入口
export const ExclusiveRateIcon = defineComponent({
  name: 'SidebarExclusiveRateIcon',
  render: () => h(Icon, { name: 'dollar', size: 'md' }),
})

export const TrophyIcon = defineComponent({
  name: 'SidebarTrophyIcon',
  render: () => h(Icon, { name: 'trophy', size: 'md' }),
})
