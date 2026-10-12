import type { ComputedRef, InjectionKey } from 'vue'

// fork：筛选面板与字段之间的约定（移植自 TokenRouter components/common/filterPanel.ts）。
// 本项目的 Select 不向字段上报选项，所以只保留面板登记已选条件这一半，字段通过 valueText / clear 接入。

// FilterChip 是筛选面板顶部的一条已选条件，el 用于按字段在面板中的位置排序。
export interface FilterChip {
  label: string
  text: string
  clear: () => void
  el: HTMLElement | null
}

export interface FilterPanelContext {
  register: (chip: ComputedRef<FilterChip | null>) => () => void
}

export const FILTER_PANEL_KEY: InjectionKey<FilterPanelContext> = Symbol('forkFilterPanel')
