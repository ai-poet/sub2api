import { nextTick } from 'vue'
import { useRouter } from 'vue-router'

/**
 * 跳到首页上的某个区块（如 #pricing、#download）：已在首页就平滑滚动，
 * 否则先回首页再滚动。页头、页脚和更新日志页共用。
 */
export function useHomeSectionNav() {
  const router = useRouter()

  function scrollTo(id: string) {
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  async function goToSection(id: string) {
    if (router.currentRoute.value.path === '/home') {
      scrollTo(id)
      return
    }
    await router.push('/home')
    await nextTick()
    window.setTimeout(() => scrollTo(id), 100)
  }

  return { goToSection }
}
