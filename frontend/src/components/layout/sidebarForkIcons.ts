// fork：fork 侧栏入口的静态图标（动态图标加载前和加载失败时显示，见 SidebarNavIcon.vue）。
import { defineComponent, h } from 'vue'
import Icon from '@/components/icons/Icon.vue'

export const TrophyIcon = defineComponent({
  name: 'SidebarTrophyIcon',
  render: () => h(Icon, { name: 'trophy', size: 'md' }),
})
