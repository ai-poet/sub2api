// 递归合并两个 locale 对象：后者补齐/覆盖前者的同名叶子节点。
type Dict = Record<string, unknown>

export function deepMerge<T extends Dict>(base: T, extra: Dict): T {
  const out: Dict = { ...base }
  for (const [key, value] of Object.entries(extra)) {
    const prev = out[key]
    if (
      prev && typeof prev === 'object' && !Array.isArray(prev) &&
      value && typeof value === 'object' && !Array.isArray(value)
    ) {
      out[key] = deepMerge(prev as Dict, value as Dict)
    } else {
      out[key] = value
    }
  }
  return out as T
}
