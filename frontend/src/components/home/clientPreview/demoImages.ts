// 画图演示用的照片，外链 Unsplash 的图片 CDN（Unsplash License，可免费商用、无需署名；
// 只选了免费图，没有 Unsplash+）。站点 CSP 的 img-src 允许 https:。
// 要换成自家存储时只改这里。来源：
// - cat      https://unsplash.com/photos/xLz5IyUyQM0  SK ZHAO
// - sunset   https://unsplash.com/photos/jWBQiZSQKkY  Merve Kalafat Yılmaz（Kodak Gold 200 胶片）
// - mountain https://unsplash.com/photos/WaYexRfsZbs  Eugene Ga
// - city     https://unsplash.com/photos/Y8r-9RMmhl0  Darien Attridge

export type DemoImageKey = 'cat' | 'sunset' | 'mountain' | 'city'

// 卡片最宽 168px，按两倍密度取 400px 的方图，auto=format 让支持的浏览器拿 WebP / AVIF
const PARAMS = 'w=400&h=400&fit=crop&q=70&auto=format'

function unsplash(photo: string): string {
  return `https://images.unsplash.com/${photo}?${PARAMS}`
}

export const DEMO_IMAGES: Record<DemoImageKey, string> = {
  cat: unsplash('photo-1715872125689-fdab3e3c11e2'),
  sunset: unsplash('photo-1760532889970-ba83774c67c4'),
  mountain: unsplash('photo-1639922957725-0c6ffd44a078'),
  city: unsplash('photo-1759273621970-79e048b38cd2'),
}

/** 在空闲时预取这几张图，轮到画图那一幕时已经在缓存里 */
export function preloadDemoImages(): void {
  if (typeof Image === 'undefined') return
  const run = () => {
    for (const src of Object.values(DEMO_IMAGES)) {
      const image = new Image()
      image.decoding = 'async'
      image.referrerPolicy = 'no-referrer'
      image.src = src
    }
  }
  if (typeof window !== 'undefined' && typeof window.requestIdleCallback === 'function') {
    window.requestIdleCallback(run)
  } else {
    setTimeout(run, 1500)
  }
}
