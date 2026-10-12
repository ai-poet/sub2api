/**
 * 带 Authorization 头取回同源文件再交给浏览器。
 *
 * 整页导航（location.href / window.open 一个接口地址）带不了请求头，以前只好把登录 token 放进
 * 地址；这里改成先 fetch 成 blob，再用 object URL 下载或打开。
 */

/** 从 Content-Disposition 取文件名，优先 RFC 5987 的 filename*。 */
export function filenameFromDisposition(header: string | null): string {
  if (!header) return '';
  const extended = /filename\*\s*=\s*utf-8''([^;]+)/i.exec(header)?.[1];
  if (extended) {
    try {
      return decodeURIComponent(extended.trim().replace(/^"|"$/g, ''));
    } catch {
      // 编码坏了就退回普通 filename。
    }
  }
  const plain = /filename\s*=\s*(?:"([^"]*)"|([^;]+))/i.exec(header);
  return (plain?.[1] ?? plain?.[2] ?? '').trim();
}

function saveBlob(blob: Blob, filename: string) {
  const href = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = href;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(href), 60_000);
}

/** 下载文件；失败（含网络错误、非 2xx）返回 false。 */
export async function downloadWithAuth(
  url: string,
  headers: Record<string, string>,
  fallbackName: string,
): Promise<boolean> {
  try {
    const res = await fetch(url, { headers });
    if (!res.ok) return false;
    const blob = await res.blob();
    saveBlob(blob, filenameFromDisposition(res.headers.get('content-disposition')) || fallbackName);
    return true;
  } catch {
    return false;
  }
}

/**
 * 在新标签页预览文件。新窗口必须在点击当下同步打开，等 fetch 回来再开会被弹窗拦截；
 * 被拦截时退回下载。
 */
export async function openWithAuth(
  url: string,
  headers: Record<string, string>,
  fallbackName: string,
): Promise<boolean> {
  const win = window.open('', '_blank');
  try {
    const res = await fetch(url, { headers });
    if (!res.ok) {
      win?.close();
      return false;
    }
    const blob = await res.blob();
    if (!win) {
      saveBlob(blob, filenameFromDisposition(res.headers.get('content-disposition')) || fallbackName);
      return true;
    }
    const href = URL.createObjectURL(blob);
    win.location.href = href;
    setTimeout(() => URL.revokeObjectURL(href), 5 * 60_000);
    return true;
  } catch {
    win?.close();
    return false;
  }
}
