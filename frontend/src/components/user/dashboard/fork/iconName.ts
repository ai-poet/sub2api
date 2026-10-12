import type Icon from '@/components/icons/Icon.vue'

// fork：Icon 组件可用的图标名，供仪表盘各组件的配置数组标注类型。
export type IconName = InstanceType<typeof Icon>['$props']['name']
