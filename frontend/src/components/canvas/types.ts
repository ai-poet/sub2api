// 画布工作台共享类型。
// 单独成文件：`<script setup>` 不能导出 ES 模块符号，工具类型要被多个组件引用。
export type CanvasTool = 'select' | 'brush' | 'eraser' | 'text'

/** 画布上对象的角色：底图（source）不可选中/删除，其余为标注（annotation）。 */
export type CanvasObjectRole = 'source' | 'annotation'
