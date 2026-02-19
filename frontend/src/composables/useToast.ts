import { useAppStore } from '@/stores'

type ToastType = 'success' | 'error' | 'warning' | 'info'

export function useToast() {
  const appStore = useAppStore()

  function showToast(message: string, type: ToastType = 'info', duration?: number): string {
    switch (type) {
      case 'success':
        return appStore.showSuccess(message, duration)
      case 'error':
        return appStore.showError(message, duration)
      case 'warning':
        return appStore.showWarning(message, duration)
      default:
        return appStore.showInfo(message, duration)
    }
  }

  return { showToast }
}

export default useToast
